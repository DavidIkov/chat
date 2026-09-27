package user

import (
	"chat/src/shared_api"
)

type UserLogInRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserLogInResponse struct {
	Error       *shared_api.Error `json:"error,omitempty"`
	UserSession *UserSession      `json:"user_session,omitempty"`
}

// UserLogOutRequest is intentionally empty: the session token is read from the
// "Authorization" header, so logging out carries no body.
type UserLogOutRequest struct{}

type UserLogOutResponse struct {
	Error *shared_api.Error `json:"error,omitempty"`
}

type UserRegistrationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserRegistrationResponse struct {
	Error       *shared_api.Error `json:"error,omitempty"`
	UserSession *UserSession      `json:"user_session,omitempty"`
}
