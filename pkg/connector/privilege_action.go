package connector

import (
	"context"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type privilegeActionResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (g *privilegeActionResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return g.resourceType
}

func (g *privilegeActionResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	for _, action := range onelogin.PrivilegeActions {
		resource, err := rs.NewResource(
			action,
			resourceTypePrivilegeAction,
			action,
		)

		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, resource)
	}

	return rv, "", nil, nil
}

func (g *privilegeActionResourceType) Entitlements(_ context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (g *privilegeActionResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func privilegeActionBuilder(client *onelogin.Client) *privilegeActionResourceType {
	return &privilegeActionResourceType{
		resourceType: resourceTypePrivilegeAction,
		client:       client,
	}
}
