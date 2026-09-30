package chats

import (
	"chat/internal/shared"
	chatapi "chat/internal/shared/api/chat"
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/services"
	"chat/internal/webui_server/services/sessions"
)

// ChatsHandler serves the per-server chat list and the chat pages.
type ChatsHandler struct {
	Services  *services.Services
	Templates *render.Templates
}

// ChatsPage is the view model for chats.html: the chats visible to the signed-in
// user on one server.
type ChatsPage struct {
	render.Page
	Server *sessions.ServerConnection
	Chats  []chatapi.Chat
	Error  string
}

// MessageItem is one message plus the resolved author name.
type MessageItem struct {
	Message  chatapi.Message
	UserName string
}

// MemberItem is one chat member plus the resolved name.
type MemberItem struct {
	UID  shared.UID
	Name string
}

// ChatPage is the view model for chat.html: one chat's messages and members.
type ChatPage struct {
	render.Page
	Server   *sessions.ServerConnection
	Chat     chatapi.Chat
	Messages []MessageItem
	Members  []MemberItem
	JoinLink string
	Error    string
}
