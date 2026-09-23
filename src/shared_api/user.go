package shared_api

type User struct {
	UID   uint   `json:"uid"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type UserLogInRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserLogInResponse struct {
	Error *Error `json:"error,omitempty"`
	User  *User  `json:"user_info,omitempty"`
}

type UserLogOutRequest struct {
	Token string `json:"token"`
}

type UserLogOutResponse struct {
	Error Error `json:"error,omitempty"`
}

type UserRegistrationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserRegistrationResponse struct {
	Error *Error `json:"error,omitempty"`
	User  *User  `json:"user_info,omitempty"`
}
