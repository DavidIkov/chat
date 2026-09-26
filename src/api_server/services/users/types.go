package users

import (
	"database/sql"
)

type User struct {
	UID  uint
	Name string
}

type UserSession struct {
	Token string
	UID   uint
}

type UsersService struct {
	db            *sql.DB
	sessions []UserSession
}
