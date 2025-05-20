package connector

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type userResourceType struct {
	resourceType   *v2.ResourceType
	client         *onelogin.Client
	users          map[int]string
	usersMutex     sync.Mutex
	usersTimestamp time.Time
}

const usersCacheTTL = 5 * time.Minute

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

// refreshUserCache updates the local user cache if TTL expired by fetching users from OneLogin.
func (u *userResourceType) refreshUserCache(ctx context.Context) error {
	u.usersMutex.Lock()
	defer u.usersMutex.Unlock()

	if u.users != nil && time.Since(u.usersTimestamp) < usersCacheTTL {
		return nil
	}

	u.users = make(map[int]string)
	cursor := ""

	for {
		users, nextCursor, err := u.client.GetUsers(ctx, onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		}, "")
		if err != nil {
			return fmt.Errorf("onelogin-connector: failed to load users for cache: %w", err)
		}

		for _, user := range users {
			u.users[user.Id] = user.Email
		}
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	u.usersTimestamp = time.Now()
	return nil
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
func (u *userResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	logger := ctxzap.Extract(ctx)

	if err := u.refreshUserCache(ctx); err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to load user cache: %w", err)
	}

	bag, cursor, err := parsePageToken(pt.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextCursor, err := u.client.GetUsers(ctx, onelogin.PaginationVars{
		Limit:  ResourcesPageSize,
		Cursor: cursor,
	}, "")
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list users: %w", err)
	}

	var resources []*v2.Resource

	for _, user := range users {
		fullUser, err := u.client.GetUserByID(ctx, user.Id)
		if err != nil {
			logger.Error("Error obtaining user", zap.Int("user_id", user.Id), zap.Error(err))
			continue
		}
		user = fullUser

		if user.ManagerId != nil {
			managerId := *user.ManagerId
			if manager, ok := u.users[managerId]; ok {
				user.ManagerEmail = manager
			}
		}

		res, err := parseIntoUserResource(user)
		if err != nil {
			return nil, "", nil, err
		}
		resources = append(resources, res)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, "", nil, err
	}

	return resources, nextPage, nil, nil
}

// Entitlements returns entitlements for a user resource. Not implemented.
func (u *userResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants returns grants for a user resource. Not implemented.
func (u *userResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// userBuilder creates a new instance of the user resource handler.
func userBuilder(client *onelogin.Client) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
	}
}
