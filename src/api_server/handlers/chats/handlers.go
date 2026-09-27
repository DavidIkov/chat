package chats

import (
	chatservice "chat/src/api_server/services/chats"
	userservice "chat/src/api_server/services/users"
	"chat/src/shared"
	"chat/src/shared_api"
	"chat/src/shared_api/chat"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-playground/form/v4"
	"net/http"
	"strings"
)

var queryDecoder = form.NewDecoder()

// getUserUIDFromRequest resolves the user behind the "Authorization: Bearer <token>"
// header, returning userservice.TokenNotFoundError when it is missing or unknown.
func (this *ChatsHandler) getUserUIDFromRequest(r *http.Request) (shared.UID, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return 0, userservice.TokenNotFoundError
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	if token == "" {
		return 0, userservice.TokenNotFoundError
	}
	return this.Services.Users.GetUserUIDByToken(token)
}

func authStatus(err error) int {
	switch {
	case errors.Is(err, userservice.TokenNotFoundError):
		return http.StatusUnauthorized
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request chat.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response chat.CreateChatResponse

	userUID, err := this.getUserUIDFromRequest(r)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	createdChat, err := this.Services.Chats.CreateChat(r.Context(), request.Name, userUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &shared_api.Error{Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	response.ChatUID = createdChat.ChatUID

	json.NewEncoder(w).Encode(response)
}

func (this *ChatsHandler) SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request chat.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response chat.SendMessageResponse

	userUID, err := this.getUserUIDFromRequest(r)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := this.ensureChatMember(r.Context(), userUID, request.ChatUID); err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "chat_uid", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	message, err := this.Services.Chats.SendMessage(r.Context(), request.ChatUID, userUID, request.Text)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &shared_api.Error{Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	response.MessageUID = message.MessageUID

	json.NewEncoder(w).Encode(response)
}

func (this *ChatsHandler) GetChatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request chat.GetChatsRequest
	if err := queryDecoder.Decode(&request, r.URL.Query()); err != nil {
		http.Error(w, "invalid query params", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response chat.GetChatsResponse

	userUID, err := this.getUserUIDFromRequest(r)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	// GetChats only returns chats the acting user is a member of.
	chats, err := this.Services.Chats.GetChats(r.Context(), userUID, request.UIDs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &shared_api.Error{Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Chats = make([]chat.Chat, 0, len(chats))
	for i := range chats {
		foundChat := &chats[i]
		response.Chats = append(response.Chats, chat.Chat{
			ChatUID:        foundChat.ChatUID,
			CreatorUserUID: foundChat.CreatorUserUID,
			CreatedAt:      foundChat.CreatedAt,
			Name:           foundChat.Name,
		})
	}

	json.NewEncoder(w).Encode(response)
}

func (this *ChatsHandler) GetChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request chat.GetChatMessagesRequest
	if err := queryDecoder.Decode(&request, r.URL.Query()); err != nil {
		http.Error(w, "invalid query params", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response chat.GetChatMessagesResponse

	userUID, err := this.getUserUIDFromRequest(r)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := this.ensureChatMember(r.Context(), userUID, request.ChatUID); err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "chat_uid", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	limit := request.Limit
	if limit == 0 {
		limit = defaultMessagesLimit
	}

	messages, err := this.Services.Chats.GetMessages(r.Context(), request.ChatUID, limit, request.BeforeMessageUID, request.AfterMessageUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &shared_api.Error{Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Messages = make([]chat.Message, 0, len(messages))
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

	json.NewEncoder(w).Encode(response)
}

func (this *ChatsHandler) CreateJoinLinkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request chat.CreateJoinLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response chat.CreateJoinLinkResponse

	userUID, err := this.getUserUIDFromRequest(r)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := this.ensureChatMember(r.Context(), userUID, request.ChatUID); err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "chat_uid", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	link := this.Services.Chats.CreateJoinLink(request.ChatUID, request.LifetimeSeconds, request.MaxUses)

	response.Token = link.Token
	response.ExpiresAt = link.ExpiresAt

	json.NewEncoder(w).Encode(response)
}

func (this *ChatsHandler) JoinChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request chat.JoinChatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response chat.JoinChatResponse

	userUID, err := this.getUserUIDFromRequest(r)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	chatUID, err := this.Services.Chats.JoinChatByLink(r.Context(), request.Token, userUID)
	if err != nil {
		w.WriteHeader(authStatus(err))
		response.Error = &shared_api.Error{Field: "join_token", Message: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	response.ChatUID = chatUID

	json.NewEncoder(w).Encode(response)
}
