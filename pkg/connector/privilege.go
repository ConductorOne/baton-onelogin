package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/types/entitlement"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type privilegeResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (g *privilegeResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return g.resourceType
}

// Create a new connector resource for an OneLogin Group.
func privilegeResource(accountPrivilege *onelogin.AccountPrivilege) (*v2.Resource, error) {
	resource, err := rs.NewResource(
		accountPrivilege.Name,
		resourceTypePrivilege,
		accountPrivilege.Id,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (g *privilegeResourceType) List(ctx context.Context, _ *v2.ResourceId, attr rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	token := attr.PageToken.Token
	bag, cursor, err := parsePageToken(
		token,
		&v2.ResourceId{
			ResourceType: resourceTypePrivilege.Id,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to parse pagination token for privilege list: %w", err)
	}

	privileges, nextCursor, err := g.client.GetPrivileges(ctx, onelogin.PaginationVars{
		Limit:  ResourcesPageSize,
		Cursor: cursor,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to list privileges: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next pagination token for privileges: %w", err)
	}

	var rv []*v2.Resource
	for _, privilege := range privileges {
		ur, err := privilegeResource(&privilege)

		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to create resource for privilege %d: %w", privilege.Id, err)
		}

		rv = append(rv, ur)
	}

	return rv, &rs.SyncOpResults{
		NextPageToken: nextPage,
	}, nil
}

func (g *privilegeResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	ents := []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			resource,
			"assign",
			entitlement.WithDescription("Assign this privilege to a user or role"),
			entitlement.WithDisplayName(fmt.Sprintf("Assign '%s' Privilege", resource.DisplayName)),
			entitlement.WithGrantableTo(resourceTypeUser, resourceTypeRole),
		),
		entitlement.NewAssignmentEntitlement(
			resource,
			"has",
			entitlement.WithDisplayName("Privilege Action "),
			entitlement.WithDescription("Privilege actions that this privilege grants"),
			entitlement.WithGrantableTo(resourceTypeUser, resourceTypeRole),
			entitlement.WithAnnotation(&v2.EntitlementImmutable{}),
		),
	}

	return ents, nil, nil
}

func (g *privilegeResourceType) Grants(ctx context.Context, resource *v2.Resource, attr rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	var bag pagination.Bag

	token := attr.PageToken.Token
	err := bag.Unmarshal(token)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to unmarshal pagination token for privilege %s grants: %w", resource.Id.Resource, err)
	}

	state := bag.Pop()

	if state == nil {
		bag.Push(pagination.PageState{
			Token:          "",
			ResourceTypeID: resourceTypeRole.Id,
		})

		bag.Push(pagination.PageState{
			Token:          "",
			ResourceTypeID: resourceTypeUser.Id,
		})

		privilege, err := g.client.GetPrivilegeById(ctx, resource.Id.Resource)
		if err != nil {
			return nil, nil, err
		}

		grants := make([]*v2.Grant, 0)

		for _, statement := range privilege.Privilege.Statement {
			for _, action := range statement.Action {
				rsPrivilege := &v2.ResourceId{
					ResourceType: resourceTypePrivilegeAction.Id,
					Resource:     action,
				}

				newGrant := grant.NewGrant(resource, "has", rsPrivilege)
				grants = append(grants, newGrant)
			}
		}

		nextToken, err := bag.Marshal()
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to marshal pagination bag for privilege %s action grants: %w", resource.Id.Resource, err)
		}

		return grants, &rs.SyncOpResults{
			NextPageToken: nextToken,
		}, nil
	}

	grants := make([]*v2.Grant, 0)

	switch state.ResourceTypeID {
	case resourceTypeRole.Id:
		rolesResponse, err := g.client.GetPrivilegeAssignableRoles(ctx, resource.Id.Resource, state.Token)
		if err != nil {
			return nil, nil, err
		}

		if rolesResponse.NextLink != "" {
			bag.Push(pagination.PageState{
				Token:          rolesResponse.NextLink,
				ResourceTypeID: resourceTypeRole.Id,
			})
		}

		for _, role := range rolesResponse.Roles {
			grants = append(
				grants,
				grant.NewGrant(
					resource,
					"assign",
					&v2.ResourceId{
						ResourceType: resourceTypeRole.Id,
						Resource:     role,
					},
					grant.WithAnnotation(
						&v2.GrantExpandable{
							EntitlementIds: []string{
								fmt.Sprintf("%s:%s:%s", resourceTypeRole.Id, role, roleAdmin),
								fmt.Sprintf("%s:%s:%s", resourceTypeRole.Id, role, roleMembership),
							},
						},
					),
				),
			)
		}

	case resourceTypeUser.Id:
		usersResponse, err := g.client.GetPrivilegeAssignableUsers(ctx, resource.Id.Resource, state.Token)
		if err != nil {
			return nil, nil, err
		}

		if usersResponse.NextLink != "" {
			bag.Push(pagination.PageState{
				Token:          usersResponse.NextLink,
				ResourceTypeID: resourceTypeUser.Id,
			})
		}

		for _, user := range usersResponse.Users {
			grants = append(
				grants,
				grant.NewGrant(
					resource,
					"assign",
					&v2.ResourceId{
						ResourceType: resourceTypeUser.Id,
						Resource:     user,
					},
				),
			)
		}
	default:
		return nil, nil, fmt.Errorf("onelogin-connector: invalid resource type %s in privilege grants", state.ResourceTypeID)
	}

	nextToken, err := bag.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to marshal pagination bag for privilege %s grants: %w", resource.Id.Resource, err)
	}

	return grants, &rs.SyncOpResults{
		NextPageToken: nextToken,
	}, nil
}

func privilegeBuilder(client *onelogin.Client) *privilegeResourceType {
	return &privilegeResourceType{
		resourceType: resourceTypePrivilege,
		client:       client,
	}
}
