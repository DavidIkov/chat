package chats

import (
	"chat/src/api_server/handlers/auth"
	"chat/src/api_server/handlers/middleware"
	chatservice "chat/src/api_server/services/chats"
	"chat/src/shared"
	"chat/src/shared/api"
	"context"
	"errors"
	"net/http"
	"strconv"
)

type contextKey struct{}

var errInvalidChatUID = errors.New("invalid chat_uid")

func withChatUID(ctx context.Context, chatUID shared.UID) context.Context {
	return context.WithValue(ctx, contextKey{}, chatUID)
}

func ChatUIDFromContext(ctx context.Context) (shared.UID, bool) {
	chatUID, ok := ctx.Value(contextKey{}).(shared.UID)
	return chatUID, ok
}

func chatUIDFromPath(r *http.Request) (shared.UID, error) {
	raw := r.PathValue(chatUIDPathKey)
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || parsed == 0 {
		return 0, errInvalidChatUID
	}
	return shared.UID(parsed), nil
}

func writeChatError(w http.ResponseWriter, status int, message string) {
	middleware.WriteJSON(w, status, struct {
		Error *api.Error `json:"error,omitempty"`
	}{Error: &api.Error{Field: chatUIDPathKey, Message: message}})
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

// chatsErrorStatus maps a chats business error to the HTTP status to return.
func chatsErrorStatus(err error) int {
	switch {
	case errors.Is(err, chatservice.InvalidJoinLinkError):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
