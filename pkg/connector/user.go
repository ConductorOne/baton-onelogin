package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *onelogin.Client
}

func (u *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return u.resourceType
}

func minimalUserResource(ctx context.Context, user *onelogin.UserUnderRole) (*v2.Resource, error) {
	var displayName, firstName, lastName string

	// split the name into first and last name
	if user.Name != "" {
		nameParts := strings.Split(user.Name, " ")
		firstName = nameParts[0]
		lastName = strings.Join(nameParts[1:], " ")
	}

	if user.Username == "" {
		if firstName != "" || lastName != "" {
			displayName = fmt.Sprintf("%s %s", firstName, lastName)
		} else {
			displayName = user.Email
		}
	} else {
		displayName = user.Username
	}

	profile := map[string]interface{}{
		"login":      displayName,
		"user_id":    fmt.Sprintf("%d", user.Id),
		"first_name": firstName,
		"last_name":  lastName,
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithEmail(user.Email, true),
		rs.WithUserProfile(profile),
		rs.WithStatus(v2.UserTrait_Status_STATUS_UNSPECIFIED),
	}

	resource, err := rs.NewUserResource(
		displayName,
		resourceTypeUser,
		user.Id,
		userTraitOptions,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// Create a new connector resource for an OneLogin User.
func userResource(ctx context.Context, user *onelogin.User) (*v2.Resource, error) {
	var displayName string
	if user.Username == "" {
		if user.Firstname != "" || user.Lastname != "" {
			displayName = fmt.Sprintf("%s %s", user.Firstname, user.Lastname)
		} else {
			displayName = user.Email
		}
	} else {
		displayName = user.Username
	}

	profile := map[string]interface{}{
		"login":      displayName,
		"user_id":    fmt.Sprintf("%d", user.Id),
		"first_name": user.Firstname,
		"last_name":  user.Lastname,
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithEmail(user.Email, true),
		rs.WithUserProfile(profile),
	}

	// more information regarding the user status can be found here:
	// https://github.com/onelogin/onelogin-go-sdk/blob/develop/pkg/onelogin/models/user.go#L12-L22
	switch user.Status {
	case 0:
		userTraitOptions = append(userTraitOptions, rs.WithStatus(v2.UserTrait_Status_STATUS_DISABLED))
	case 1:
		userTraitOptions = append(userTraitOptions, rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED))
	case 2:
		userTraitOptions = append(userTraitOptions, rs.WithStatus(v2.UserTrait_Status_STATUS_DELETED))
	default:
		userTraitOptions = append(userTraitOptions, rs.WithStatus(v2.UserTrait_Status_STATUS_UNSPECIFIED))
	}

	resource, err := rs.NewUserResource(
		displayName,
		resourceTypeUser,
		user.Id,
		userTraitOptions,
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (u *userResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, cursor, err := parsePageToken(pt.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextCursor, err := u.client.GetUsers(
		ctx,
		onelogin.PaginationVars{
			Limit:  ResourcesPageSize,
			Cursor: cursor,
		},
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("onelogin-connector: failed to list users: %w", err)
	}

	nextPage, err := bag.NextToken(nextCursor)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, user := range users {
		userCopy := user
		ur, err := userResource(ctx, &userCopy)

		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, ur)
	}

	return rv, nextPage, nil, nil
}

func (u *userResourceType) Entitlements(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (u *userResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func userBuilder(client *onelogin.Client) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
	}
}
