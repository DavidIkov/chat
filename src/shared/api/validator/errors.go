package validator

import (
	"chat/src/shared/api"
)

var (
	errUserNameTooSmall     = &api.Error{Field: "name", Message: "too small user name"}
	errUserNameTooBig       = &api.Error{Field: "name", Message: "too big user name"}
	errUserPasswordTooSmall = &api.Error{Field: "password", Message: "too small user password"}
	errUserPasswordTooBig   = &api.Error{Field: "password", Message: "too big user password"}
	errChatNameTooSmall     = &api.Error{Field: "name", Message: "too small chat name"}
	errChatNameTooBig       = &api.Error{Field: "name", Message: "too big chat name"}
	errMessageTextTooSmall  = &api.Error{Field: "text", Message: "too small message text"}
	errMessageTextTooBig    = &api.Error{Field: "text", Message: "too big message text"}
)
