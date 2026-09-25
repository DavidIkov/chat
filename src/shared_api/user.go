package shared_api

type User struct {
	UID  uint   `json:"uid"`
	Name string `json:"name"`
}

type UserSession struct {
	UID   uint   `json:"uid"`
	Token string `json:"token"`
}

type UserLogInRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserLogInResponse struct {
	Error       *Error       `json:"error,omitempty"`
	UserSession *UserSession `json:"user_session,omitempty"`
}

type UserLogOutRequest struct {
	Token string `json:"token"`
}

type UserLogOutResponse struct {
	Error *Error `json:"error,omitempty"`
}

type UserRegistrationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserRegistrationResponse struct {
	Error       *Error       `json:"error,omitempty"`
	UserSession *UserSession `json:"user_session,omitempty"`
}

type UsersGetRequest struct {
	Token string `json:"token"`
	UIDs  []uint `json:"uids"`
}

type UsersGetResponse struct {
	Error *Error `json:"error,omitempty"`
	Users []User `json:"users,omitempty"`
}
