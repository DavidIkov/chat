package apitest

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"chat/internal/shared"
	chatapi "chat/internal/shared/api/chat"
)

const (
	testUserName = "alice"
	testPassword = "secret"
	otherName    = "bobby"
)

// TestRegisterLoginLogout walks the account lifecycle over HTTP: register,
// duplicate registration, failed and successful login, and token invalidation on
// logout.
func TestRegisterLoginLogout(t *testing.T) {
	e := newEnv(t)

	status, reg := e.register(testUserName, testPassword)
	e.requireStatus(http.StatusOK, status, mustJSON(reg))
	if reg.UserSession == nil || reg.UserSession.UID == 0 || reg.UserSession.Token == "" {
		t.Fatalf("register returned an empty session: %s", mustJSON(reg))
	}
	session := *reg.UserSession

	// Re-registering the same name is a conflict, not a second account.
	status, dup := e.register(testUserName, testPassword)
	e.requireStatus(http.StatusConflict, status, mustJSON(dup))
	if dup.Error == nil || dup.Error.Field != "name" {
		t.Fatalf("duplicate register error = %s, want field \"name\"", mustJSON(dup))
	}

	// A wrong password is unauthorized.
	status, badLogin := e.login(testUserName, "wrong-password")
	e.requireStatus(http.StatusUnauthorized, status, mustJSON(badLogin))
	if badLogin.Error == nil {
		t.Fatalf("bad login returned no error: %s", mustJSON(badLogin))
	}

	// A correct login resolves to the same account.
	status, login := e.login(testUserName, testPassword)
	e.requireStatus(http.StatusOK, status, mustJSON(login))
	if login.UserSession == nil || login.UserSession.UID != session.UID {
		t.Fatalf("login = %s, want uid %d", mustJSON(login), session.UID)
	}

	// Logging out invalidates the token.
	status, logout := e.logout(session.Token)
	e.requireStatus(http.StatusOK, status, mustJSON(logout))
	if status, _ := e.getChats(session.Token, nil); status != http.StatusUnauthorized {
		t.Fatalf("logged-out token still works: status = %d, want 401", status)
	}
}

// TestRegisterValidation checks that out-of-range names and passwords are
// rejected with a per-field 422.
func TestRegisterValidation(t *testing.T) {
	e := newEnv(t)

	cases := []struct {
		name     string
		user     string
		password string
		field    string
	}{
		{"short name", "ab", testPassword, "name"},
		{"long name", "this-name-is-way-too-long", testPassword, "name"},
		{"short password", testUserName, "ab", "password"},
		{"long password", testUserName, "this-password-is-way-too-long", "password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp := e.register(tc.user, tc.password)
			e.requireStatus(http.StatusUnprocessableEntity, status, mustJSON(resp))
			if resp.Error == nil || resp.Error.Field != tc.field {
				t.Fatalf("error = %s, want field %q", mustJSON(resp), tc.field)
			}
		})
	}
}

func TestMalformedJSONIsRejected(t *testing.T) {
	e := newEnv(t)

	status, _ := e.doRaw(http.MethodPost, "/user/register", "", "{not json")
	e.requireStatus(http.StatusBadRequest, status, nil)
}

// TestProtectedEndpointsRequireToken checks that every authenticated route
// answers 401 without (or with an unknown) bearer token.
func TestProtectedEndpointsRequireToken(t *testing.T) {
	e := newEnv(t)

	protected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/user/logout"},
		{http.MethodPost, "/user/delete"},
		{http.MethodGet, "/user/get?uids=1"},
		{http.MethodPost, "/chat/create"},
		{http.MethodGet, "/chat/get_chats"},
		{http.MethodPost, "/chat/join_chat"},
	}
	for _, p := range protected {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			status, _ := e.do(p.method, p.path, "", nil)
			e.requireStatus(http.StatusUnauthorized, status, nil)
		})
	}

	t.Run("unknown token", func(t *testing.T) {
		status, _ := e.do(http.MethodPost, "/chat/create", "not-a-real-token", struct{}{})
		e.requireStatus(http.StatusUnauthorized, status, nil)
	})
}

// TestCreateAndGetChats covers chat creation and the "my chats" / uid-filtered
// listing behaviour, including that chats are only visible to members.
func TestCreateAndGetChats(t *testing.T) {
	e := newEnv(t)

	alice := e.mustRegister(testUserName, testPassword)
	bob := e.mustRegister(otherName, testPassword)

	chatUID := e.mustCreateChat(alice.Token, "weekend plans")
	bobChatUID := e.mustCreateChat(bob.Token, "bob private")

	// No uids: every chat the caller belongs to.
	status, all := e.getChats(alice.Token, nil)
	e.requireStatus(http.StatusOK, status, mustJSON(all))
	if len(all.Chats) != 1 || all.Chats[0].ChatUID != chatUID {
		t.Fatalf("alice chats = %s, want only chat %d", mustJSON(all), chatUID)
	}
	if all.Chats[0].Name != "weekend plans" || all.Chats[0].CreatorUserUID != shared.UID(alice.UID) {
		t.Fatalf("unexpected chat metadata: %s", mustJSON(all.Chats[0]))
	}

	// Filtering by uid returns only the requested chats the caller belongs to.
	status, filtered := e.getChats(alice.Token, []shared.UID{chatUID, bobChatUID})
	e.requireStatus(http.StatusOK, status, mustJSON(filtered))
	if len(filtered.Chats) != 1 || filtered.Chats[0].ChatUID != chatUID {
		t.Fatalf("filtered chats = %s, want only chat %d", mustJSON(filtered), chatUID)
	}
}

// TestSendAndGetMessagesWithPagination exercises message sending plus the three
// pagination modes (no cursor, before cursor, after cursor).
func TestSendAndGetMessagesWithPagination(t *testing.T) {
	e := newEnv(t)

	alice := e.mustRegister(testUserName, testPassword)
	chatUID := e.mustCreateChat(alice.Token, "chat")

	texts := []string{"one", "two", "three", "four", "five"}
	uids := make([]shared.UID, 0, len(texts))
	for _, text := range texts {
		uids = append(uids, e.mustSendMessage(alice.Token, chatUID, text))
	}

	// No cursor: everything, oldest first.
	status, all := e.getMessages(alice.Token, chatUID, nil)
	e.requireStatus(http.StatusOK, status, mustJSON(all))
	assertMessageUIDs(t, all.Messages, uids)

	// limit only: the newest page.
	_, newest := e.getMessages(alice.Token, chatUID, url.Values{"limit": {"2"}})
	assertMessageUIDs(t, newest.Messages, uids[3:])

	// before cursor: the page just older than the cursor.
	_, before := e.getMessages(alice.Token, chatUID, url.Values{
		"limit":              {"2"},
		"before_message_uid": {strconv.FormatUint(uint64(uids[3]), 10)},
	})
	assertMessageUIDs(t, before.Messages, uids[1:3])

	// after cursor: the page just newer than the cursor.
	_, after := e.getMessages(alice.Token, chatUID, url.Values{
		"limit":             {"2"},
		"after_message_uid": {strconv.FormatUint(uint64(uids[2]), 10)},
	})
	assertMessageUIDs(t, after.Messages, uids[3:])
}

// TestChatAccessControl covers validation and membership enforcement.
func TestChatAccessControl(t *testing.T) {
	e := newEnv(t)

	alice := e.mustRegister(testUserName, testPassword)
	bob := e.mustRegister(otherName, testPassword)
	chatUID := e.mustCreateChat(alice.Token, "chat")

	// Whitespace-only text trims to empty and is rejected.
	status, bad := e.sendMessage(alice.Token, chatUID, "   ")
	e.requireStatus(http.StatusUnprocessableEntity, status, mustJSON(bad))
	if bad.Error == nil || bad.Error.Field != "text" {
		t.Fatalf("empty message error = %s, want field \"text\"", mustJSON(bad))
	}

	// A non-member can neither read nor write the chat.
	status, _ = e.sendMessage(bob.Token, chatUID, "hello")
	e.requireStatus(http.StatusForbidden, status, nil)
	status, _ = e.getMessages(bob.Token, chatUID, nil)
	e.requireStatus(http.StatusForbidden, status, nil)
	status, _ = e.getMembers(bob.Token, chatUID)
	e.requireStatus(http.StatusForbidden, status, nil)
	status, _ = e.leaveChat(bob.Token, chatUID)
	e.requireStatus(http.StatusForbidden, status, nil)

	// A non-numeric chat uid in the path is a bad request.
	status, _ = e.do(http.MethodGet, "/chat/not-a-number/get_messages", alice.Token, nil)
	e.requireStatus(http.StatusBadRequest, status, nil)
}

// TestJoinLinkWorkflow covers invite creation, invalid tokens, the creator not
// consuming a use by rejoining, max_uses accounting, and membership afterwards.
func TestJoinLinkWorkflow(t *testing.T) {
	e := newEnv(t)

	alice := e.mustRegister(testUserName, testPassword)
	bob := e.mustRegister(otherName, testPassword)
	carol := e.mustRegister("carol", testPassword)
	chatUID := e.mustCreateChat(alice.Token, "chat")

	status, link := e.createJoinLink(alice.Token, chatUID, 0, 1)
	e.requireStatus(http.StatusOK, status, mustJSON(link))
	if link.Token == "" {
		t.Fatalf("create join link returned no token: %s", mustJSON(link))
	}

	// Unknown tokens are forbidden.
	status, invalid := e.joinChat(bob.Token, "nope")
	e.requireStatus(http.StatusForbidden, status, mustJSON(invalid))

	// The creator rejoining its own chat is a no-op and must not spend the
	// single use.
	status, rejoin := e.joinChat(alice.Token, link.Token)
	e.requireStatus(http.StatusOK, status, mustJSON(rejoin))
	if rejoin.ChatUID != chatUID {
		t.Fatalf("rejoin chat_uid = %d, want %d", rejoin.ChatUID, chatUID)
	}

	// Bob spends the only use.
	status, joined := e.joinChat(bob.Token, link.Token)
	e.requireStatus(http.StatusOK, status, mustJSON(joined))

	// Carol cannot: the link is used up.
	status, _ = e.joinChat(carol.Token, link.Token)
	e.requireStatus(http.StatusForbidden, status, nil)

	// Bob is now a member and can post.
	status, members := e.getMembers(alice.Token, chatUID)
	e.requireStatus(http.StatusOK, status, mustJSON(members))
	assertMemberUIDs(t, members.Members, []shared.UID{shared.UID(alice.UID), shared.UID(bob.UID)})

	status, sent := e.sendMessage(bob.Token, chatUID, "hi from bob")
	e.requireStatus(http.StatusOK, status, mustJSON(sent))
}

// TestLeaveChat checks that leaving keeps the chat alive while members remain
// and cascades a delete once the last member leaves.
func TestLeaveChat(t *testing.T) {
	e := newEnv(t)

	alice := e.mustRegister(testUserName, testPassword)
	bob := e.mustRegister(otherName, testPassword)
	chatUID := e.mustCreateChat(alice.Token, "chat")
	e.mustSendMessage(alice.Token, chatUID, "hello")

	_, link := e.createJoinLink(alice.Token, chatUID, 0, 0)
	if status, joined := e.joinChat(bob.Token, link.Token); status != http.StatusOK {
		t.Fatalf("bob join: status = %d, body = %s", status, mustJSON(joined))
	}

	// Bob leaves: the chat and its messages survive for alice.
	status, left := e.leaveChat(bob.Token, chatUID)
	e.requireStatus(http.StatusOK, status, mustJSON(left))
	if status, _ := e.getMessages(bob.Token, chatUID, nil); status != http.StatusForbidden {
		t.Fatalf("bob still has access: status = %d, want 403", status)
	}
	if status, chats := e.getChats(alice.Token, nil); status != http.StatusOK || len(chats.Chats) != 1 {
		t.Fatalf("alice chats after bob left = %s", mustJSON(chats))
	}
	if _, msgs := e.getMessages(alice.Token, chatUID, nil); len(msgs.Messages) != 1 {
		t.Fatalf("messages after bob left = %s, want 1", mustJSON(msgs))
	}

	// Alice, now the last member, leaves: chat and messages are deleted.
	status, left = e.leaveChat(alice.Token, chatUID)
	e.requireStatus(http.StatusOK, status, mustJSON(left))
	if _, chats := e.getChats(alice.Token, nil); len(chats.Chats) != 0 {
		t.Fatalf("chats after last member left = %s, want none", mustJSON(chats))
	}
	if chats, messages := rowCount(t, "chats", chatUID), messageCount(t, chatUID); chats != 0 || messages != 0 {
		t.Fatalf("chat rows = %d, message rows = %d, want 0 and 0", chats, messages)
	}
}

// TestDeleteUser checks both account-deletion modes: removing the user's
// messages, and keeping them (their author becomes anonymous).
func TestDeleteUser(t *testing.T) {
	t.Run("delete messages", func(t *testing.T) {
		e := newEnv(t)

		alice := e.mustRegister(testUserName, testPassword)
		bob := e.mustRegister(otherName, testPassword)
		chatUID := e.mustCreateChat(alice.Token, "chat")
		_, link := e.createJoinLink(alice.Token, chatUID, 0, 0)
		e.joinChat(bob.Token, link.Token)
		e.mustSendMessage(alice.Token, chatUID, "alice says hi")
		e.mustSendMessage(bob.Token, chatUID, "bob says hi")

		status, resp := e.deleteUser(alice.Token, true)
		e.requireStatus(http.StatusOK, status, mustJSON(resp))

		if status, _ := e.getChats(alice.Token, nil); status != http.StatusUnauthorized {
			t.Fatalf("deleted user's token still works: status = %d, want 401", status)
		}

		status, msgs := e.getMessages(bob.Token, chatUID, nil)
		e.requireStatus(http.StatusOK, status, mustJSON(msgs))
		if len(msgs.Messages) != 1 || msgs.Messages[0].Text != "bob says hi" {
			t.Fatalf("messages after delete = %s, want only bob's", mustJSON(msgs))
		}
	})

	t.Run("keep messages", func(t *testing.T) {
		e := newEnv(t)

		alice := e.mustRegister(testUserName, testPassword)
		bob := e.mustRegister(otherName, testPassword)
		chatUID := e.mustCreateChat(alice.Token, "chat")
		_, link := e.createJoinLink(alice.Token, chatUID, 0, 0)
		e.joinChat(bob.Token, link.Token)
		e.mustSendMessage(alice.Token, chatUID, "alice says hi")
		e.mustSendMessage(bob.Token, chatUID, "bob says hi")

		status, resp := e.deleteUser(alice.Token, false)
		e.requireStatus(http.StatusOK, status, mustJSON(resp))

		status, msgs := e.getMessages(bob.Token, chatUID, nil)
		e.requireStatus(http.StatusOK, status, mustJSON(msgs))
		if len(msgs.Messages) != 2 {
			t.Fatalf("messages after delete = %s, want both kept", mustJSON(msgs))
		}

		// Alice's message survives but is no longer attributed to her.
		var aliceMessage, bobMessage *chatapi.Message
		for i := range msgs.Messages {
			switch msgs.Messages[i].Text {
			case "alice says hi":
				aliceMessage = &msgs.Messages[i]
			case "bob says hi":
				bobMessage = &msgs.Messages[i]
			}
		}
		if aliceMessage == nil || bobMessage == nil {
			t.Fatalf("messages after delete = %s", mustJSON(msgs))
		}
		if aliceMessage.UserUID != 0 {
			t.Fatalf("orphaned message user_uid = %d, want 0", aliceMessage.UserUID)
		}
		if bobMessage.UserUID != shared.UID(bob.UID) {
			t.Fatalf("bob's message user_uid = %d, want %d", bobMessage.UserUID, bob.UID)
		}
	})
}

func TestGetUsers(t *testing.T) {
	e := newEnv(t)

	alice := e.mustRegister(testUserName, testPassword)
	bob := e.mustRegister(otherName, testPassword)

	status, resp := e.getUsers(alice.Token, []uint{uint(alice.UID), uint(bob.UID), 9999})
	e.requireStatus(http.StatusOK, status, mustJSON(resp))
	if len(resp.Users) != 2 {
		t.Fatalf("users = %s, want 2", mustJSON(resp))
	}

	names := map[uint]string{}
	for _, u := range resp.Users {
		names[u.UID] = u.Name
	}
	if names[uint(alice.UID)] != testUserName || names[uint(bob.UID)] != otherName {
		t.Fatalf("users = %s", mustJSON(resp))
	}
}

// --- Assertions -------------------------------------------------------------

func assertMessageUIDs(t *testing.T, got []chatapi.Message, want []shared.UID) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d messages %v, want %d %v", len(got), messageUIDs(got), len(want), want)
	}
	for i := range want {
		if got[i].MessageUID != want[i] {
			t.Fatalf("message[%d].uid = %d, want %d (got %v)", i, got[i].MessageUID, want[i], messageUIDs(got))
		}
	}
}

func assertMemberUIDs(t *testing.T, got []chatapi.Member, want []shared.UID) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d members %v, want %d %v", len(got), memberUIDs(got), len(want), want)
	}
	for i := range want {
		if got[i].UserUID != want[i] {
			t.Fatalf("member[%d].uid = %d, want %d (got %v)", i, got[i].UserUID, want[i], memberUIDs(got))
		}
	}
}

func messageUIDs(messages []chatapi.Message) []shared.UID {
	uids := make([]shared.UID, 0, len(messages))
	for _, m := range messages {
		uids = append(uids, m.MessageUID)
	}
	return uids
}

func memberUIDs(members []chatapi.Member) []shared.UID {
	uids := make([]shared.UID, 0, len(members))
	for _, m := range members {
		uids = append(uids, m.UserUID)
	}
	return uids
}

func rowCount(t *testing.T, table string, chatUID shared.UID) int {
	t.Helper()

	if table != "chats" { // keep the query static; only chats is ever asked for
		t.Fatalf("rowCount: unsupported table %q", table)
	}
	var count int
	if err := testDB.QueryRow("select count(*) from chats where uid = $1", chatUID).Scan(&count); err != nil {
		t.Fatalf("count chats: %v", err)
	}
	return count
}

func messageCount(t *testing.T, chatUID shared.UID) int {
	t.Helper()

	var count int
	if err := testDB.QueryRow("select count(*) from messages where chat_uid = $1", chatUID).Scan(&count); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	return count
}
