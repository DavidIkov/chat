package chats

import (
	"chat/src/shared"
	"context"
	"github.com/lib/pq"
	"time"
)

func (this *ChatsService) CreateChat(ctx context.Context, chatName string, userUID shared.UID) (*Chat, error) {
	curTime := shared.Time(time.Now().UnixMilli())
	chatRow := this.db.QueryRowContext(ctx, "insert into chats (name, creator_user_uid,created_at) values ($1, $2, $3) returning uid, name, creator_user_uid, created_at", chatName, userUID, curTime)
	var chat Chat
	if err := chatRow.Scan(&chat.ChatUID, &chat.Name, &chat.CreatorUserUID, &chat.CreatedAt); err != nil {
		return nil, err
	}
	return &chat, nil
}

func (this *ChatsService) SendMessage(ctx context.Context, chatUID shared.UID, userUID shared.UID, text string) (*Message, error) {
	curTime := shared.Time(time.Now().UnixMilli())
	messageRow := this.db.QueryRowContext(ctx, "insert into messages (user_uid, chat_uid,created_at, text) values ($1, $2, $3, $4) returning uid, user_uid, chat_uid, created_at, text", userUID, chatUID, curTime, text)
	var message Message
	if err := messageRow.Scan(&message.UserUID, &message.ChatUID, &message.CreatedAt, &message.Text); err != nil {
		return nil, err
	}
	return &message, nil

}

func (this *ChatsService) GetChats(ctx context.Context, uids []shared.UID) ([]Chat, error) {
	chatsRows, err := this.db.QueryContext(ctx, "select uid, name, creator_user_uid, created_at from chats where uid = any($1)", pq.Array(uids))
	if err != nil {
		return nil, err
	}
	defer chatsRows.Close()

	chats := make([]Chat, 0, len(uids))
	for chatsRows.Next() {
		var chat Chat
		if err := chatsRows.Scan(&chat.ChatUID, &chat.Name, &chat.CreatorUserUID, &chat.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, nil
}

func (this *ChatsService) GetMessages(ctx context.Context, chatUID shared.UID, targetMessageUID shared.UID, filterType MessageCursorDirection, limit uint) ([]Message, error) {
	var comparisonOperator string
	switch {
	case filterType == MessagesBefore:
		comparisonOperator = "<"
	case filterType == MessagesAfter:
		comparisonOperator = ">"
	default:
		return nil, UnknownMessageFilterTypeByUID
	}

	messagesRows, err := this.db.QueryContext(ctx, "select uid, user_uid, chat_uid, created_at, text from messages where chat_uid = $1 and message_uid "+comparisonOperator+" $2 limit $3", chatUID, targetMessageUID, limit)
	if err != nil {
		return nil, err
	}
	defer messagesRows.Close()

	messages := make([]Message, 0, limit)
	for messagesRows.Next() {
		var message Message
		if err := messagesRows.Scan(&message.UserUID, &message.ChatUID, &message.CreatedAt, &message.Text); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil

}
