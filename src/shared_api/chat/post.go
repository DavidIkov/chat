package chat

import (
	"chat/src/shared"
	"chat/src/shared_api"
)

type CreateChatRequest struct {
	Name string `json:"name"`
}

type CreateChatResponse struct {
	Error   *shared_api.Error `json:"error,omitempty"`
	ChatUID shared.UID        `json:"chat_uid,omitempty"`
}

type SendMessageRequest struct {
	ChatUID shared.UID `json:"chat_uid"`
	Text    string     `json:"text"`
}

type SendMessageResponse struct {
	Error      *shared_api.Error `json:"error,omitempty"`
	MessageUID shared.UID        `json:"message_uid,omitempty"`
}
