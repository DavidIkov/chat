package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"chat/internal/shared"
	chatapi "chat/internal/shared/api/chat"
)

// CreateChat creates a chat and returns its uid. Maps to POST /chat/create.
func (this *Client) CreateChat(ctx context.Context, baseURL string, token string, name string) (shared.UID, error) {
	request := chatapi.CreateChatRequest{Name: name}
	var response chatapi.CreateChatResponse
	if err := this.do(ctx, http.MethodPost, baseURL, chatCreatePath, nil, token, request, &response); err != nil {
		return 0, err
	}
	if response.Error != nil {
		return 0, asAPIError(http.StatusOK, response.Error)
	}
	return response.ChatUID, nil
}

// GetChats lists chats. When uids is empty it returns the chats the caller
// belongs to. Maps to GET /chat/get_chats?uids=...
func (this *Client) GetChats(ctx context.Context, baseURL string, token string, uids []shared.UID) ([]chatapi.Chat, error) {
	// Only send uids when the caller asked for specific chats: an empty uids
	// query must be omitted so the api_server returns every chat the caller
	// belongs to (D8).
	var values url.Values
	if len(uids) > 0 {
		values = url.Values{}
		for _, uid := range uids {
			values.Add("uids", strconv.FormatUint(uint64(uid), 10))
		}
	}
	var response chatapi.GetChatsResponse
	if err := this.do(ctx, http.MethodGet, baseURL, chatGetChatsPath, values, token, nil, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return response.Chats, nil
}

// GetChatMessages returns messages of a chat. limit==0 means the api_server
// default. before/after are optional cursors (0 = unset).
// Maps to GET /chat/{uid}/get_messages.
func (this *Client) GetChatMessages(ctx context.Context, baseURL string, token string, chatUID shared.UID, limit uint, beforeMessageUID shared.UID, afterMessageUID shared.UID) ([]chatapi.Message, error) {
	values := url.Values{}
	if limit != 0 {
		values.Add("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if beforeMessageUID != 0 {
		values.Add("before_message_uid", strconv.FormatUint(uint64(beforeMessageUID), 10))
	}
	if afterMessageUID != 0 {
		values.Add("after_message_uid", strconv.FormatUint(uint64(afterMessageUID), 10))
	}
	var response chatapi.GetChatMessagesResponse
	if err := this.do(ctx, http.MethodGet, baseURL, fmt.Sprintf(chatGetMessagesPath, chatUID), values, token, nil, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return response.Messages, nil
}

// SendMessage posts a message and returns its uid.
// Maps to POST /chat/{uid}/send_message.
func (this *Client) SendMessage(ctx context.Context, baseURL string, token string, chatUID shared.UID, text string) (shared.UID, error) {
	request := chatapi.SendMessageRequest{Text: text}
	var response chatapi.SendMessageResponse
	if err := this.do(ctx, http.MethodPost, baseURL, fmt.Sprintf(chatSendMessagePath, chatUID), nil, token, request, &response); err != nil {
		return 0, err
	}
	if response.Error != nil {
		return 0, asAPIError(http.StatusOK, response.Error)
	}
	return response.MessageUID, nil
}

// GetChatMembers lists the members of a chat.
// Maps to GET /chat/{uid}/get_members.
func (this *Client) GetChatMembers(ctx context.Context, baseURL string, token string, chatUID shared.UID) ([]chatapi.Member, error) {
	var response chatapi.GetChatMembersResponse
	if err := this.do(ctx, http.MethodGet, baseURL, fmt.Sprintf(chatGetMembersPath, chatUID), nil, token, nil, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return response.Members, nil
}

// CreateJoinLink creates a join token for a chat.
// Maps to POST /chat/{uid}/create_join_link.
func (this *Client) CreateJoinLink(ctx context.Context, baseURL string, token string, chatUID shared.UID, lifetimeSeconds int64, maxUses uint) (*chatapi.CreateJoinLinkResponse, error) {
	request := chatapi.CreateJoinLinkRequest{LifetimeSeconds: lifetimeSeconds, MaxUses: maxUses}
	var response chatapi.CreateJoinLinkResponse
	if err := this.do(ctx, http.MethodPost, baseURL, fmt.Sprintf(chatCreateJoinLinkPath, chatUID), nil, token, request, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return &response, nil
}

// JoinChat joins a chat by join token and returns the chat uid.
// Maps to POST /chat/join_chat.
func (this *Client) JoinChat(ctx context.Context, baseURL string, token string, joinToken string) (shared.UID, error) {
	request := chatapi.JoinChatRequest{Token: joinToken}
	var response chatapi.JoinChatResponse
	if err := this.do(ctx, http.MethodPost, baseURL, chatJoinChatPath, nil, token, request, &response); err != nil {
		return 0, err
	}
	if response.Error != nil {
		return 0, asAPIError(http.StatusOK, response.Error)
	}
	return response.ChatUID, nil
}

// LeaveChat removes the caller from a chat. Maps to POST /chat/{uid}/leave.
func (this *Client) LeaveChat(ctx context.Context, baseURL string, token string, chatUID shared.UID) error {
	var response chatapi.LeaveChatResponse
	if err := this.do(ctx, http.MethodPost, baseURL, fmt.Sprintf(chatLeavePath, chatUID), nil, token, nil, &response); err != nil {
		return err
	}
	if response.Error != nil {
		return asAPIError(http.StatusOK, response.Error)
	}
	return nil
}
