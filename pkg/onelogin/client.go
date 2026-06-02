package onelogin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const (
	BaseURL = "https://%s.onelogin.com/"

	AuthBaseUrl          = BaseURL + "auth/"
	GenerateTokenBaseURL = AuthBaseUrl + "oauth2/v2/token"

	APIBaseV1URL                       = BaseURL + "api/1/"
	APIBaseURL                         = BaseURL + "api/2/"
	UsersBaseURL                       = APIBaseURL + "users"
	UserBaseUrl                        = UsersBaseURL + "/%s"
	RolesBaseURL                       = APIBaseURL + "roles"
	RoleUsersBaseURL                   = APIBaseURL + "roles/%s/users"
	RoleAdminsBaseURL                  = APIBaseURL + "roles/%s/admins"
	RoleAppsBaseURL                    = APIBaseURL + "roles/%s/apps"
	AppsBaseURL                        = APIBaseURL + "apps"
	AppUsersBaseURL                    = APIBaseURL + "apps/%s/users"
	GroupsBaseURL                      = APIBaseV1URL + "groups"
	ConnectorsBaseURL                  = APIBaseURL + "connectors"
	PrivilegesBaseURL                  = APIBaseV1URL + "privileges"
	GetPrivilegeByIdBaseURL            = PrivilegesBaseURL + "/%s"
	GetPrivilegeAssignableRolesBaseUrl = PrivilegesBaseURL + "/%s/roles"
	GetPrivilegeAssignableUsersBaseUrl = PrivilegesBaseURL + "/%s/users"

	UserPrivilegesBaseURL = UserBaseUrl + "/privileges"

	c1LastActionAttr = "c1_last_action"
)

type Client struct {
	httpClient *uhttp.BaseHttpClient
	token      string
	subdomain  string
}

func NewClient(ctx context.Context, clientId, clientSecret, subdomain string) (*Client, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector: failed to create HTTP client: %w", err)
	}

	wrappedClient, err := uhttp.NewBaseHttpClientWithContext(ctx, httpClient)
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector: failed to create wrapped HTTP client: %w", err)
	}

	accessToken, err := generateToken(ctx, wrappedClient, clientId, clientSecret, subdomain)
	if err != nil {
		return nil, fmt.Errorf("onelogin-connector: failed to generate access token during client initialization: %w", err)
	}

	return &Client{
		httpClient: wrappedClient,
		token:      accessToken,
		subdomain:  subdomain,
	}, nil
}

func (c *Client) GetUsers(ctx context.Context, paginationVars PaginationVars, groupId string) ([]*User, string, error) {
	var usersResponse []*User

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(UsersBaseURL, c.subdomain),
		http.MethodGet,
		&usersResponse,
		nil,
		[]QueryParam{
			&paginationVars,
			prepareUserFilters(),
			prepareGroupUsersFilters(groupId),
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return usersResponse, nextPage, nil
}

func (c *Client) GetUserByID(ctx context.Context, userID int) (*User, error) {
	var userResponse *User

	_, err := c.doRequest(
		ctx,
		fmt.Sprintf(UserBaseUrl, c.subdomain, strconv.Itoa(userID)),
		http.MethodGet,
		&userResponse,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return userResponse, nil
}

func (c *Client) GetUserSystemPrivileges(ctx context.Context, userID string) ([]UserSystemPrivilegeAssignment, error) {
	var response []UserSystemPrivilegeAssignment

	_, err := c.doRequest(
		ctx,
		fmt.Sprintf(UserPrivilegesBaseURL, c.subdomain, userID),
		http.MethodGet,
		&response,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) GetApps(ctx context.Context, paginationVars PaginationVars) ([]App, string, error) {
	var appsResponse []App

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(AppsBaseURL, c.subdomain),
		http.MethodGet,
		&appsResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return appsResponse, nextPage, nil
}

func (c *Client) GetAppUsers(ctx context.Context, appId string, paginationVars PaginationVars) ([]User, string, error) {
	var appUsersResponse []User

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(AppUsersBaseURL, c.subdomain, appId),
		http.MethodGet,
		&appUsersResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return appUsersResponse, nextPage, nil
}

func (c *Client) GetGroups(ctx context.Context, paginationVars PaginationVars) ([]Group, string, error) {
	var groupsResponse struct {
		Data       []Group    `json:"data"`
		Pagination Pagination `json:"pagination"`
	}

	_, err := c.doRequest(
		ctx,
		fmt.Sprintf(GroupsBaseURL, c.subdomain),
		http.MethodGet,
		&groupsResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	// GetGroups API doesn't return after-cursor header, so we need to extract it from the response
	nextPage := groupsResponse.Pagination.AfterCursor

	return groupsResponse.Data, nextPage, nil
}

func (c *Client) GetRoles(ctx context.Context, paginationVars PaginationVars) ([]Role, string, error) {
	var rolesResponse []Role

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(RolesBaseURL, c.subdomain),
		http.MethodGet,
		&rolesResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return rolesResponse, nextPage, nil
}

func (c *Client) GetRoleUsers(ctx context.Context, roleId string, paginationVars PaginationVars) ([]UserUnderRole, string, error) {
	var roleUsersResponse []UserUnderRole

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(RoleUsersBaseURL, c.subdomain, roleId),
		http.MethodGet,
		&roleUsersResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return roleUsersResponse, nextPage, nil
}

func (c *Client) GetRoleAdmins(ctx context.Context, roleId string, paginationVars PaginationVars) ([]UserUnderRole, string, error) {
	var roleAdminsResponse []UserUnderRole

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(RoleAdminsBaseURL, c.subdomain, roleId),
		http.MethodGet,
		&roleAdminsResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return roleAdminsResponse, nextPage, nil
}

func (c *Client) GetRoleApps(ctx context.Context, roleId string, paginationVars PaginationVars) ([]App, string, error) {
	var roleAppsResponse []App

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(RoleAppsBaseURL, c.subdomain, roleId),
		http.MethodGet,
		&roleAppsResponse,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return roleAppsResponse, nextPage, nil
}

func (c *Client) GrantRole(ctx context.Context, roleId, userId, entitlement string) error {
	var assignRoleResponse []BaseResource
	var roleUrl string

	if entitlement == "admin" {
		roleUrl = fmt.Sprintf(RoleAdminsBaseURL, c.subdomain, roleId)
	} else {
		roleUrl = fmt.Sprintf(RoleUsersBaseURL, c.subdomain, roleId)
	}

	payload, e := json.Marshal([]string{userId})
	if e != nil {
		return fmt.Errorf("onelogin-connector: failed to marshal grant role payload for user %s to role %s: %w", userId, roleId, e)
	}

	_, err := c.doRequest(
		ctx,
		roleUrl,
		http.MethodPost,
		&assignRoleResponse,
		payload,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) RevokeRole(ctx context.Context, roleId, userId, entitlement string) error {
	var roleUrl string
	if entitlement == "admin" {
		roleUrl = fmt.Sprintf(RoleAdminsBaseURL, c.subdomain, roleId)
	} else {
		roleUrl = fmt.Sprintf(RoleUsersBaseURL, c.subdomain, roleId)
	}

	payload, e := json.Marshal([]string{userId})
	if e != nil {
		return fmt.Errorf("onelogin-connector: failed to marshal revoke role payload for user %s from role %s: %w", userId, roleId, e)
	}
	_, err := c.doRequest(
		ctx,
		roleUrl,
		http.MethodDelete,
		nil,
		payload,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) SyncUserMappings(ctx context.Context, userID string) error {
	var updateResponse User

	payload, err := json.Marshal(UserUpdatePayload{
		CustomAttributes: map[string]interface{}{
			c1LastActionAttr: time.Now().Unix(),
		},
	})
	if err != nil {
		return fmt.Errorf("onelogin-connector: failed to marshal sync mappings payload for user %s: %w", userID, err)
	}

	_, err = c.doRequest(
		ctx,
		fmt.Sprintf(UserBaseUrl, c.subdomain, userID),
		http.MethodPut,
		&updateResponse,
		payload,
		&MappingsSyncParam{},
	)
	if err != nil {
		return fmt.Errorf("onelogin-connector: failed to sync mappings for user %s: %w", userID, err)
	}

	return nil
}

// ValidateScope checks if user has 'Manage all' scope needed to read/write all resources.
func (c *Client) ValidateScope(ctx context.Context, paginationVars PaginationVars) (string, error) {
	var response []BaseResource
	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(ConnectorsBaseURL, c.subdomain),
		http.MethodGet,
		&response,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return "", fmt.Errorf("onelogin-connector: failed to validate connector scope permissions: %w", err)
	}

	return nextPage, nil
}

func (c *Client) GetPrivileges(ctx context.Context, paginationVars PaginationVars) ([]AccountPrivilege, string, error) {
	var response []AccountPrivilege

	nextPage, err := c.doRequest(
		ctx,
		fmt.Sprintf(PrivilegesBaseURL, c.subdomain),
		http.MethodGet,
		&response,
		nil,
		[]QueryParam{
			&paginationVars,
		}...,
	)

	if err != nil {
		return nil, "", err
	}

	return response, nextPage, nil
}

func (c *Client) GetPrivilegeById(ctx context.Context, id string) (*AccountPrivilege, error) {
	var response AccountPrivilege

	_, err := c.doRequest(
		ctx,
		fmt.Sprintf(GetPrivilegeByIdBaseURL, c.subdomain, id),
		http.MethodGet,
		&response,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GetPrivilegeAssignableRoles(ctx context.Context, id string, nextLink string) (*PrivilegeAssigmentRolesResponse, error) {
	var response PrivilegeAssigmentRolesResponse

	if nextLink == "" {
		nextLink = fmt.Sprintf(GetPrivilegeAssignableRolesBaseUrl, c.subdomain, id)
	}

	_, err := c.doRequest(
		ctx,
		nextLink,
		http.MethodGet,
		&response,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GetPrivilegeAssignableUsers(ctx context.Context, id string, nextLink string) (*PrivilegeAssigmentUsersResponse, error) {
	var response PrivilegeAssigmentUsersResponse

	if nextLink == "" {
		nextLink = fmt.Sprintf(GetPrivilegeAssignableUsersBaseUrl, c.subdomain, id)
	}

	_, err := c.doRequest(
		ctx,
		nextLink,
		http.MethodGet,
		&response,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

func generateToken(ctx context.Context, httpClient *uhttp.BaseHttpClient, clientId, clientSecret, subdomain string) (string, error) {
	var credentialsResponse Credentials
	var body io.Reader

	// set request body
	jsonBody, err := json.Marshal(NewCredentialsGrant())
	if err != nil {
		return "", fmt.Errorf("onelogin-connector: failed to marshal token request body: %w", err)
	}
	body = bytes.NewBuffer(jsonBody)

	// create request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf(GenerateTokenBaseURL, subdomain),
		body,
	)
	if err != nil {
		return "", fmt.Errorf("onelogin-connector: failed to create token generation request: %w", err)
	}

	// set request headers
	req.Header.Set("Authorization", fmt.Sprintf("client_id:%s,client_secret:%s", clientId, clientSecret))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// send the request
	rawResponse, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer rawResponse.Body.Close()

	if err := json.NewDecoder(rawResponse.Body).Decode(&credentialsResponse); err != nil {
		return "", fmt.Errorf("onelogin-connector: failed to decode token response body: %w", err)
	}

	return credentialsResponse.AccessToken, nil
}

func (c *Client) doRequest(
	ctx context.Context,
	urlAddress string,
	method string,
	resourceResponse interface{},
	payload []byte,
	paramOptions ...QueryParam,
) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlAddress, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("onelogin-connector: failed to create HTTP request: %w", err)
	}

	queryParams := url.Values{}
	for _, queryParam := range paramOptions {
		queryParam.setup(&queryParams)
	}

	req.URL.RawQuery = queryParams.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer rawResponse.Body.Close()

	if method != http.MethodDelete {
		if err := json.NewDecoder(rawResponse.Body).Decode(&resourceResponse); err != nil {
			return "", fmt.Errorf("onelogin-connector: failed to decode response body from %s: %w", urlAddress, err)
		}
	}

	// extract header after-cursor for pagination
	nextPage := rawResponse.Header.Get("after-cursor")

	return nextPage, nil
}
