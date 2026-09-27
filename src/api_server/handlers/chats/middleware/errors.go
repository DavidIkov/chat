package middleware

import (
	chatservice "chat/src/api_server/services/chats"
	"errors"
	"net/http"
)

var errInvalidChatUID = errors.New("invalid chat_uid")

// ChatsErrorStatus maps a chats business error to the HTTP status to return.
func ChatsErrorStatus(err error) int {
	switch {
	case errors.Is(err, chatservice.InvalidJoinLinkError):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
