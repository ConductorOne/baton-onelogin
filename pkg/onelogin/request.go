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
	Limit    int
	Cursor   string
	V1Cursor string
}

type Pagination struct {
	BeforeCursor string `json:"before_cursor"`
	AfterCursor  string `json:"after_cursor"`
}

func (pV *PaginationVars) setup(params *url.Values) {
	// need to set limit just for the first call
	if pV.Limit != 0 && pV.Cursor == "" {
		params.Set("limit", fmt.Sprintf("%d", pV.Limit))
	}

	if pV.Cursor != "" {
		params.Set("cursor", pV.Cursor)
	}

	// different name for different API version
	if pV.V1Cursor != "" {
		params.Set("after_cursor", pV.V1Cursor)
	}
}

// Filtering variables and types.
var (
	UserFields = []string{"id", "email", "username", "firstname", "lastname", "status", "group_id"}
)

type FilterVars struct {
	Fields  []string
	GroupId string
}

func (fV *FilterVars) setup(params *url.Values) {
	if len(fV.Fields) != 0 {
		params.Set("fields", strings.Join(fV.Fields, ","))
	}

	if fV.GroupId != "" {
		params.Set("group_id", fV.GroupId)
	}
}

func prepareUserFilters() *FilterVars {
	return &FilterVars{
		Fields: UserFields,
	}
}

func prepareGroupUsersFilters(groupId string) *FilterVars {
	return &FilterVars{
		GroupId: groupId,
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
