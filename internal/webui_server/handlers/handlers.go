package handlers

import (
	"chat/internal/webui_server/handlers/chats"
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/handlers/servers"
	"chat/internal/webui_server/services"
	"io/fs"
	"net/http"
)

// Handlers groups the HTTP handler sets of the webui server. It mirrors
// api_server/handlers.Handlers.
type Handlers struct {
	Servers *servers.ServersHandler
	Chats   *chats.ChatsHandler
}

// CreateHandlers registers every webui route on mux and mounts the static file
// server under /static/. It mirrors api_server/handlers.CreateHandlers, with the
// extra template and static dependencies the webui needs.
//
// Setup order matters only in that the static mount must not collide with the
// page routes; the per-domain packages own their own patterns.
func CreateHandlers(mux *http.ServeMux, services *services.Services, templates *render.Templates, staticFS fs.FS) (*Handlers, error) {
	serversHandler := servers.CreateServers(mux, services, templates)
	chatsHandler := chats.CreateChats(mux, services, templates)

	// staticFS is rooted at the static directory, so strip the "/static/" URL
	// prefix before the file server looks the request path up in it.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	return &Handlers{
		Servers: serversHandler,
		Chats:   chatsHandler,
	}, nil
}
