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

type CreateJoinLinkRequest struct {
	ChatUID shared.UID `json:"chat_uid"`
	// LifetimeSeconds is how long the link stays valid; 0 means forever.
	LifetimeSeconds int64 `json:"lifetime_seconds"`
	// MaxUses caps the number of users that can join; 0 means unlimited.
	MaxUses uint `json:"max_uses"`
}

type CreateJoinLinkResponse struct {
	Error *shared_api.Error `json:"error,omitempty"`
	Token string            `json:"token,omitempty"`
	// ExpiresAt is 0 when the link never expires while the server runs.
	ExpiresAt shared.Time `json:"expires_at,omitempty"`
}

type JoinChatRequest struct {
	Token string `json:"token"`
}

type JoinChatResponse struct {
	Error   *shared_api.Error `json:"error,omitempty"`
	ChatUID shared.UID        `json:"chat_uid,omitempty"`
}
