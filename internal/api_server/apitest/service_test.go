package apitest

import (
	"context"
	"errors"
	"testing"
	"time"

	"chat/internal/api_server/services"
	chatservice "chat/internal/api_server/services/chats"
	userservice "chat/internal/api_server/services/users"
	"chat/internal/shared"
)

// These tests call the service layer directly. They cover business rules that
// are slow or awkward to drive end-to-end over HTTP (time-based expiry, the
// exact use-counting of join links, cursor pagination, cascade deletes and
// session bookkeeping).

func TestServiceMessagePagination(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice := registerService(t, svc, testUserName, testPassword)
	chat, err := svc.Chats.CreateChat(ctx, "chat", shared.UID(alice.UID))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}

	var uids []shared.UID
	for _, text := range []string{"one", "two", "three", "four", "five"} {
		message, err := svc.Chats.SendMessage(ctx, chat.ChatUID, shared.UID(alice.UID), text)
		if err != nil {
			t.Fatalf("send message %q: %v", text, err)
		}
		uids = append(uids, message.MessageUID)
	}

	// No cursor, limit 2: the newest page.
	got := mustGetMessages(t, svc, chat.ChatUID, 2, 0, 0)
	assertServiceMessageUIDs(t, got, uids[3:])

	// before cursor.
	got = mustGetMessages(t, svc, chat.ChatUID, 2, uids[3], 0)
	assertServiceMessageUIDs(t, got, uids[1:3])

	// after cursor.
	got = mustGetMessages(t, svc, chat.ChatUID, 2, 0, uids[2])
	assertServiceMessageUIDs(t, got, uids[3:])
}

// TestServiceJoinLinkUsesAndRejoin pins down the use-counting rule: a distinct
// user joining spends a use, rejoining without creating a membership does not,
// and the link dies once its last use is spent.
func TestServiceJoinLinkUsesAndRejoin(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice := registerService(t, svc, testUserName, testPassword)
	bob := registerService(t, svc, otherName, testPassword)
	carol := registerService(t, svc, "carol", testPassword)
	dave := registerService(t, svc, "dave", testPassword)

	chat, err := svc.Chats.CreateChat(ctx, "chat", shared.UID(alice.UID))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}
	link := svc.Chats.CreateJoinLink(chat.ChatUID, 0, 2)

	// The creator rejoining does not spend a use.
	if _, err := svc.Chats.JoinChatByLink(ctx, link.Token, shared.UID(alice.UID)); err != nil {
		t.Fatalf("creator rejoin: %v", err)
	}
	// Two distinct users spend both uses.
	if _, err := svc.Chats.JoinChatByLink(ctx, link.Token, shared.UID(bob.UID)); err != nil {
		t.Fatalf("bob join: %v", err)
	}
	if _, err := svc.Chats.JoinChatByLink(ctx, link.Token, shared.UID(carol.UID)); err != nil {
		t.Fatalf("carol join: %v", err)
	}
	// The third distinct user finds the link consumed.
	if _, err := svc.Chats.JoinChatByLink(ctx, link.Token, shared.UID(dave.UID)); !errors.Is(err, chatservice.InvalidJoinLinkError) {
		t.Fatalf("dave join error = %v, want InvalidJoinLinkError", err)
	}
}

func TestServiceJoinLinkExpires(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice := registerService(t, svc, testUserName, testPassword)
	bob := registerService(t, svc, otherName, testPassword)

	chat, err := svc.Chats.CreateChat(ctx, "chat", shared.UID(alice.UID))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}
	link := svc.Chats.CreateJoinLink(chat.ChatUID, 1, 0)

	time.Sleep(1100 * time.Millisecond)

	if _, err := svc.Chats.JoinChatByLink(ctx, link.Token, shared.UID(bob.UID)); !errors.Is(err, chatservice.InvalidJoinLinkError) {
		t.Fatalf("expired link error = %v, want InvalidJoinLinkError", err)
	}
}

// TestServiceLeaveLastMemberDeletesChat verifies the cascade: when the last
// member leaves, the chat row and all of its messages are removed.
func TestServiceLeaveLastMemberDeletesChat(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice := registerService(t, svc, testUserName, testPassword)
	chat, err := svc.Chats.CreateChat(ctx, "chat", shared.UID(alice.UID))
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}
	if _, err := svc.Chats.SendMessage(ctx, chat.ChatUID, shared.UID(alice.UID), "hi"); err != nil {
		t.Fatalf("send message: %v", err)
	}

	if err := svc.Chats.LeaveChat(ctx, chat.ChatUID, shared.UID(alice.UID)); err != nil {
		t.Fatalf("leave chat: %v", err)
	}

	if chats, messages := rowCount(t, "chats", chat.ChatUID), messageCount(t, chat.ChatUID); chats != 0 || messages != 0 {
		t.Fatalf("chat rows = %d, message rows = %d, want 0 and 0", chats, messages)
	}
}

// TestServiceSessions covers registration conflicts and the session lifecycle.
func TestServiceSessions(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice, err := svc.Users.RegisterUser(ctx, testUserName, testPassword)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// The token resolves to the registered uid.
	if uid, err := svc.Users.GetUserUIDByToken(alice.Token); err != nil || uid != shared.UID(alice.UID) {
		t.Fatalf("resolve token = (%d, %v), want (%d, nil)", uid, err, alice.UID)
	}

	// The same name cannot be registered twice.
	if _, err := svc.Users.RegisterUser(ctx, testUserName, testPassword); !errors.Is(err, userservice.DuplicateUserNameError) {
		t.Fatalf("duplicate register error = %v, want DuplicateUserNameError", err)
	}

	// Logging in again reuses the existing session.
	again, err := svc.Users.LogInUser(ctx, testUserName, testPassword)
	if err != nil || again.Token != alice.Token {
		t.Fatalf("second login = (%+v, %v), want the original session", again, err)
	}

	// Logging out invalidates the token.
	if err := svc.Users.LogOutUser(ctx, alice.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Users.GetUserUIDByToken(alice.Token); !errors.Is(err, userservice.TokenNotFoundError) {
		t.Fatalf("resolve after logout error = %v, want TokenNotFoundError", err)
	}
}

// --- Helpers ----------------------------------------------------------------

func registerService(t *testing.T, svc *services.Services, name, password string) userservice.UserSession {
	t.Helper()

	session, err := svc.Users.RegisterUser(context.Background(), name, password)
	if err != nil {
		t.Fatalf("register %q: %v", name, err)
	}
	return session
}

func mustGetMessages(t *testing.T, svc *services.Services, chatUID shared.UID, limit uint, before, after shared.UID) []chatservice.Message {
	t.Helper()

	messages, err := svc.Chats.GetMessages(context.Background(), chatUID, limit, before, after)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	return messages
}

func assertServiceMessageUIDs(t *testing.T, got []chatservice.Message, want []shared.UID) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].MessageUID != want[i] {
			t.Fatalf("message[%d].uid = %d, want %d", i, got[i].MessageUID, want[i])
		}
	}
}
