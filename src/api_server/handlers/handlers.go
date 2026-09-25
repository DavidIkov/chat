package handlers

import (
	"chat/src/api_server/handlers/users"
	"chat/src/api_server/services"
	"net/http"
)

type Handlers struct {
	Users *users.UsersHandler
}

func CreateHandlers(mux *http.ServeMux, services *services.Services) (*Handlers, error) {
	users := users.CreateUsers(mux, services)
	return &Handlers{
		Users: users,
	}, nil

}
