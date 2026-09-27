package chats

import (
	"chat/src/shared"
	"context"
	"github.com/lib/pq"
	"time"
)

func (this *ChatsService) CreateChat(ctx context.Context, chatName string, userUID shared.UID) (*Chat, error) {
	curTime := shared.Time(time.Now().UnixMilli())

	tx, err := this.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	chatRow := tx.QueryRowContext(ctx, "insert into chats (name, creator_user_uid, created_at) values ($1, $2, $3) returning uid, name, creator_user_uid, created_at", chatName, userUID, curTime)
	var chat Chat
	if err := chatRow.Scan(&chat.ChatUID, &chat.Name, &chat.CreatorUserUID, &chat.CreatedAt); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, "insert into chat_members (chat_uid, user_uid) values ($1, $2)", chat.ChatUID, userUID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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

func (this *ChatsService) GetChats(ctx context.Context, userUID shared.UID, uids []shared.UID) ([]Chat, error) {
	chatsRows, err := this.db.QueryContext(ctx, "select c.uid, c.name, c.creator_user_uid, c.created_at from chats c join chat_members m on m.chat_uid = c.uid where m.user_uid = $1 and c.uid = any($2)", userUID, pq.Array(uids))
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
	return chats, chatsRows.Err()
}

func (this *ChatsService) IsChatMember(ctx context.Context, chatUID shared.UID, userUID shared.UID) (bool, error) {
	var isMember bool
	err := this.db.QueryRowContext(ctx, "select exists (select 1 from chat_members where chat_uid = $1 and user_uid = $2)", chatUID, userUID).Scan(&isMember)
	if err != nil {
		return false, err
	}
	return isMember, nil
}

// JoinChat adds userUID as a member of chatUID. It reports whether a new
// membership was created: re-joining is a no-op returning false, so a limited
// invite is not consumed when the user is already a member.
func (this *ChatsService) JoinChat(ctx context.Context, chatUID shared.UID, userUID shared.UID) (bool, error) {
	result, err := this.db.ExecContext(ctx, "insert into chat_members (chat_uid, user_uid) values ($1, $2) on conflict do nothing", chatUID, userUID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
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

// GetChatMembers returns the members of chatUID ordered by uid so that the
// output is stable.
func (this *ChatsService) GetChatMembers(ctx context.Context, chatUID shared.UID) ([]Member, error) {
	membersRows, err := this.db.QueryContext(ctx, "select user_uid from chat_members where chat_uid = $1 order by user_uid", chatUID)
	if err != nil {
		return nil, err
	}
	defer membersRows.Close()

	members := make([]Member, 0)
	for membersRows.Next() {
		var member Member
		if err := membersRows.Scan(&member.UserUID); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, membersRows.Err()
}

// LeaveChat removes userUID from chatUID. A missing membership row is a no-op.
// When the last member leaves, the chat is deleted: its messages go first for
// foreign-key ordering, then the chat itself, and finally every in-memory join
// link pointing at it is purged so that a stale link cannot rejoin a deleted
// chat.
func (this *ChatsService) LeaveChat(ctx context.Context, chatUID shared.UID, userUID shared.UID) error {
	tx, err := this.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "delete from chat_members where chat_uid = $1 and user_uid = $2", chatUID, userUID); err != nil {
		return err
	}

	var remainingMembers int
	if err := tx.QueryRowContext(ctx, "select count(*) from chat_members where chat_uid = $1", chatUID).Scan(&remainingMembers); err != nil {
		return err
	}

	if remainingMembers == 0 {
		if _, err := tx.ExecContext(ctx, "delete from messages where chat_uid = $1", chatUID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "delete from chats where uid = $1", chatUID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if remainingMembers == 0 {
		this.joinLinks.removeChatLinks(chatUID)
	}

	return nil
}
