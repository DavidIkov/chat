package servers

import (
	"net/http"
	"strconv"
	"strings"

	"chat/internal/webui_server/handlers/middleware"
	"chat/internal/webui_server/handlers/render"
	"chat/internal/webui_server/handlers/routes"
	"chat/internal/webui_server/services/sessions"
)

// serverFromRequest resolves the connection named by the {server_id} wildcard of
// r inside session. It reports false when the value is missing, malformed, or
// does not belong to the session.
func (this *ServersHandler) serverFromRequest(r *http.Request, session *sessions.WebUISession) (*sessions.ServerConnection, bool) {
	parsed, err := strconv.ParseUint(r.PathValue(routes.ServerIDPathKey), 10, 32)
	if err != nil || parsed == 0 {
		return nil, false
	}
	return session.ServerByID(sessions.ServerID(parsed))
}

// serverPath builds the canonical URL of a connection's page.
func serverPath(id sessions.ServerID) string {
	return "/servers/" + strconv.FormatUint(uint64(id), 10)
}

// renderServerPage renders server.html for one connection, using the name (or
// URL) as the page title, showing notice as a success banner and message as a
// form error.
func (this *ServersHandler) renderServerPage(w http.ResponseWriter, status int, connection *sessions.ServerConnection, notice string, message string) {
	title := connection.Name
	if title == "" {
		title = connection.URL
	}
	this.Templates.Render(w, status, serverTemplateName, ServerPage{
		Page:     render.Page{Title: title},
		Server:   connection,
		SignedIn: connection.Session != nil,
		Notice:   notice,
		Error:    message,
	})
}

// IndexHandler renders the landing page: the list of connected api servers and
// the "add server" form.
//
// Route: GET /
func (this *ServersHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	this.Templates.Render(w, http.StatusOK, indexTemplateName, IndexPage{
		Page:    render.Page{Title: "Servers"},
		Servers: session.ListServers(),
	})
}

// AddServerHandler adds a connection from the form fields "url" (required) and
// "name" (optional) and redirects to that server's page.
//
// Route: POST /servers
func (this *ServersHandler) AddServerHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())
	_ = r.ParseForm()

	connection, err := session.AddServer(strings.TrimSpace(r.FormValue("url")), strings.TrimSpace(r.FormValue("name")))
	if err != nil {
		this.Templates.Render(w, http.StatusUnprocessableEntity, indexTemplateName, IndexPage{
			Page:    render.Page{Title: "Servers"},
			Servers: session.ListServers(),
			Error:   err.Error(),
		})
		return
	}

	render.Redirect(w, r, serverPath(connection.ID))
}

// ServerPageHandler renders one connection: the sign-in (login/register) forms
// when signed out, or the signed-in summary and a link to the chats page when
// signed in.
//
// Route: GET /servers/{server_id}
func (this *ServersHandler) ServerPageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())

	connection, ok := this.serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}

	notice := ""
	if r.URL.Query().Has("deleted") {
		notice = "The account was deleted on this server."
	}

	this.renderServerPage(w, http.StatusOK, connection, notice, "")
}

// RemoveServerHandler drops a connection from the session and returns to the
// landing page. It does not touch the api_server.
//
// Route: POST /servers/{server_id}/remove
func (this *ServersHandler) RemoveServerHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())

	connection, ok := this.serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}

	session.RemoveServer(connection.ID)
	render.Redirect(w, r, "/")
}

// LogInHandler signs in on one connection from the "name"/"password" form and
// stores the returned session on that connection.
//
// Route: POST /servers/{server_id}/login
func (this *ServersHandler) LogInHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())

	connection, ok := this.serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}

	_ = r.ParseForm()
	userSession, err := this.Services.API.LogIn(r.Context(), connection.URL, r.FormValue("name"), r.FormValue("password"))
	if err != nil {
		this.renderServerPage(w, http.StatusUnauthorized, connection, "", err.Error())
		return
	}

	if err := session.SetServerSession(connection.ID, userSession); err != nil {
		this.Templates.RenderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Redirect(w, r, serverPath(connection.ID))
}

// RegisterHandler registers a new account on one connection from the
// "name"/"password" form and stores the returned session on that connection.
//
// Route: POST /servers/{server_id}/register
func (this *ServersHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())

	connection, ok := this.serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}

	_ = r.ParseForm()
	userSession, err := this.Services.API.Register(r.Context(), connection.URL, r.FormValue("name"), r.FormValue("password"))
	if err != nil {
		this.renderServerPage(w, http.StatusUnprocessableEntity, connection, "", err.Error())
		return
	}

	if err := session.SetServerSession(connection.ID, userSession); err != nil {
		this.Templates.RenderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Redirect(w, r, serverPath(connection.ID))
}

// LogOutHandler logs out on one connection (invalidating the token on the
// api_server) and clears the stored session.
//
// Route: POST /servers/{server_id}/logout
func (this *ServersHandler) LogOutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())

	connection, ok := this.serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}

	if connection.Session != nil {
		_ = this.Services.API.LogOut(r.Context(), connection.URL, connection.Session.Token)
	}
	_ = session.ClearServerSession(connection.ID)

	render.Redirect(w, r, serverPath(connection.ID))
}

// DeleteUserHandler permanently deletes the signed-in account on one connection.
// The "delete_messages" checkbox decides whether the account's messages are also
// removed from every chat. Because the account and its tokens no longer exist
// afterwards, the local session is cleared and the connection returns to the
// signed-out state.
//
// Route: POST /servers/{server_id}/delete
func (this *ServersHandler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFromContext(r.Context())

	connection, ok := this.serverFromRequest(r, session)
	if !ok {
		this.Templates.RenderError(w, http.StatusNotFound, "server not found")
		return
	}
	if connection.Session == nil {
		render.Redirect(w, r, serverPath(connection.ID))
		return
	}

	_ = r.ParseForm()
	deleteMessages := r.Form.Has("delete_messages")

	if err := this.Services.API.DeleteUser(r.Context(), connection.URL, connection.Session.Token, deleteMessages); err != nil {
		this.renderServerPage(w, http.StatusUnprocessableEntity, connection, "", err.Error())
		return
	}

	// The account and its token are gone on the api_server, so forget the local
	// session too rather than pretending to stay signed in.
	_ = session.ClearServerSession(connection.ID)

	render.Redirect(w, r, serverPath(connection.ID)+"?deleted=1")
}
