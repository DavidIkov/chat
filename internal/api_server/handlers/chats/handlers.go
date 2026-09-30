package chats

import (
	"chat/internal/api_server/handlers/auth"
	chatsmiddleware "chat/internal/api_server/handlers/chats/middleware"
	"chat/internal/api_server/handlers/middleware"
	chatservice "chat/internal/api_server/services/chats"
	"chat/internal/shared/api"
	chatapi "chat/internal/shared/api/chat"
	"chat/internal/shared/api/validator"
	"net/http"
)

func (this *ChatsHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	var request chatapi.CreateChatRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	name, name_error := validator.ValidateChatName(request.Name)
	if name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, chatapi.CreateChatResponse{
			Error: name_error,
		})
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	createdChat, err := this.Services.Chats.CreateChat(r.Context(), name, session.UID)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chatapi.CreateChatResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chatapi.CreateChatResponse{ChatUID: createdChat.ChatUID})
}

func (this *ChatsHandler) SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	var request chatapi.SendMessageRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	text, text_error := validator.ValidateMessageText(request.Text)
	if text_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, chatapi.SendMessageResponse{
			Error: text_error,
		})
		return
	}

	session, _ := auth.SessionFromContext(r.Context())
	chatUID, _ := chatsmiddleware.ChatUIDFromContext(r.Context())

	message, err := this.Services.Chats.SendMessage(r.Context(), chatUID, session.UID, text)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chatapi.SendMessageResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chatapi.SendMessageResponse{MessageUID: message.MessageUID})
}

func (this *ChatsHandler) GetChatsHandler(w http.ResponseWriter, r *http.Request) {
	var request chatapi.GetChatsRequest
	if !middleware.DecodeQuery(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	// No uids provided: the caller wants every chat it belongs to.
	var chats []chatservice.Chat
	var err error
	if len(request.UIDs) == 0 {
		chats, err = this.Services.Chats.GetUserChats(r.Context(), session.UID)
	} else {
		chats, err = this.Services.Chats.GetChats(r.Context(), session.UID, request.UIDs)
	}
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chatapi.GetChatsResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	response := chatapi.GetChatsResponse{Chats: make([]chatapi.Chat, 0, len(chats))}
	for i := range chats {
		foundChat := &chats[i]
		response.Chats = append(response.Chats, chatapi.Chat{
			ChatUID:        foundChat.ChatUID,
			CreatorUserUID: foundChat.CreatorUserUID,
			CreatedAt:      foundChat.CreatedAt,
			Name:           foundChat.Name,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *ChatsHandler) GetChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	var request chatapi.GetChatMessagesRequest
	if !middleware.DecodeQuery(w, r, &request) {
		return
	}

	chatUID, _ := chatsmiddleware.ChatUIDFromContext(r.Context())

	limit := request.Limit
	if limit == 0 {
		limit = defaultMessagesLimit
	}

	messages, err := this.Services.Chats.GetMessages(r.Context(), chatUID, limit, request.BeforeMessageUID, request.AfterMessageUID)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chatapi.GetChatMessagesResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	response := chatapi.GetChatMessagesResponse{Messages: make([]chatapi.Message, 0, len(messages))}
	for i := range messages {
		foundMessage := &messages[i]
		response.Messages = append(response.Messages, chatapi.Message{
			UserUID:    foundMessage.UserUID,
			ChatUID:    foundMessage.ChatUID,
			MessageUID: foundMessage.MessageUID,
			CreatedAt:  foundMessage.CreatedAt,
			Text:       foundMessage.Text,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *ChatsHandler) GetChatMembersHandler(w http.ResponseWriter, r *http.Request) {
	chatUID, _ := chatsmiddleware.ChatUIDFromContext(r.Context())

	members, err := this.Services.Chats.GetChatMembers(r.Context(), chatUID)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, chatapi.GetChatMembersResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	response := chatapi.GetChatMembersResponse{Members: make([]chatapi.Member, 0, len(members))}
	for i := range members {
		member := &members[i]
		response.Members = append(response.Members, chatapi.Member{UserUID: member.UserUID})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *ChatsHandler) CreateJoinLinkHandler(w http.ResponseWriter, r *http.Request) {
	var request chatapi.CreateJoinLinkRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	chatUID, _ := chatsmiddleware.ChatUIDFromContext(r.Context())

	link := this.Services.Chats.CreateJoinLink(chatUID, request.LifetimeSeconds, request.MaxUses)

	middleware.WriteJSON(w, http.StatusOK, chatapi.CreateJoinLinkResponse{
		Token:     link.Token,
		ExpiresAt: link.ExpiresAt,
	})
}

func (this *ChatsHandler) LeaveChatHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	chatUID, _ := chatsmiddleware.ChatUIDFromContext(r.Context())

	if err := this.Services.Chats.LeaveChat(r.Context(), chatUID, session.UID); err != nil {
		middleware.WriteJSON(w, chatsmiddleware.ChatsErrorStatus(err), chatapi.LeaveChatResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chatapi.LeaveChatResponse{})
}

func (this *ChatsHandler) JoinChatHandler(w http.ResponseWriter, r *http.Request) {
	var request chatapi.JoinChatRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	session, _ := auth.SessionFromContext(r.Context())

	chatUID, err := this.Services.Chats.JoinChatByLink(r.Context(), request.Token, session.UID)
	if err != nil {
		middleware.WriteJSON(w, chatsmiddleware.ChatsErrorStatus(err), chatapi.JoinChatResponse{
			Error: &api.Error{Field: "join_token", Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, chatapi.JoinChatResponse{ChatUID: chatUID})
}
