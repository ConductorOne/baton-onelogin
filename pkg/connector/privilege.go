package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
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
	profile := map[string]interface{}{
		"id":   accountPrivilege.Id,
		"name": accountPrivilege.Name,
	}

	appTraitOptions := []rs.AppTraitOption{
		rs.WithAppProfile(profile),
	}

	resource, err := rs.NewAppResource(
		accountPrivilege.Name,
		resourceTypePrivilege,
		accountPrivilege.Id,
		appTraitOptions,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (g *privilegeResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(pt.Token, &v2.ResourceId{ResourceType: resourceTypeRole.Id})
	if err != nil {
		return nil, "", nil, err
	}

	privileges, nextCursor, err := g.client.GetPrivileges(ctx, onelogin.PaginationVars{
		Limit:  ResourcesPageSize,
		Cursor: cursor,
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list groups: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, privilege := range privileges {
		ur, err := privilegeResource(&privilege)

		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, ur)
	}

	return rv, nextPage, nil, nil
}

func (g *privilegeResourceType) Entitlements(_ context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (g *privilegeResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	privilege, err := g.client.GetPrivilegeById(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
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

	return grants, "", nil, nil
}

func privilegeBuilder(client *onelogin.Client) *privilegeResourceType {
	return &privilegeResourceType{
		resourceType: resourceTypePrivilege,
		client:       client,
	}
}
