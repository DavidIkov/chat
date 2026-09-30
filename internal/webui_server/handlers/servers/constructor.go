package servers

import (
	"chat/internal/webui_server/handlers/middleware"
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/handlers/routes"
	"chat/internal/webui_server/services"
	"net/http"
)

// CreateServers registers the landing-page and connection routes.
func CreateServers(mux *http.ServeMux, services *services.Services, templates *render.Templates) *ServersHandler {
	handler := ServersHandler{Services: services, Templates: templates}
	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RequireSession(services.Sessions, next)
	}

	server := "{" + routes.ServerIDPathKey + "}"

	mux.HandleFunc("GET /{$}", requireSession(handler.IndexHandler))
	mux.HandleFunc("POST /servers", requireSession(handler.AddServerHandler))
	mux.HandleFunc("GET /servers/"+server, requireSession(handler.ServerPageHandler))
	mux.HandleFunc("POST /servers/"+server+"/remove", requireSession(handler.RemoveServerHandler))
	mux.HandleFunc("POST /servers/"+server+"/login", requireSession(handler.LogInHandler))
	mux.HandleFunc("POST /servers/"+server+"/register", requireSession(handler.RegisterHandler))
	mux.HandleFunc("POST /servers/"+server+"/logout", requireSession(handler.LogOutHandler))
	mux.HandleFunc("POST /servers/"+server+"/delete", requireSession(handler.DeleteUserHandler))

	return &handler
}
