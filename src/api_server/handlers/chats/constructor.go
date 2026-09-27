package chats

import (
	"chat/src/api_server/services"
	"net/http"
)

func CreateChats(mux *http.ServeMux, services *services.Services) *ChatsHandler {
	handler := ChatsHandler{Services: services}
	mux.HandleFunc("/chat/create", handler.CreateChatHandler)
	mux.HandleFunc("/chat/send_message", handler.SendMessageHandler)
	mux.HandleFunc("/chat/get_chats", handler.GetChatsHandler)
	mux.HandleFunc("/chat/get_messages", handler.GetChatMessagesHandler)
	return &handler
}
