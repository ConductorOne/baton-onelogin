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

type appResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (a *appResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return a.resourceType
}

// Create a new connector resource for an OneLogin App.
func appResource(app *onelogin.App) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"app_id":   app.Id,
		"app_name": app.Name,
	}

	appTraitOptions := []rs.AppTraitOption{
		rs.WithAppProfile(profile),
	}

	resource, err := rs.NewAppResource(
		app.Name,
		resourceTypeApp,
		app.Id,
		appTraitOptions,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *appResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(pt.Token, &v2.ResourceId{ResourceType: resourceTypeApp.Id})
	if err != nil {
		return nil, "", nil, err
	}

	apps, nextCursor, err := a.client.GetApps(
		ctx,
		onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		},
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list apps: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, app := range apps {
		appCopy := app
		ur, err := appResource(&appCopy)

		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, ur)
	}

	return rv, nextPage, nil, nil
}

func (a *appResourceType) Entitlements(_ context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	memberAssignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s App %s", resource.DisplayName, roleMembership)),
		ent.WithDescription(fmt.Sprintf("Access to %s app in OneLogin", resource.DisplayName)),
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

func (a *appResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(token.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	appUsers, nextCursor, err := a.client.GetAppUsers(
		ctx,
		resource.Id.Resource,
		onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		},
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list app users: %w", err)
	}

	var rv []*v2.Grant

	for _, app := range appUsers {
		userResource := &v2.ResourceId{
			ResourceType: resourceTypeUser.Id,
			Resource:     strconv.Itoa(app.Id),
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

func appBuilder(client *onelogin.Client) *appResourceType {
	return &appResourceType{
		resourceType: resourceTypeApp,
		client:       client,
	}
}
