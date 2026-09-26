package chats

import (
	"database/sql"
)

func CreateChats(db *sql.DB) (*ChatsService, error) {
	var service ChatsService

	service.db = db

	if _, err := db.Exec(`
create table if not exists chats (
    uid serial primary key,
    name text not null unique,
    creator_user_uid integer not null references users(uid),
	created_at BIGINT not null
)`); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`
create table if not exists messages (
    uid serial primary key,
    user_uid integer not null references users(uid),
    chat_uid integer not null references chats(uid),
	created_at BIGINT not null,
	text TEXT not null
)`); err != nil {
		return nil, err
	}

	return &service, nil
}
