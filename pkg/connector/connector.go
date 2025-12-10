package connector

import (
	"context"
	"fmt"

	cfg "github.com/conductorone/baton-onelogin/pkg/config"
	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
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
	resourceTypePrivilegeAction = &v2.ResourceType{
		Id:          "privilege_action",
		DisplayName: "Privilege Action",
	}
	resourceTypePrivilege = &v2.ResourceType{
		Id:          "privilege",
		DisplayName: "Privilege",
	}
)

type OneLogin struct {
	client         *onelogin.Client
	syncPrivileges bool
}

func (o *OneLogin) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncerV2 {
	resources := []connectorbuilder.ResourceSyncerV2{
		userBuilder(o.client),
		roleBuilder(o.client),
		appBuilder(o.client),
		groupBuilder(o.client),
	}

	if o.syncPrivileges {
		resources = append(resources, privilegeActionBuilder(o.client), privilegeBuilder(o.client))
	}

	return resources
}

func (o *OneLogin) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "OneLogin",
		Description: "Connector syncing OneLogin users, roles, groups and applications to Baton.",
	}, nil
}

// Validates that credentials have required scope for the connector.
func (o *OneLogin) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, err := o.client.ValidateScope(ctx, onelogin.PaginationVars{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector: credentials lack required scope for connector validation: %w", err)
	}

	return nil, nil
}

// New returns the OneLogin connector.
func NewConnector(ctx context.Context, clientId, clientSecret, subdomain string, syncPrivileges bool) (*OneLogin, error) {
	oneLoginClient, err := onelogin.NewClient(ctx, clientId, clientSecret, subdomain)
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector: failed to initialize OneLogin client: %w", err)
	}

	return &OneLogin{
		client:         oneLoginClient,
		syncPrivileges: syncPrivileges,
	}, nil
}

// New returns the OneLogin connector configured to sync against the instance URL.
func New(ctx context.Context, config *cfg.Onelogin, opts *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	l := ctxzap.Extract(ctx)

	subdomain, err := sanitizeDomainInput(config.Subdomain)
	if err != nil {
		return nil, nil, fmt.Errorf("error sanitizing subdomain input: %w", err)
	}

	cb, err := NewConnector(
		ctx,
		config.OneloginClientId,
		config.OneloginClientSecret,
		subdomain,
		config.PrivilegesEnabled,
	)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, nil, err
	}

	return cb, nil, nil
}
