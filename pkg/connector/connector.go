package connector

import (
	"context"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: annotationsForUserResourceType(),
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
	}
	resourceTypeApp = &v2.ResourceType{
		Id:          "app",
		DisplayName: "App",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_APP,
		},
	}
	resourceTypeGroup = &v2.ResourceType{
		Id:          "group",
		DisplayName: "Group",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
	}
)

type OneLogin struct {
	client *onelogin.Client
}

func (o *OneLogin) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(o.client),
		roleBuilder(o.client),
		appBuilder(o.client),
		groupBuilder(o.client),
	}
}

func (o *OneLogin) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "OneLogin",
		Description: "Connector syncing OneLogin users, roles, groups and applications to Baton.",
	}, nil
}

// Validates that the user has read access to all relevant tables (more information in the readme).
func (o *OneLogin) Validate(ctx context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New returns the OneLogin connector.
func New(ctx context.Context, clientId, clientSecret, subdomain string) (*OneLogin, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	oneLoginClient, err := onelogin.NewClient(ctx, httpClient, clientId, clientSecret, subdomain)
	if err != nil {
		return nil, err
	}

	return &OneLogin{
		client: oneLoginClient,
	}, nil
}
