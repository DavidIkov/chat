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
	requireChatMember := func(next http.HandlerFunc) http.HandlerFunc {
		return RequireChatMember(services.Chats, next)
	}
	mux.HandleFunc("POST /chat/create", requireUser(handler.CreateChatHandler))
	mux.HandleFunc("POST /chat/{" + chatUIDPathKey + "}/send_message", requireUser(requireChatMember(handler.SendMessageHandler)))
	mux.HandleFunc("GET /chat/get_chats", requireUser(handler.GetChatsHandler))
	mux.HandleFunc("GET /chat/{" + chatUIDPathKey + "}/get_messages", requireUser(requireChatMember(handler.GetChatMessagesHandler)))
	mux.HandleFunc("GET /chat/{" + chatUIDPathKey + "}/get_members", requireUser(requireChatMember(handler.GetChatMembersHandler)))
	mux.HandleFunc("POST /chat/{" + chatUIDPathKey + "}/create_join_link", requireUser(requireChatMember(handler.CreateJoinLinkHandler)))
	mux.HandleFunc("POST /chat/{" + chatUIDPathKey + "}/leave", requireUser(handler.LeaveChatHandler))
	mux.HandleFunc("POST /chat/join_chat", requireUser(handler.JoinChatHandler))
	return &handler
}

