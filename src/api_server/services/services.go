package services

import (
	"chat/src/api_server/services/users"
	"database/sql"
)

type Services struct {
	Users *users.UsersService
}

func CreateServices(db *sql.DB) (*Services, error) {
	users, err := users.CreateUsers(db)
	if err != nil {
		return nil, err
	}

	return &Services{Users: users}, nil
}
