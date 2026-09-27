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
	if err := messageRow.Scan(&message.MessageUID, &message.UserUID, &message.ChatUID, &message.CreatedAt, &message.Text); err != nil {
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

// GetMessages returns messages of a chat ordered chronologically (oldest first).
//
// If afterMessageUID is not zero, up to limit messages with uid > afterMessageUID
// are returned (forward pagination). Otherwise the most recent messages are
// returned; if beforeMessageUID is not zero only messages with uid < it are
// considered (backward pagination), otherwise the very last messages are used.
func (this *ChatsService) GetMessages(ctx context.Context, chatUID shared.UID, limit uint, beforeMessageUID shared.UID, afterMessageUID shared.UID) ([]Message, error) {
	const columns = "uid, user_uid, chat_uid, created_at, text"

	var (
		query string
		args  []any
	)
	switch {
	case afterMessageUID != 0:
		// Forward pagination: take the oldest messages after the cursor. Ascending
		// order already matches the oldest-first order returned to the caller.
		query = "select " + columns + " from messages where chat_uid = $1 and uid > $2 order by uid asc limit $3"
		args = []any{chatUID, afterMessageUID, limit}
	case beforeMessageUID != 0:
		// Backward pagination: the inner query orders descending to pick the page
		// closest to the cursor, the outer query flips it back so the caller always
		// receives messages oldest-first.
		query = "select " + columns + " from (select " + columns + " from messages where chat_uid = $1 and uid < $2 order by uid desc limit $3) as page order by uid asc"
		args = []any{chatUID, beforeMessageUID, limit}
	default:
		// No cursor: the very last messages, again returned oldest-first.
		query = "select " + columns + " from (select " + columns + " from messages where chat_uid = $1 order by uid desc limit $2) as page order by uid asc"
		args = []any{chatUID, limit}
	}

	messagesRows, err := this.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer messagesRows.Close()

	messages := make([]Message, 0, limit)
	for messagesRows.Next() {
		var message Message
		if err := messagesRows.Scan(&message.MessageUID, &message.UserUID, &message.ChatUID, &message.CreatedAt, &message.Text); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := messagesRows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
