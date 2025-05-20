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
