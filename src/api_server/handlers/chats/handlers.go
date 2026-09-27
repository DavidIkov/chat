package chats

import (
	"chat/src/api_server/handlers/auth"
	"chat/src/api_server/handlers/middleware"
	chatservice "chat/src/api_server/services/chats"
	"chat/src/shared"
	"chat/src/shared_api"
	"chat/src/shared_api/chat"
	"context"
	"errors"
	"net/http"
)

// chatsErrorStatus maps a chats business error to the HTTP status to return:
// forbidden when the caller may not touch the chat, internal server error
// otherwise. Authentication errors never reach here: auth.RequireUser answers them.
func chatsErrorStatus(err error) int {
	switch {
	case errors.Is(err, errNotChatMember):
		return http.StatusForbidden
	case errors.Is(err, chatservice.InvalidJoinLinkError):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// ensureChatMember returns nil when userUID is a member of chatUID,
// errNotChatMember when it is not, and the underlying error otherwise.
func (this *ChatsHandler) ensureChatMember(ctx context.Context, userUID shared.UID, chatUID shared.UID) error {
	isMember, err := this.Services.Chats.IsChatMember(ctx, chatUID, userUID)
	if err != nil {
		return err
	}
	if !isMember {
		return errNotChatMember
	}
	return nil
}

func (this *ChatsHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	var request chat.CreateChatRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	createdChat, err := this.Services.Chats.CreateChat(r.Context(), request.Name, session.UID)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chat.CreateChatResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chat.CreateChatResponse{ChatUID: createdChat.ChatUID})
}

func (this *ChatsHandler) SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	var request chat.SendMessageRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	if err := this.ensureChatMember(r.Context(), session.UID, request.ChatUID); err != nil {
		middleware.WriteJSON(w, chatsErrorStatus(err), chat.SendMessageResponse{
			Error: &shared_api.Error{Field: "chat_uid", Message: err.Error()},
		})
		return
	}

	message, err := this.Services.Chats.SendMessage(r.Context(), request.ChatUID, session.UID, request.Text)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chat.SendMessageResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chat.SendMessageResponse{MessageUID: message.MessageUID})
}

func (this *ChatsHandler) GetChatsHandler(w http.ResponseWriter, r *http.Request) {
	var request chat.GetChatsRequest
	if !middleware.DecodeQuery(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	// GetChats only returns chats the acting user is a member of.
	chats, err := this.Services.Chats.GetChats(r.Context(), session.UID, request.UIDs)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chat.GetChatsResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	response := chat.GetChatsResponse{Chats: make([]chat.Chat, 0, len(chats))}
	for i := range chats {
		foundChat := &chats[i]
		response.Chats = append(response.Chats, chat.Chat{
			ChatUID:        foundChat.ChatUID,
			CreatorUserUID: foundChat.CreatorUserUID,
			CreatedAt:      foundChat.CreatedAt,
			Name:           foundChat.Name,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *ChatsHandler) GetChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	var request chat.GetChatMessagesRequest
	if !middleware.DecodeQuery(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	if err := this.ensureChatMember(r.Context(), session.UID, request.ChatUID); err != nil {
		middleware.WriteJSON(w, chatsErrorStatus(err), chat.GetChatMessagesResponse{
			Error: &shared_api.Error{Field: "chat_uid", Message: err.Error()},
		})
		return
	}

	limit := request.Limit
	if limit == 0 {
		limit = defaultMessagesLimit
	}

	messages, err := this.Services.Chats.GetMessages(r.Context(), request.ChatUID, limit, request.BeforeMessageUID, request.AfterMessageUID)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chat.GetChatMessagesResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	response := chat.GetChatMessagesResponse{Messages: make([]chat.Message, 0, len(messages))}
	for i := range messages {
		foundMessage := &messages[i]
		response.Messages = append(response.Messages, chat.Message{
			UserUID:    foundMessage.UserUID,
			ChatUID:    foundMessage.ChatUID,
			MessageUID: foundMessage.MessageUID,
			CreatedAt:  foundMessage.CreatedAt,
			Text:       foundMessage.Text,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *ChatsHandler) CreateJoinLinkHandler(w http.ResponseWriter, r *http.Request) {
	var request chat.CreateJoinLinkRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	if err := this.ensureChatMember(r.Context(), session.UID, request.ChatUID); err != nil {
		middleware.WriteJSON(w, chatsErrorStatus(err), chat.CreateJoinLinkResponse{
			Error: &shared_api.Error{Field: "chat_uid", Message: err.Error()},
		})
		return
	}

	link := this.Services.Chats.CreateJoinLink(request.ChatUID, request.LifetimeSeconds, request.MaxUses)

	middleware.WriteJSON(w, http.StatusOK, chat.CreateJoinLinkResponse{
		Token:     link.Token,
		ExpiresAt: link.ExpiresAt,
	})
}

func (this *ChatsHandler) JoinChatHandler(w http.ResponseWriter, r *http.Request) {
	var request chat.JoinChatRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	chatUID, err := this.Services.Chats.JoinChatByLink(r.Context(), request.Token, session.UID)
	if err != nil {
		middleware.WriteJSON(w, chatsErrorStatus(err), chat.JoinChatResponse{
			Error: &shared_api.Error{Field: "join_token", Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chat.JoinChatResponse{ChatUID: chatUID})
}

