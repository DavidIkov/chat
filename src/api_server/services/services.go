package services

import (
	"chat/src/api_server/services/chats"
	"chat/src/api_server/services/users"
	"database/sql"
)

type Services struct {
	Users *users.UsersService
	Chats *chats.ChatsService
}

func CreateServices(db *sql.DB) (*Services, error) {
	users, err := users.CreateUsers(db)
	if err != nil {
		return nil, err
	}

	chats, err := chats.CreateChats(db)
	if err != nil {
		return nil, err
	}

	return &Services{Users: users, Chats: chats}, nil
}
