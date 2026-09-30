package chats

import (
	"chat/internal/shared"
	"database/sql"
	"sync"
)

type Chat struct {
	ChatUID        shared.UID
	CreatorUserUID shared.UID
	CreatedAt      shared.Time
	Name           string
}

type Message struct {
	UserUID    shared.UID  `json:"user_uid"`
	ChatUID    shared.UID  `json:"chat_uid"`
	MessageUID shared.UID  `json:"message_uid"`
	CreatedAt  shared.Time `json:"created_at"`
	Text       string      `json:"text"`
}

type Member struct {
	UserUID shared.UID
}

type ChatsService struct {
	db        *sql.DB
	mutex     sync.RWMutex
	joinLinks *joinLinkStore
}
