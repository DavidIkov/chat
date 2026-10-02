package apitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"chat/internal/api_server/handlers"
	"chat/internal/api_server/services"
	"chat/internal/shared"
	chatapi "chat/internal/shared/api/chat"
	userapi "chat/internal/shared/api/user"
)

// env is a freshly wired api_server serving real HTTP requests over the shared
// throwaway database. Each env gets its own service layer (and therefore its own
// in-memory sessions and join links) and the tables are emptied before it is
// built, so every test starts from a clean, deterministic (serial 1, 2, ...)
// state.
type env struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client
	svc    *services.Services
}

func newEnv(t *testing.T) *env {
	t.Helper()

	svc := newServices(t)
	handler, err := handlers.NewServer(svc)
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &env{t: t, server: server, client: server.Client(), svc: svc}
}

// newServices resets the database and returns a fresh service layer.
func newServices(t *testing.T) *services.Services {
	t.Helper()

	if testing.Short() {
		t.Skip("database-backed test skipped in -short mode")
	}
	if testDB == nil {
		t.Fatal("test database is not initialised (did TestMain run?)")
	}

	resetDB(t)

	svc, err := services.CreateServices(testDB)
	if err != nil {
		t.Fatalf("create services: %v", err)
	}
	return svc
}

func resetDB(t *testing.T) {
	t.Helper()

	if _, err := testDB.Exec("truncate users, chats, messages, chat_members restart identity cascade"); err != nil {
		t.Fatalf("reset database: %v", err)
	}
}

// do sends a JSON request (body may be nil) and returns the status code and raw
// response body.
func (e *env) do(method, path, token string, body any) (int, []byte) {
	e.t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal request body: %v", err)
		}
	}
	return e.send(method, path, token, payload, body != nil)
}

// doRaw sends a request with an already-encoded body, used to exercise malformed
// JSON.
func (e *env) doRaw(method, path, token, body string) (int, []byte) {
	e.t.Helper()
	return e.send(method, path, token, []byte(body), body != "")
}

func (e *env) send(method, path, token string, body []byte, hasBody bool) (int, []byte) {
	e.t.Helper()

	var reader io.Reader
	if hasBody {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		e.t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, data
}

// decode unmarshals a response body, failing the test on malformed JSON.
func decode[T any](t *testing.T, data []byte) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode response %q: %v", data, err)
	}
	return value
}

func (e *env) requireStatus(want, got int, body any) {
	e.t.Helper()
	if got != want {
		e.t.Fatalf("status = %d, want %d (body: %v)", got, want, body)
	}
}

// --- Typed endpoint helpers -------------------------------------------------

func (e *env) register(name, password string) (int, userapi.UserRegistrationResponse) {
	status, data := e.do(http.MethodPost, "/user/register", "",
		userapi.UserRegistrationRequest{Name: name, Password: password})
	return status, decode[userapi.UserRegistrationResponse](e.t, data)
}

func (e *env) mustRegister(name, password string) userapi.UserSession {
	e.t.Helper()

	status, resp := e.register(name, password)
	if status != http.StatusOK || resp.UserSession == nil {
		e.t.Fatalf("register %q: status = %d, body = %s", name, status, mustJSON(resp))
	}
	return *resp.UserSession
}

func (e *env) login(name, password string) (int, userapi.UserLogInResponse) {
	status, data := e.do(http.MethodPost, "/user/login", "",
		userapi.UserLogInRequest{Name: name, Password: password})
	return status, decode[userapi.UserLogInResponse](e.t, data)
}

func (e *env) logout(token string) (int, userapi.UserLogOutResponse) {
	status, data := e.do(http.MethodPost, "/user/logout", token, nil)
	return status, decode[userapi.UserLogOutResponse](e.t, data)
}

func (e *env) deleteUser(token string, deleteMessages bool) (int, userapi.UserDeleteResponse) {
	status, data := e.do(http.MethodPost, "/user/delete", token,
		userapi.UserDeleteRequest{DeleteMessages: deleteMessages})
	return status, decode[userapi.UserDeleteResponse](e.t, data)
}

func (e *env) getUsers(token string, uids []uint) (int, userapi.UsersGetResponse) {
	values := url.Values{}
	for _, uid := range uids {
		values.Add("uids", strconv.FormatUint(uint64(uid), 10))
	}
	status, data := e.do(http.MethodGet, "/user/get?"+values.Encode(), token, nil)
	return status, decode[userapi.UsersGetResponse](e.t, data)
}

func (e *env) createChat(token, name string) (int, chatapi.CreateChatResponse) {
	status, data := e.do(http.MethodPost, "/chat/create", token,
		chatapi.CreateChatRequest{Name: name})
	return status, decode[chatapi.CreateChatResponse](e.t, data)
}

func (e *env) mustCreateChat(token, name string) shared.UID {
	e.t.Helper()

	status, resp := e.createChat(token, name)
	if status != http.StatusOK || resp.ChatUID == 0 {
		e.t.Fatalf("create chat %q: status = %d, body = %s", name, status, mustJSON(resp))
	}
	return resp.ChatUID
}

func (e *env) getChats(token string, uids []shared.UID) (int, chatapi.GetChatsResponse) {
	path := "/chat/get_chats"
	if len(uids) > 0 {
		values := url.Values{}
		for _, uid := range uids {
			values.Add("uids", strconv.FormatUint(uint64(uid), 10))
		}
		path += "?" + values.Encode()
	}
	status, data := e.do(http.MethodGet, path, token, nil)
	return status, decode[chatapi.GetChatsResponse](e.t, data)
}

func (e *env) sendMessage(token string, chatUID shared.UID, text string) (int, chatapi.SendMessageResponse) {
	status, data := e.do(http.MethodPost, chatPath(chatUID, "send_message"), token,
		chatapi.SendMessageRequest{Text: text})
	return status, decode[chatapi.SendMessageResponse](e.t, data)
}

func (e *env) mustSendMessage(token string, chatUID shared.UID, text string) shared.UID {
	e.t.Helper()

	status, resp := e.sendMessage(token, chatUID, text)
	if status != http.StatusOK || resp.MessageUID == 0 {
		e.t.Fatalf("send message: status = %d, body = %s", status, mustJSON(resp))
	}
	return resp.MessageUID
}

func (e *env) getMessages(token string, chatUID shared.UID, query url.Values) (int, chatapi.GetChatMessagesResponse) {
	path := chatPath(chatUID, "get_messages")
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	status, data := e.do(http.MethodGet, path, token, nil)
	return status, decode[chatapi.GetChatMessagesResponse](e.t, data)
}

func (e *env) getMembers(token string, chatUID shared.UID) (int, chatapi.GetChatMembersResponse) {
	status, data := e.do(http.MethodGet, chatPath(chatUID, "get_members"), token, nil)
	return status, decode[chatapi.GetChatMembersResponse](e.t, data)
}

func (e *env) createJoinLink(token string, chatUID shared.UID, lifetime int64, maxUses uint) (int, chatapi.CreateJoinLinkResponse) {
	status, data := e.do(http.MethodPost, chatPath(chatUID, "create_join_link"), token,
		chatapi.CreateJoinLinkRequest{LifetimeSeconds: lifetime, MaxUses: maxUses})
	return status, decode[chatapi.CreateJoinLinkResponse](e.t, data)
}

func (e *env) joinChat(token, joinToken string) (int, chatapi.JoinChatResponse) {
	status, data := e.do(http.MethodPost, "/chat/join_chat", token,
		chatapi.JoinChatRequest{Token: joinToken})
	return status, decode[chatapi.JoinChatResponse](e.t, data)
}

func (e *env) leaveChat(token string, chatUID shared.UID) (int, chatapi.LeaveChatResponse) {
	status, data := e.do(http.MethodPost, chatPath(chatUID, "leave"), token, nil)
	return status, decode[chatapi.LeaveChatResponse](e.t, data)
}

func chatPath(chatUID shared.UID, action string) string {
	return "/chat/" + strconv.FormatUint(uint64(chatUID), 10) + "/" + action
}

func mustJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
