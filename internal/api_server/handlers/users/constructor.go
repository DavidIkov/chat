package users

import (
	"chat/internal/api_server/handlers/auth"
	"chat/internal/api_server/services"
	"net/http"
)

func CreateUsers(mux *http.ServeMux, services *services.Services) *UsersHandler {
	handler := UsersHandler{Services: services}
	requireUser := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireUser(services.Users, next)
	}
	mux.HandleFunc("POST /user/register", handler.UserRegistrationHandler)
	mux.HandleFunc("POST /user/login", handler.UserLogInHandler)
	mux.HandleFunc("POST /user/logout", requireUser(handler.UserLogOutHandler))
	mux.HandleFunc("POST /user/delete", requireUser(handler.UserDeleteHandler))
	mux.HandleFunc("GET /user/get", requireUser(handler.UsersGetHandler))
	return &handler
}
