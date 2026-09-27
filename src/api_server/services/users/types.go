package users

import (
	"database/sql"
	"sync"
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
	db       *sql.DB
	mutex    sync.RWMutex
	sessions []UserSession
}
