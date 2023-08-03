package onelogin

type BaseResource struct {
	Id int `json:"id"`
}

type User struct {
	BaseResource
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Status    int    `json:"status"`
}

type Role struct {
	BaseResource
	Name   string `json:"name"`
	Admins []int  `json:"admins"`
	Users  []int  `json:"users"`
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
