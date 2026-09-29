package chat

import (
	"chat/src/shared"
	"chat/src/shared/api"
)

type GetChatsRequest struct {
	// UIDs limits the result to these chats. Optional: when no uids are given the
	// server returns every chat the caller is a member of (see GetUserChats).
	UIDs []shared.UID `form:"uids,omitempty"`
}

type GetChatsResponse struct {
	Error *api.Error `json:"error,omitempty"`
	Chats []Chat     `json:"chats,omitempty"`
}

/*
If BeforeMessageUID is not zero, messages with uid < BeforeMessageUID are returned,
if AfterMessageUID is not zero, messages with uid > AfterMessageUID are returned,
otherwise only last messages are returned.
Amount of messages is limited by Limit.
*/
type GetChatMessagesRequest struct {
	Limit            uint       `form:"limit"`
	BeforeMessageUID shared.UID `form:"before_message_uid,omitempty"`
	AfterMessageUID  shared.UID `form:"after_message_uid,omitempty"`
}

type GetChatMessagesResponse struct {
	Error    *api.Error `json:"error,omitempty"`
	Messages []Message  `json:"messages,omitempty"`
}

type GetChatMembersResponse struct {
	Error   *api.Error `json:"error,omitempty"`
	Members []Member   `json:"members,omitempty"`
}
