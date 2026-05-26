package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func roleResource(role *onelogin.Role) (*v2.Resource, error) {
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

func (r *roleResourceType) List(ctx context.Context, _ *v2.ResourceId, attr rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	token := attr.PageToken.Token
	bag, cursor, err := parsePageToken(token, &v2.ResourceId{ResourceType: resourceTypeRole.Id})
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to parse pagination token for role list: %w", err)
	}

	roles, nextCursor, err := r.client.GetRoles(
		ctx,
		onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to list roles: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next pagination token for roles: %w", err)
	}

	var rv []*v2.Resource
	for _, role := range roles {
		roleCopy := role
		rr, err := roleResource(&roleCopy)

		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to create resource for role %d: %w", roleCopy.Id, err)
		}

		rv = append(rv, rr)
	}

	return rv, &rs.SyncOpResults{
		NextPageToken: nextPage,
	}, nil
}

func (r *roleResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	var rv []*v2.Entitlement

	memberAssignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser, resourceTypeApp),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, roleMembership)),
		ent.WithDescription(fmt.Sprintf("Access to %s role in OneLogin", resource.DisplayName)),
	}

	adminAssignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser, resourceTypeApp),
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

	return rv, nil, nil
}

func (r *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, attr rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	token := attr.PageToken.Token
	bag, cursor, err := parsePageToken(token, resource.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to parse pagination token for grants of role %s: %w", resource.Id.Resource, err)
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
		// get role apps
		bag.Push(pagination.PageState{
			ResourceTypeID: resourceTypeApp.Id,
		})

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
			return nil, nil, fmt.Errorf("onelogin-connector: failed to list users under role %s: %w", resource.Id.Resource, err)
		}

		// for each user, create a grant
		for _, user := range roleUsers {
			userResource := &v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     strconv.Itoa(user.Id),
			}

			rv = append(
				rv,
				grant.NewGrant(
					resource,
					roleMembership,
					userResource,
				),
			)
		}

		nextPage, err := bag.NextToken(nextCursor)
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next pagination token for role %s member grants: %w", resource.Id.Resource, err)
		}

		return rv, &rs.SyncOpResults{
			NextPageToken: nextPage,
		}, nil

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
			return nil, nil, fmt.Errorf("onelogin-connector: failed to list users under role %s: %w", resource.Id.Resource, err)
		}

		// for each user, create a grant
		for _, user := range roleAdmins {
			userResource := &v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     strconv.Itoa(user.Id),
			}

			rv = append(
				rv,
				grant.NewGrant(
					resource,
					roleAdmin,
					userResource,
				),
			)
		}

		nextPage, err := bag.NextToken(nextCursor)
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next pagination token for role %s admin grants: %w", resource.Id.Resource, err)
		}

		return rv, &rs.SyncOpResults{
			NextPageToken: nextPage,
		}, nil

	case resourceTypeApp.Id:
		roleApps, nextCursor, err := r.client.GetRoleApps(
			ctx,
			resource.Id.Resource,
			onelogin.PaginationVars{
				Limit:  ResourcesPageSize,
				Cursor: cursor,
			},
		)
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to list apps under role %s: %w", resource.Id.Resource, err)
		}

		// for each app, create a grant
		for _, app := range roleApps {
			appCopy := app
			ur, err := appResource(&appCopy)
			if err != nil {
				return nil, nil, fmt.Errorf("onelogin-connector: failed to create resource for application %d in role %s grants: %w", appCopy.Id, resource.Id.Resource, err)
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
			return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next pagination token for role %s app grants: %w", resource.Id.Resource, err)
		}

		return rv, &rs.SyncOpResults{
			NextPageToken: nextPage,
		}, nil

	default:
		return nil, nil, fmt.Errorf("onelogin-connector: unknown resource type %s encountered in role grants", bag.ResourceTypeID())
	}

	nextPage, err := bag.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to marshal pagination bag for role %s grants: %w", resource.Id.Resource, err)
	}

	return rv, &rs.SyncOpResults{
		NextPageToken: nextPage,
	}, nil
}

func (r *roleResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"onelogin-connector: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, status.Errorf(codes.InvalidArgument,
			"onelogin-connector: invalid principal type %s for role grant operation",
			principal.Id.ResourceType,
		)
	}

	err := r.client.GrantRole(ctx, entitlement.Resource.Id.Resource, principal.Id.Resource, entitlement.Slug)
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector: failed to grant %s role: %w", entitlement.Slug, err)
	}

	if err := r.client.SyncUserMappings(ctx, principal.Id.Resource); err != nil {
		l.Warn("onelogin-connector: failed to sync user mappings after role grant",
			zap.String("user_id", principal.Id.Resource),
			zap.Error(err),
		)
	}

	return nil, nil
}

func (r *roleResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	entitlement := grant.Entitlement
	principal := grant.Principal

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"onelogin-connector: only users can have role membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, status.Errorf(codes.InvalidArgument,
			"onelogin-connector: invalid principal type %s for role revoke operation",
			principal.Id.ResourceType,
		)
	}

	err := r.client.RevokeRole(ctx, entitlement.Resource.Id.Resource, principal.Id.Resource, entitlement.Slug)
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector failed to revoke %s role: %w", entitlement.Slug, err)
	}

	if err := r.client.SyncUserMappings(ctx, principal.Id.Resource); err != nil {
		l.Warn("onelogin-connector: failed to sync user mappings after role revoke",
			zap.String("user_id", principal.Id.Resource),
			zap.Error(err),
		)
	}

	return nil, nil
}

func roleBuilder(client *onelogin.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
