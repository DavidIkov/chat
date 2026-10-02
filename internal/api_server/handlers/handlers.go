package handlers

import (
	"chat/internal/api_server/handlers/chats"
	"chat/internal/api_server/handlers/middleware"
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

// NewServer registers every api_server route on a fresh mux and wraps it in
// panic recovery, returning the handler the HTTP server should serve. It is the
// single wiring point shared by cmd/api_server and the integration tests.
func NewServer(services *services.Services) (http.Handler, error) {
	mux := http.NewServeMux()
	if _, err := CreateHandlers(mux, services); err != nil {
		return nil, err
	}
	return middleware.Recover(mux), nil
}
