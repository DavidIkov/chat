package chats

import (
	"chat/internal/webui_server/handlers/middleware"
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/handlers/routes"
	"chat/internal/webui_server/services"
	"net/http"
)

// CreateChats registers the chat list, chat and message routes. Every route is
// scoped to a connection id and, where relevant, an api_server chat uid.
func CreateChats(mux *http.ServeMux, services *services.Services, templates *render.Templates) *ChatsHandler {
	handler := ChatsHandler{Services: services, Templates: templates}
	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RequireSession(services.Sessions, next)
	}

	server := "{" + routes.ServerIDPathKey + "}"
	chat := "{" + routes.ChatUIDPathKey + "}"
	base := "/servers/" + server + "/chats"

	mux.HandleFunc("GET "+base, requireSession(handler.ChatsPageHandler))
	mux.HandleFunc("POST "+base, requireSession(handler.CreateChatHandler))
	mux.HandleFunc("POST "+base+"/join", requireSession(handler.JoinChatHandler))
	mux.HandleFunc("GET "+base+"/"+chat, requireSession(handler.ChatPageHandler))
	mux.HandleFunc("POST "+base+"/"+chat+"/messages", requireSession(handler.SendMessageHandler))
	mux.HandleFunc("POST "+base+"/"+chat+"/join_link", requireSession(handler.CreateJoinLinkHandler))
	mux.HandleFunc("POST "+base+"/"+chat+"/leave", requireSession(handler.LeaveChatHandler))

	return &handler
}
