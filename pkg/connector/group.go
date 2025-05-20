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
)

type groupResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (g *groupResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return g.resourceType
}

// Create a new connector resource for an OneLogin Group.
func groupResource(group *onelogin.Group) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"group_id":   group.Id,
		"group_name": group.Name,
	}

	groupTraitOptions := []rs.GroupTraitOption{
		rs.WithGroupProfile(profile),
	}

	resource, err := rs.NewGroupResource(
		group.Name,
		resourceTypeGroup,
		group.Id,
		groupTraitOptions,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (g *groupResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(pt.Token, &v2.ResourceId{ResourceType: resourceTypeGroup.Id})
	if err != nil {
		return nil, "", nil, err
	}

	groups, nextCursor, err := g.client.GetGroups(
		ctx,
		onelogin.PaginationVars{
			Limit:    ResourcesPageSize,
			V1Cursor: cursor,
		},
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list groups: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, group := range groups {
		groupCopy := group
		ur, err := groupResource(&groupCopy)

		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, ur)
	}

	return rv, nextPage, nil, nil
}

func (g *groupResourceType) Entitlements(_ context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	memberAssignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Group %s", resource.DisplayName, roleMembership)),
		ent.WithDescription(fmt.Sprintf("Access to %s group in OneLogin", resource.DisplayName)),
	}

	rv = append(
		rv,
		ent.NewAssignmentEntitlement(
			resource,
			roleMembership,
			memberAssignmentOptions...,
		),
	)

	return rv, "", nil, nil
}

func (g *groupResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(token.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	users, nextCursor, err := g.client.GetUsers(
		ctx,
		onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		},
		resource.Id.Resource,
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list group users: %w", err)
	}

	var rv []*v2.Grant

	for _, user := range users {
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
		return nil, "", nil, err
	}

	return rv, nextPage, nil, nil
}

func groupBuilder(client *onelogin.Client) *groupResourceType {
	return &groupResourceType{
		resourceType: resourceTypeGroup,
		client:       client,
	}
}
