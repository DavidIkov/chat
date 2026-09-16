package shared_api

type User struct {
	UID   uint   `json:"uid"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type LogInUserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LogInUserResponse struct {
	Error Error `json:"error"`
	User  User  `json:"user_info"`
}

type LogOutUserRequest struct {
	Token string `json:"token"`
}

type LogOutUserResponse struct {
	Error Error `json:"error"`
}

type UserRegistrationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserRegistrationResponse struct {
	Error Error `json:"error"`
	User  User  `json:"user_info"`
}
