package onelogin

type BaseResource struct {
	Id int `json:"id"`
}

type User struct {
	BaseResource
	Username     string `json:"username"`
	Email        string `json:"email"`
	Firstname    string `json:"firstname"`
	Lastname     string `json:"lastname"`
	Status       int    `json:"status"`
	ManagerId    *int   `json:"manager_user_id,omitempty"`
	ManagerEmail string
}

type Role struct {
	BaseResource
	Name   string `json:"name"`
	Admins []int  `json:"admins"`
	Users  []int  `json:"users"`
	Apps   []int  `json:"apps"`
}

type UserUnderRole struct {
	BaseResource
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type Credentials struct {
	AccessToken string `json:"access_token"`
}

type App struct {
	BaseResource
	Name    string `json:"name"`
	RoleIDs []int  `json:"role_ids"`
}

type Group struct {
	BaseResource
	Name string `json:"name"`
}

type PrivilegeStatement struct {
	Effect string   `json:"Effect"`
	Action []string `json:"Action"`
	Scope  []string `json:"Scope"`
}

type Privilege struct {
	Version   string               `json:"Version"`
	Statement []PrivilegeStatement `json:"Statement"`
}

type AccountPrivilege struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Privilege   Privilege `json:"privilege"`
}

type PrivilegeAssigmentRolesResponse struct {
	Total        int      `json:"total"`
	Roles        []string `json:"roles"`
	BeforeCursor string   `json:"beforeCursor"`
	PreviousLink string   `json:"previousLink"`
	AfterCursor  string   `json:"afterCursor"`
	NextLink     string   `json:"nextLink"`
}

type PrivilegeAssigmentUsersResponse struct {
	Total        int    `json:"total"`
	Users        []int  `json:"users"`
	BeforeCursor string `json:"beforeCursor"`
	PreviousLink string `json:"previousLink"`
	AfterCursor  string `json:"afterCursor"`
	NextLink     string `json:"nextLink"`
}
