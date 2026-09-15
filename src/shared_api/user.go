package shared_api

type User struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Error Error  `json:"error"`
	Token string `json:"token"`
	User  User   `json:"user_info"`
}

type LogoutRequest struct {
	Token string `json:"token"`
}

type LogoutResponse struct {
	Error Error `json:"error"`
}

type RegistrationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type RegistrationResponse struct {
	Error Error  `json:"error"`
	Token string `json:"token"`
	User  User   `json:"user_info"`
}
