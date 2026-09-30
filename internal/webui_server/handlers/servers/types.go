package servers

import (
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/services"
	"chat/internal/webui_server/services/sessions"
)

// ServersHandler serves the landing page and the per-server connection pages.
type ServersHandler struct {
	Services  *services.Services
	Templates *render.Templates
}

// IndexPage is the view model for index.html: every connected api server.
type IndexPage struct {
	render.Page
	Servers []*sessions.ServerConnection
	Error   string
}

// ServerPage is the view model for server.html: one connection, and whether a
// user is currently signed in on it.
type ServerPage struct {
	render.Page
	Server   *sessions.ServerConnection
	SignedIn bool
	// Notice is an optional success message (e.g. after deleting the account).
	Notice string
	Error  string
}
