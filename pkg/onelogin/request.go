package onelogin

import (
	"fmt"
	"net/url"
	"strings"
)

// Query parameters types.
type QueryParam interface {
	setup(params *url.Values)
}

type PaginationVars struct {
	Limit  int
	Cursor string
}

func (pV *PaginationVars) setup(params *url.Values) {
	if pV.Limit != 0 {
		params.Set("limit", fmt.Sprintf("%d", pV.Limit))
	}

	if pV.Cursor != "" {
		params.Set("cursor", pV.Cursor)
	}
}

// Filtering variables and types.
var (
	UserFields = []string{"id", "email", "username", "firstname", "lastname", "status"}
	RoleFields = []string{"id", "name", "admins", "users"}
)

type FilterVars struct {
	Fields []string
}

func (fV *FilterVars) setup(params *url.Values) {
	if len(fV.Fields) != 0 {
		params.Set("fields", strings.Join(fV.Fields, ","))
	}
}

func prepareUserFilters() *FilterVars {
	return &FilterVars{
		Fields: UserFields,
	}
}

func prepareRoleFilters() *FilterVars {
	return &FilterVars{
		Fields: RoleFields,
	}
}

// Request Body types.
type GrantBody struct {
	GrantType string `json:"grant_type"`
}

func NewCredentialsGrant() *GrantBody {
	return &GrantBody{
		GrantType: "client_credentials",
	}
}
