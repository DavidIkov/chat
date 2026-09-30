package handlers

import (
	"chat/internal/api_server/handlers/chats"
	"chat/internal/api_server/handlers/users"
	"chat/internal/api_server/services"
	"net/http"
)

type Handlers struct {
	Users *users.UsersHandler
	Chats *chats.ChatsHandler
}

func CreateHandlers(mux *http.ServeMux, services *services.Services) (*Handlers, error) {
	users := users.CreateUsers(mux, services)
	chats := chats.CreateChats(mux, services)
	return &Handlers{
		Users: users,
		Chats: chats,
	}, nil

}
