package users

import (
	"database/sql"
)

func CreateUsers(db *sql.DB) (*UsersService, error) {
	var manager UsersService

	manager.db = db

	_, err := db.Exec(`
create table if not exists users (
    uid serial primary key,
    name text not null unique,
    password_hash text not null
)`)
	if err != nil {
		return nil, err
	}

	return &manager, nil
}
