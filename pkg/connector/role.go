package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const (
	roleMembership = "member"
	roleAdmin      = "admin"

	adminResourceId = "Admin"
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (r *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return r.resourceType
}

// Create a new connector resource for an OneLogin Role.
func roleResource(ctx context.Context, role *onelogin.Role) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"role_name": role.Name,
		"role_id":   role.Id,
	}

	roleTraitOptions := []rs.RoleTraitOption{
		rs.WithRoleProfile(profile),
	}

	resource, err := rs.NewRoleResource(
		role.Name,
		resourceTypeRole,
		role.Id,
		roleTraitOptions,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (r *roleResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(pt.Token, &v2.ResourceId{ResourceType: resourceTypeRole.Id})
	if err != nil {
		return nil, "", nil, err
	}

	roles, nextCursor, err := r.client.GetRoles(
		ctx,
		onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		},
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list roles: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, role := range roles {
		roleCopy := role
		rr, err := roleResource(ctx, &roleCopy)

		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, rr)
	}

	return rv, nextPage, nil, nil
}

func (r *roleResourceType) Entitlements(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	memberAssignmentOptions := []ent.EntitlementOption{
		// TODO: later add applications
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, roleMembership)),
		ent.WithDescription(fmt.Sprintf("Access to %s role in OneLogin", resource.DisplayName)),
	}

	adminAssignmentOptions := []ent.EntitlementOption{
		// TODO: later add applications
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, roleAdmin)),
		ent.WithDescription(fmt.Sprintf("Admin access to %s role in OneLogin", resource.DisplayName)),
	}

	rv = append(
		rv,
		ent.NewAssignmentEntitlement(
			resource,
			roleMembership,
			memberAssignmentOptions...,
		),
		ent.NewAssignmentEntitlement(
			resource,
			roleAdmin,
			adminAssignmentOptions...,
		),
	)

	return rv, "", nil, nil
}

func (r *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(pt.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Grant
	switch bag.ResourceTypeID() {
	case resourceTypeRole.Id:

		bag.Pop()
		// get role users
		bag.Push(pagination.PageState{
			ResourceTypeID: resourceTypeUser.Id,
		})
		// get role admins
		bag.Push(pagination.PageState{
			ResourceTypeID: adminResourceId,
		})
		// TODO: get role applications

	case resourceTypeUser.Id:
		roleUsers, nextCursor, err := r.client.GetRoleUsers(
			ctx,
			resource.Id.Resource,
			onelogin.PaginationVars{
				Limit:  ResourcesPageSize,
				Cursor: cursor,
			},
		)
		if err != nil {
			return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list users under role %s: %w", resource.Id.Resource, err)
		}

		// for each user, create a grant
		for _, user := range roleUsers {
			userCopy := user
			ur, err := minimalUserResource(ctx, &userCopy)
			if err != nil {
				return nil, "", nil, err
			}

			rv = append(
				rv,
				grant.NewGrant(
					resource,
					roleMembership,
					ur.Id,
				),
			)
		}

		nextPage, err := bag.NextToken(nextCursor)
		if err != nil {
			return nil, "", nil, err
		}

		return rv, nextPage, nil, nil

	case adminResourceId:
		roleAdmins, nextCursor, err := r.client.GetRoleAdmins(
			ctx,
			resource.Id.Resource,
			onelogin.PaginationVars{
				Limit:  ResourcesPageSize,
				Cursor: cursor,
			},
		)
		if err != nil {
			return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list users under role %s: %w", resource.Id.Resource, err)
		}

		// for each user, create a grant
		for _, user := range roleAdmins {
			userCopy := user
			ur, err := minimalUserResource(ctx, &userCopy)
			if err != nil {
				return nil, "", nil, err
			}

			rv = append(
				rv,
				grant.NewGrant(
					resource,
					roleAdmin,
					ur.Id,
				),
			)
		}

		nextPage, err := bag.NextToken(nextCursor)
		if err != nil {
			return nil, "", nil, err
		}

		return rv, nextPage, nil, nil
	default:
		return nil, "", nil, fmt.Errorf("unknown resource type: %s", bag.ResourceTypeID())
	}

	nextPage, err := bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, nextPage, nil, nil
}

func roleBuilder(client *onelogin.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
