package users

import (
	"chat/src/api_server/services"
	"net/http"
)

func CreateUsers(mux *http.ServeMux, services *services.Services) *UsersHandler {
	handler := UsersHandler{Services: services}
	mux.HandleFunc("/user/register", handler.UserRegistrationHandler)
	mux.HandleFunc("/user/login", handler.UserLogInHandler)
	mux.HandleFunc("/user/logout", handler.UserLogOutHandler)
	mux.HandleFunc("/user/get", handler.UsersGetHandler)
	return &handler
}
