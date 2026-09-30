package user

import (
	"chat/internal/shared/api"
)

type UserLogInRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserLogInResponse struct {
	Error       *api.Error   `json:"error,omitempty"`
	UserSession *UserSession `json:"user_session,omitempty"`
}

type UserLogOutResponse struct {
	Error *api.Error `json:"error,omitempty"`
}

type UserDeleteRequest struct {
	DeleteMessages bool `json:"delete_messages"`
}

type UserDeleteResponse struct {
	Error *api.Error `json:"error,omitempty"`
}

type UserRegistrationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserRegistrationResponse struct {
	Error       *api.Error   `json:"error,omitempty"`
	UserSession *UserSession `json:"user_session,omitempty"`
}
