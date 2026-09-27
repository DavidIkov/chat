package chats

import (
	"chat/src/api_server/handlers/auth"
	"chat/src/api_server/services"
	"net/http"
)

func CreateChats(mux *http.ServeMux, services *services.Services) *ChatsHandler {
	handler := ChatsHandler{Services: services}
	requireUser := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireUser(services.Users, next)
	}
	mux.HandleFunc("POST /chat/create", requireUser(handler.CreateChatHandler))
	mux.HandleFunc("POST /chat/send_message", requireUser(handler.SendMessageHandler))
	mux.HandleFunc("GET /chat/get_chats", requireUser(handler.GetChatsHandler))
	mux.HandleFunc("GET /chat/get_messages", requireUser(handler.GetChatMessagesHandler))
	mux.HandleFunc("POST /chat/create_join_link", requireUser(handler.CreateJoinLinkHandler))
	mux.HandleFunc("POST /chat/join_chat", requireUser(handler.JoinChatHandler))
	return &handler
}

