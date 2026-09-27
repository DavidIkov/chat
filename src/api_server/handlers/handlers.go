package handlers

import (
	"chat/src/api_server/handlers/chats"
	"chat/src/api_server/handlers/users"
	"chat/src/api_server/services"
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
