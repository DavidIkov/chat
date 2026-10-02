package chats

import (
	"chat/internal/api_server/handlers/auth"
	chatsmiddleware "chat/internal/api_server/handlers/chats/middleware"
	"chat/internal/api_server/services"
	"net/http"
)

func CreateChats(mux *http.ServeMux, services *services.Services) *ChatsHandler {
	handler := ChatsHandler{Services: services}
	requireUser := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireUser(services.Users, next)
	}
	requireChatMember := func(next http.HandlerFunc) http.HandlerFunc {
		return chatsmiddleware.RequireChatMember(services.Chats, next)
	}
	mux.HandleFunc("POST /chat/create", requireUser(handler.CreateChatHandler))
	mux.HandleFunc("POST /chat/{"+chatsmiddleware.ChatUIDPathKey+"}/send_message", requireUser(requireChatMember(handler.SendMessageHandler)))
	mux.HandleFunc("GET /chat/get_chats", requireUser(handler.GetChatsHandler))
	mux.HandleFunc("GET /chat/{"+chatsmiddleware.ChatUIDPathKey+"}/get_messages", requireUser(requireChatMember(handler.GetChatMessagesHandler)))
	mux.HandleFunc("GET /chat/{"+chatsmiddleware.ChatUIDPathKey+"}/get_members", requireUser(requireChatMember(handler.GetChatMembersHandler)))
	mux.HandleFunc("POST /chat/{"+chatsmiddleware.ChatUIDPathKey+"}/create_join_link", requireUser(requireChatMember(handler.CreateJoinLinkHandler)))
	mux.HandleFunc("POST /chat/{"+chatsmiddleware.ChatUIDPathKey+"}/leave", requireUser(requireChatMember(handler.LeaveChatHandler)))
	mux.HandleFunc("POST /chat/join_chat", requireUser(handler.JoinChatHandler))
	return &handler
}
