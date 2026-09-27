package chat

import (
	"chat/src/shared"
	"chat/src/shared_api"
)

type GetChatsRequest struct {
	UIDs []shared.UID `form:"uids"`
}

type GetChatsResponse struct {
	Error *shared_api.Error `json:"error,omitempty"`
	Chats []Chat            `json:"chats,omitempty"`
}

/*
If BeforeMessageUID is not zero, messages with uid < BeforeMessageUID are returned,
if AfterMessageUID is not zero, messages with uid > AfterMessageUID are returned,
otherwise only last messages are returned.
Amount of messages is limited by Limit.
*/
type GetChatMessagesRequest struct {
	ChatUID          shared.UID `form:"chat_uid"`
	Limit            uint       `form:"limit"`
	BeforeMessageUID shared.UID `form:"before_message_uid,omitempty"`
	AfterMessageUID  shared.UID `form:"after_message_uid,omitempty"`
}

type GetChatMessagesResponse struct {
	Error    *shared_api.Error `json:"error,omitempty"`
	Messages []Message         `json:"messages,omitempty"`
}
