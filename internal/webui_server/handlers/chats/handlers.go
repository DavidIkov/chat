package chats

import (
	"net/http"
	"net/url"
	"strconv"

	"chat/internal/shared"
	chatapi "chat/internal/shared/api/chat"
	"chat/internal/webui_server/handlers/middleware"
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/handlers/routes"
	"chat/internal/webui_server/services/sessions"
)

// serverFromRequest resolves the connection named by the {server_id} wildcard of
// r inside session. It reports false when the value is missing, malformed, or
// does not belong to the session.
func serverFromRequest(r *http.Request, session *sessions.WebUISession) (*sessions.ServerConnection, bool) {
	parsed, err := strconv.ParseUint(r.PathValue(routes.ServerIDPathKey), 10, 32)
	if err != nil || parsed == 0 {
		return nil, false
	}
	return session.ServerByID(sessions.ServerID(parsed))
}

// chatUIDFromRequest parses the {chat_uid} wildcard of r into an api_server chat
// uid. It reports false when the value is missing, malformed, or zero.
func chatUIDFromRequest(r *http.Request) (shared.UID, bool) {
	parsed, err := strconv.ParseUint(r.PathValue(routes.ChatUIDPathKey), 10, 32)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return shared.UID(parsed), true
}

// chatListPath builds the canonical URL of a connection's chat list page.
func chatListPath(connection *sessions.ServerConnection) string {
	return "/servers/" + strconv.FormatUint(uint64(connection.ID), 10) + "/chats"
}

// chatPagePath builds the canonical URL of one chat's page on a connection.
func chatPagePath(connection *sessions.ServerConnection, chatUID shared.UID) string {
	return chatListPath(connection) + "/" + strconv.FormatUint(uint64(chatUID), 10)
}

// renderChatsPage renders chats.html for one connection, listing chats and
// showing message as a form error.
func (this *ChatsHandler) renderChatsPage(w http.ResponseWriter, status int, connection *sessions.ServerConnection, chats []chatapi.Chat, message string) {
	this.Templates.Render(w, status, chatsTemplateName, ChatsPage{
		Page:   render.Page{Title: "Chats"},
		Server: connection,
		Chats:  chats,
		Error:  message,
	})
}

// renderChatPage renders chat.html for one chat: its metadata, messages and
// members, the send form, the join-link form and the leave button. It is shared
// by the GET chat page and the join-link POST so both show the same data (and,
// for the join link, the freshly generated token).
func (this *ChatsHandler) renderChatPage(w http.ResponseWriter, r *http.Request, status int, connection *sessions.ServerConnection, chatUID shared.UID, joinLink string, message string) {
	token := connection.Session.Token

	// Chat metadata (name) via the uids filter. A backend failure is a 5xx; an
	// empty result means the chat does not exist or the caller is not a member
	// (the api_server filters by membership), which is a 404.
	chats, err := this.Services.API.GetChats(r.Context(), connection.URL, token, []shared.UID{chatUID})
	if err != nil {
		this.Templates.RenderError(w, http.StatusBadGateway, err.Error())
		return
	}
	if len(chats) == 0 {
		this.Templates.RenderError(w, http.StatusNotFound, "chat not found")
		return
	}
	chat := chats[0]

	messages, err := this.Services.API.GetChatMessages(r.Context(), connection.URL, token, chatUID, 0, 0, 0)
	if err != nil {
		this.Templates.RenderError(w, http.StatusBadGateway, err.Error())
		return
	}

	members, err := this.Services.API.GetChatMembers(r.Context(), connection.URL, token, chatUID)
	if err != nil {
		this.Templates.RenderError(w, http.StatusBadGateway, err.Error())
		return
	}

	uids := make([]uint, 0, len(members))
	for _, member := range members {
		uids = append(uids, uint(member.UserUID))
	}
	names := map[shared.UID]string{}
	if len(uids) > 0 {
		if users, err := this.Services.API.GetUsers(r.Context(), connection.URL, token, uids); err == nil {
			for _, user := range users {
				names[shared.UID(user.UID)] = user.Name
			}
		}
	}

	messageItems := make([]MessageItem, 0, len(messages))
	for _, m := range messages {
		messageItems = append(messageItems, MessageItem{Message: m, UserName: names[m.UserUID]})
	}
	memberItems := make([]MemberItem, 0, len(members))
	for _, m := range members {
		memberItems = append(memberItems, MemberItem{UID: m.UserUID, Name: names[m.UserUID]})
	}

	this.Templates.Render(w, status, chatTemplateName, ChatPage{
		Page:     render.Page{Title: chat.Name},
		Server:   connection,
		Chat:     chat,
		Messages: messageItems,
		Members:  memberItems,
		JoinLink: joinLink,
		Error:    message,
	})
}

// ChatsPageHandler renders the chats of the signed-in user on one server, with
// the create/join forms.
//
// Route: GET /servers/{server_id}/chats
func (this *ChatsHandler) ChatsPageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	// nil uids means "every chat the caller belongs to" (D8).
	chats, err := this.Services.API.GetChats(r.Context(), connection.URL, connection.Session.Token, nil)
	if err != nil {
		this.renderChatsPage(w, http.StatusBadGateway, connection, nil, err.Error())
		return
	}

	this.renderChatsPage(w, http.StatusOK, connection, chats, "")
}

// CreateChatHandler creates a chat from the "name" form field and redirects to
// its chat page.
//
// Route: POST /servers/{server_id}/chats
func (this *ChatsHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	_ = r.ParseForm()
	chatUID, err := this.Services.API.CreateChat(r.Context(), connection.URL, connection.Session.Token, r.FormValue("name"))
	if err != nil {
		this.renderChatsPage(w, http.StatusUnprocessableEntity, connection, this.chatsOrNil(r, connection), err.Error())
		return
	}

	render.Redirect(w, r, chatPagePath(connection, chatUID))
}

// JoinChatHandler joins a chat by the "token" form field and redirects to its
// chat page. On failure the api_server error (invalid/expired token) is shown on
// the chats page.
//
// Route: POST /servers/{server_id}/chats/join
func (this *ChatsHandler) JoinChatHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	_ = r.ParseForm()
	chatUID, err := this.Services.API.JoinChat(r.Context(), connection.URL, connection.Session.Token, r.FormValue("token"))
	if err != nil {
		this.renderChatsPage(w, http.StatusUnprocessableEntity, connection, this.chatsOrNil(r, connection), err.Error())
		return
	}

	render.Redirect(w, r, chatPagePath(connection, chatUID))
}

// chatsOrNil re-fetches the caller's chat list for an error re-render, falling
// back to nil when the re-fetch itself fails so the form error is still shown.
func (this *ChatsHandler) chatsOrNil(r *http.Request, connection *sessions.ServerConnection) []chatapi.Chat {
	chats, err := this.Services.API.GetChats(r.Context(), connection.URL, connection.Session.Token, nil)
	if err != nil {
		return nil
	}
	return chats
}

// ChatPageHandler renders one chat: its messages and members, the send form, the
// join-link form and the leave button. A join token minted by the preceding
// join-link POST is shown here once (it is held in the session between the POST
// and this GET).
//
// Route: GET /servers/{server_id}/chats/{chat_uid}
func (this *ChatsHandler) ChatPageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	chatUID, ok := chatUIDFromRequest(r)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "chat not found")
		return
	}

	// Show a just-created join token once; consuming it here (on the GET that
	// follows the join-link POST) is what makes reloading that page a safe GET.
	joinLink := session.TakePendingJoinLink(connection.ID, chatUID)

	this.renderChatPage(w, r, http.StatusOK, connection, chatUID, joinLink, "")
}

// SendMessageHandler posts the "text" form field to the chat and redirects back
// to the chat page (Post/Redirect/Get), so the reload shows the new message.
//
// Route: POST /servers/{server_id}/chats/{chat_uid}/messages
func (this *ChatsHandler) SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	chatUID, ok := chatUIDFromRequest(r)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "chat not found")
		return
	}

	_ = r.ParseForm()
	if _, err := this.Services.API.SendMessage(r.Context(), connection.URL, connection.Session.Token, chatUID, r.FormValue("text")); err != nil {
		this.renderChatPage(w, r, http.StatusUnprocessableEntity, connection, chatUID, "", err.Error())
		return
	}

	render.Redirect(w, r, chatPagePath(connection, chatUID))
}

// CreateJoinLinkHandler creates a join token from the join-link form, stashes it
// on the session and redirects to the chat page, where ChatPageHandler shows it
// once. The redirect (Post/Redirect/Get) ensures reloading the page does not
// resubmit the form and mint another token. The "limit_lifetime" and "limit_max_uses"
// checkboxes opt into the "lifetime_seconds" and "max_uses" fields; when a box is
// clear its field is ignored and 0 is sent, which the api_server reads as "no
// limit". A ticked box with a missing or non-positive value is rejected so the
// link is never accidentally unlimited.
//
// Route: POST /servers/{server_id}/chats/{chat_uid}/join_link
func (this *ChatsHandler) CreateJoinLinkHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	chatUID, ok := chatUIDFromRequest(r)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "chat not found")
		return
	}

	_ = r.ParseForm()

	lifetime, lifetimeError := parseJoinLinkLifetime(r.Form)
	if lifetimeError != "" {
		this.renderChatPage(w, r, http.StatusUnprocessableEntity, connection, chatUID, "", lifetimeError)
		return
	}

	maxUses, maxUsesError := parseJoinLinkMaxUses(r.Form)
	if maxUsesError != "" {
		this.renderChatPage(w, r, http.StatusUnprocessableEntity, connection, chatUID, "", maxUsesError)
		return
	}

	result, err := this.Services.API.CreateJoinLink(r.Context(), connection.URL, connection.Session.Token, chatUID, lifetime, maxUses)
	if err != nil {
		this.renderChatPage(w, r, http.StatusUnprocessableEntity, connection, chatUID, "", err.Error())
		return
	}

	// Hand the token to the redirect target instead of rendering it here; see the
	// handler comment for why.
	session.SetPendingJoinLink(connection.ID, chatUID, result.Token)
	render.Redirect(w, r, chatPagePath(connection, chatUID))
}

// parseJoinLinkLifetime reads the optional lifetime of the join-link form. The
// "limit_lifetime" checkbox turns the "lifetime_seconds" field on; when it is
// clear the field is ignored and 0 (never expires) is returned. A ticked box with
// a missing or non-positive value is an error.
func parseJoinLinkLifetime(form url.Values) (int64, string) {
	if !form.Has("limit_lifetime") {
		return 0, ""
	}
	lifetime, err := strconv.ParseInt(form.Get("lifetime_seconds"), 10, 64)
	if err != nil || lifetime <= 0 {
		return 0, "Lifetime must be a positive number of seconds."
	}
	return lifetime, ""
}

// parseJoinLinkMaxUses mirrors parseJoinLinkLifetime for the "limit_max_uses"
// checkbox and the "max_uses" field; 0 (unlimited) is returned when it is clear.
func parseJoinLinkMaxUses(form url.Values) (uint, string) {
	if !form.Has("limit_max_uses") {
		return 0, ""
	}
	maxUses, err := strconv.ParseUint(form.Get("max_uses"), 10, 32)
	if err != nil || maxUses == 0 {
		return 0, "Max uses must be a positive number."
	}
	return uint(maxUses), ""
}

// LeaveChatHandler removes the user from the chat and returns to the chat list.
//
// Route: POST /servers/{server_id}/chats/{chat_uid}/leave
func (this *ChatsHandler) LeaveChatHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	connection, ok := serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, "/servers/"+strconv.FormatUint(uint64(connection.ID), 10))
		return
	}

	chatUID, ok := chatUIDFromRequest(r)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "chat not found")
		return
	}

	if err := this.Services.API.LeaveChat(r.Context(), connection.URL, connection.Session.Token, chatUID); err != nil {
		this.Templates.RenderError(w, http.StatusBadGateway, err.Error())
		return
	}

	render.Redirect(w, r, chatListPath(connection))
}
