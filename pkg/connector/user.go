package connector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/session"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/types/sessions"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (u *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return u.resourceType
}

// buildUserProfile constructs a display name and profile from user details.
func buildUserProfile(displayName, email, firstName, lastName string, managerId *int, managerEmail string, id int) (map[string]interface{}, []rs.UserTraitOption) {
	profile := map[string]interface{}{
		"login":      displayName,
		"user_id":    fmt.Sprintf("%d", id),
		"first_name": firstName,
		"last_name":  lastName,
	}

	if managerId != nil {
		profile["manager_user_id"] = fmt.Sprintf("%d", *managerId)
	}
	if managerEmail != "" {
		profile["manager_email"] = managerEmail
	}

	options := []rs.UserTraitOption{
		rs.WithEmail(email, true),
		rs.WithUserProfile(profile),
	}
	return profile, options
}

// userResource creates a connector resource for a complete OneLogin user object.
func parseIntoUserResource(user *onelogin.User) (*v2.Resource, error) {
	displayName := resolveDisplayName(user)

	_, options := buildUserProfile(
		displayName,
		user.Email,
		user.Firstname,
		user.Lastname,
		user.ManagerId,
		user.ManagerEmail,
		user.Id,
	)

	switch user.Status {
	case 0:
		options = append(options, rs.WithStatus(v2.UserTrait_Status_STATUS_DISABLED))
	case 1:
		options = append(options, rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED))
	case 2:
		options = append(options, rs.WithStatus(v2.UserTrait_Status_STATUS_DELETED))
	default:
		options = append(options, rs.WithStatus(v2.UserTrait_Status_STATUS_UNSPECIFIED))
	}

	return rs.NewUserResource(displayName, resourceTypeUser, user.Id, options)
}

// storeUsersInCache sets the session storage user cache by fetching users from OneLogin.
func (u *userResourceType) storeUsersInCache(ctx context.Context, sessionStorage sessions.SessionStore) error {
	cursor := ""

	for {
		users, nextCursor, err := u.client.GetUsers(ctx, onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		}, "")
		if err != nil {
			return fmt.Errorf("onelogin-connector: failed to load users for cache: %w", err)
		}

		session.SetManyJSON(ctx, sessionStorage, parseJSONCache(users))
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return nil
}

func parseJSONCache(users []*onelogin.User) map[string]string {
	usersMap := make(map[string]string)
	for _, user := range users {
		userIDStr := strconv.Itoa(user.Id)
		usersMap[userIDStr] = user.Email
	}
	return usersMap
}

// resolveDisplayName returns a user's display name based on available fields.
func resolveDisplayName(user *onelogin.User) string {
	if user.Username != "" {
		return user.Username
	}
	name := fmt.Sprintf("%s %s", user.Firstname, user.Lastname)
	if strings.TrimSpace(name) == "" {
		return user.Email
	}
	return name
}

// List retrieves users from OneLogin and returns them as connector resources.
func (u *userResourceType) List(ctx context.Context, _ *v2.ResourceId, attr rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	logger := ctxzap.Extract(ctx)

	token := attr.PageToken.Token
	if token == "" {
		// First page, load cache
		u.storeUsersInCache(ctx, attr.Session)
	}
	bag, cursor, err := parsePageToken(token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to parse pagination token for user list: %w", err)
	}

	users, nextCursor, err := u.client.GetUsers(ctx, onelogin.PaginationVars{
		Limit:  ResourcesPageSize,
		Cursor: cursor,
	}, "")
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to list users: %w", err)
	}

	var resources []*v2.Resource

	for _, user := range users {
		fullUser, err := u.client.GetUserByID(ctx, user.Id)
		if err != nil {
			logger.Error("onelogin-connector: failed to fetch user details during list", zap.Int("user_id", user.Id), zap.Error(err))
			continue
		}
		user = fullUser

		if user.ManagerId != nil {
			managerId := strconv.Itoa(*user.ManagerId)
			manager, found, err := session.GetJSON[string](ctx, attr.Session, managerId)
			if err != nil {
				return nil, nil, fmt.Errorf("onelogin-connector: failed to get manager from cache for user %d: %w", user.Id, err)
			}
			if found {
				user.ManagerEmail = manager
			}
		}

		res, err := parseIntoUserResource(user)
		if err != nil {
			return nil, nil, fmt.Errorf("onelogin-connector: failed to create resource for user %d: %w", user.Id, err)
		}
		resources = append(resources, res)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, nil, fmt.Errorf("onelogin-connector: failed to generate next pagination token for users: %w", err)
	}

	return resources, &rs.SyncOpResults{
		NextPageToken: nextPage,
	}, nil
}

// Entitlements returns entitlements for a user resource. Not implemented.
func (u *userResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// Grants returns grants for a user resource. Not implemented.
func (u *userResourceType) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// userBuilder creates a new instance of the user resource handler.
func userBuilder(client *onelogin.Client) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
	}
}
