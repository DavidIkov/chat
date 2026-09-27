package chat

import (
	"chat/src/shared"
)

type Chat struct {
	ChatUID        shared.UID        `json:"chat_uid"`
	CreatorUserUID shared.UID        `json:"creator_user_uid"`
	CreatedAt      shared.Time `json:"created_at"`
	Name           string      `json:"name"`
}

type Message struct {
	UserUID    shared.UID        `json:"user_uid"`
	ChatUID    shared.UID        `json:"chat_uid"`
	MessageUID shared.UID        `json:"message_uid"`
	CreatedAt  shared.Time `json:"created_at"`
	Text       string      `json:"text"`
}

type Member struct {
	UserUID shared.UID `json:"user_uid"`
}
