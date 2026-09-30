package user

import (
	"chat/internal/shared/api"
)

type UsersGetRequest struct {
	UIDs []uint `form:"uids"`
}

type UsersGetResponse struct {
	Error *api.Error `json:"error,omitempty"`
	Users []User     `json:"users,omitempty"`
}
