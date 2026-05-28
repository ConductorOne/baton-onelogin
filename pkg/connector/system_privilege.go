package connector

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/session"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

// systemPrivilegeNames maps stable slug IDs to the display names returned by
// GET /api/2/users/{id}/privileges. The slug is used as the Baton resource ID;
// the display name is matched case-insensitively against the API response.
//
// Excluded from this list:
//   - "Default"            — basic portal access, not an admin privilege
//   - "Manage role"        — scoped per role; already captured via /api/2/roles/{id}/admins
//   - "Manage group"       — scoped per group
//   - "Manage application" — scoped per app
//   - "Manage accounts"    — reseller-only
//   - "Manage subscriptions" — reseller-only
var systemPrivilegeNames = map[string]string{
	"super_user":                    "Super user",
	"manage_users":                  "Manage users",
	"assume_users":                  "Assume users",
	"assume_users_read_only":        "Assume users (read-only)",
	"help_desk":                     "Help desk",
	"manage_devices":                "Manage devices",
	"manage_shared_app_credentials": "Manage shared app credentials",
}

// systemPrivilegeIDs is the reverse of systemPrivilegeNames: lowercase display
// name → slug ID. Built once so the API-returned name can be resolved to its
// resource ID with a plain map lookup instead of a per-iteration EqualFold.
var systemPrivilegeIDs = func() map[string]string {
	m := make(map[string]string, len(systemPrivilegeNames))
	for id, name := range systemPrivilegeNames {
		m[strings.ToLower(name)] = id
	}
	return m
}()

const systemPrivilegeAssigned = "assigned"

type systemPrivilegeResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (s *systemPrivilegeResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return s.resourceType
}

// List pages through all users, fetching each user's system privileges and
// storing the privilege→userIDs mapping in session storage. The 7 static
// system privilege resources are returned only on the first page (empty token)
// so the SDK creates each resource exactly once. Subsequent pages (non-empty
// token) return no resources and are used solely to accumulate session data.
func (s *systemPrivilegeResourceType) List(ctx context.Context, _ *v2.ResourceId, attr rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	token := attr.PageToken.Token
	bag, cursor, err := parsePageToken(token, &v2.ResourceId{ResourceType: resourceTypeSystemPrivilege.Id})
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to parse page token for system privilege list: %w", err)
	}

	users, nextCursor, err := s.client.GetUsers(ctx, onelogin.PaginationVars{
		Limit:  ResourcesPageSize,
		Cursor: cursor,
	}, "")
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to list users for system privilege cache: %w", err)
	}

	// Load the current privilege→userIDs map from session storage in one call.
	slugs := make([]string, 0, len(systemPrivilegeNames))
	for slug := range systemPrivilegeNames {
		slugs = append(slugs, slug)
	}
	privUsers, err := session.GetManyJSON[[]string](ctx, attr.Session, slugs)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to load system privileges from session: %w", err)
	}
	if privUsers == nil {
		privUsers = make(map[string][]string, len(systemPrivilegeNames))
	}

	// Fetch each user's privileges and append them to the map.
	for _, user := range users {
		userPrivileges, err := s.client.GetUserSystemPrivileges(ctx, strconv.Itoa(user.Id))
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to get privileges for user %d: %w", user.Id, err)
		}
		for _, priv := range userPrivileges {
			slug := systemPrivilegeIDs[strings.ToLower(priv.Name)]
			if slug == "" {
				continue
			}
			privUsers[slug] = append(privUsers[slug], strconv.Itoa(user.Id))
		}
	}

	// Persist the updated map back to session storage in one call.
	if err := session.SetManyJSON(ctx, attr.Session, privUsers); err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to store system privileges in session: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next page token for system privilege list: %w", err)
	}

	// Emit the 7 static resources only on the first page so the SDK sees
	// each resource exactly once regardless of how many user pages there are.
	if token != "" {
		return nil, &rs.SyncOpResults{NextPageToken: nextPage}, nil
	}

	ids := make([]string, 0, len(systemPrivilegeNames))
	for id := range systemPrivilegeNames {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var rv []*v2.Resource
	for _, id := range ids {
		resource, err := rs.NewResource(systemPrivilegeNames[id], resourceTypeSystemPrivilege, id)
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to create system privilege resource %q: %w", id, err)
		}
		rv = append(rv, resource)
	}
	return rv, &rs.SyncOpResults{NextPageToken: nextPage}, nil
}

// Entitlements satisfies ResourceSyncerV2; StaticEntitlements handles the actual work.
func (s *systemPrivilegeResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// StaticEntitlements returns a single template entitlement applied by the SDK to all 7 privilege
// resources. The SDK uses each resource's DisplayName when the template display name is empty.
func (s *systemPrivilegeResourceType) StaticEntitlements(_ context.Context, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			nil,
			systemPrivilegeAssigned,
			entitlement.WithDisplayName("System Privilege Assigned"),
			entitlement.WithDescription("User has this built-in system privilege in OneLogin"),
			entitlement.WithGrantableTo(resourceTypeUser),
		),
	}, nil, nil
}

// Grants reads the privilege→userIDs mapping that List() built in session
// storage and emits one grant per user. No API calls are made here.
func (s *systemPrivilegeResourceType) Grants(ctx context.Context, resource *v2.Resource, attr rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	userIDs, _, err := session.GetJSON[[]string](ctx, attr.Session, resource.Id.Resource)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to read system privilege %q from session: %w", resource.Id.Resource, err)
	}

	var rv []*v2.Grant
	for _, userID := range userIDs {
		rv = append(rv, grant.NewGrant(
			resource,
			systemPrivilegeAssigned,
			&v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     userID,
			},
		))
	}

	return rv, &rs.SyncOpResults{}, nil
}

func systemPrivilegeBuilder(client *onelogin.Client) *systemPrivilegeResourceType {
	return &systemPrivilegeResourceType{
		resourceType: resourceTypeSystemPrivilege,
		client:       client,
	}
}
