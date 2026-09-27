package user

import (
	"chat/src/shared_api"
)

type UsersGetRequest struct {
	UIDs []uint `form:"uids"`
}

type UsersGetResponse struct {
	Error *shared_api.Error `json:"error,omitempty"`
	Users []User            `json:"users,omitempty"`
}
