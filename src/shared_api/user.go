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

// UserLogOutRequest is intentionally empty: the session token is read from the
// "Authorization" header, so logging out carries no body.
type UserLogOutRequest struct{}

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

// UsersGetRequest is decoded from the URL query, so its fields carry "form"
// tags instead of JSON ones.
type UsersGetRequest struct {
	UIDs []uint `form:"uids"`
}

type UsersGetResponse struct {
	Error *Error `json:"error,omitempty"`
	Users []User `json:"users,omitempty"`
}
