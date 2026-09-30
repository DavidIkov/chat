package middleware

import (
	"chat/internal/api_server/handlers/auth"
	httpmiddleware "chat/internal/api_server/handlers/middleware"
	chatservice "chat/internal/api_server/services/chats"
	"chat/internal/shared"
	"chat/internal/shared/api"
	"net/http"
	"strconv"
)

func chatUIDFromPath(r *http.Request) (shared.UID, error) {
	raw := r.PathValue(ChatUIDPathKey)
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || parsed == 0 {
		return 0, errInvalidChatUID
	}
	return shared.UID(parsed), nil
}

func writeChatError(w http.ResponseWriter, status int, message string) {
	httpmiddleware.WriteJSON(w, status, struct {
		Error *api.Error `json:"error,omitempty"`
	}{Error: &api.Error{Field: ChatUIDPathKey, Message: message}})
}

// RequireChatMember must be wrapped by auth.RequireUser, which resolves the
// session it relies on.
func RequireChatMember(chats *chatservice.ChatsService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chatUID, err := chatUIDFromPath(r)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, err.Error())
			return
		}

		session, _ := auth.SessionFromContext(r.Context())

		isMember, err := chats.IsChatMember(r.Context(), chatUID, session.UID)
		if err != nil {
			writeChatError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !isMember {
			writeChatError(w, http.StatusForbidden, "user does not belong to the chat")
			return
		}

		next(w, r.WithContext(withChatUID(r.Context(), chatUID)))
	}
}
