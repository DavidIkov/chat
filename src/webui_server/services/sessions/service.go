package sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	neturl "net/url"
	"strings"

	"chat/src/shared"
	"chat/src/shared/api/user"
)

// Create makes a new WebUI session with a fresh random SessionID (e.g. 16 random
// bytes hex-encoded, matching how api_server builds its own tokens).
func (this *SessionsService) Create(ctx context.Context) (*WebUISession, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return nil, err
	}
	session := &WebUISession{ID: SessionID(hex.EncodeToString(buffer))}

	this.mu.Lock()
	defer this.mu.Unlock()
	this.sessions[session.ID] = session
	return session, nil
}

// Get returns the session with the given id, if it still exists.
func (this *SessionsService) Get(id SessionID) (*WebUISession, bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	session, ok := this.sessions[id]
	return session, ok
}

// Delete removes a session and everything it holds.
func (this *SessionsService) Delete(id SessionID) {
	this.mu.Lock()
	defer this.mu.Unlock()
	delete(this.sessions, id)
}

// AddServer validates url, appends a new ServerConnection with the next ServerID,
// and returns it. name may be empty, in which case callers should display the URL.
func (this *WebUISession) AddServer(url string, name string) (*ServerConnection, error) {
	parsed, err := neturl.Parse(url)
	if err != nil {
		return nil, InvalidServerURLError
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, InvalidServerURLError
	}
	if parsed.Host == "" {
		return nil, InvalidServerURLError
	}

	this.mu.Lock()
	defer this.mu.Unlock()
	this.nextID++
	connection := &ServerConnection{
		ID:   this.nextID,
		URL:  strings.TrimRight(url, "/"),
		Name: name,
	}
	this.Servers = append(this.Servers, connection)
	return connection, nil
}

// ServerByID returns the connection with the given id, if present.
func (this *WebUISession) ServerByID(id ServerID) (*ServerConnection, bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	for _, connection := range this.Servers {
		if connection.ID == id {
			return connection, true
		}
	}
	return nil, false
}

// RemoveServer drops a connection (and its stored credentials) from the session.
func (this *WebUISession) RemoveServer(id ServerID) {
	this.mu.Lock()
	defer this.mu.Unlock()
	for index, connection := range this.Servers {
		if connection.ID == id {
			this.Servers = append(this.Servers[:index], this.Servers[index+1:]...)
			return
		}
	}
}

// ListServers returns a copy of the session's connections so callers can read
// them without holding the session lock.
func (this *WebUISession) ListServers() []*ServerConnection {
	this.mu.RLock()
	defer this.mu.RUnlock()
	servers := make([]*ServerConnection, len(this.Servers))
	copy(servers, this.Servers)
	return servers
}

// SetPendingJoinLink remembers a freshly created join token for one chat so the
// next render of that chat page can show it once (see TakePendingJoinLink).
func (this *WebUISession) SetPendingJoinLink(serverID ServerID, chatUID shared.UID, token string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.pendingJoinLink = &pendingJoinLink{serverID: serverID, chatUID: chatUID, token: token}
}

// TakePendingJoinLink returns and clears the pending join token for the given
// chat, or "" when there is none. Tokens belonging to another connection or chat
// are left untouched so that navigating elsewhere does not lose them.
func (this *WebUISession) TakePendingJoinLink(serverID ServerID, chatUID shared.UID) string {
	this.mu.Lock()
	defer this.mu.Unlock()
	pending := this.pendingJoinLink
	if pending == nil || pending.serverID != serverID || pending.chatUID != chatUID {
		return ""
	}
	this.pendingJoinLink = nil
	return pending.token
}

// SetServerSession stores the api_server user session returned by login/register
// on the given connection.
func (this *WebUISession) SetServerSession(id ServerID, session *user.UserSession) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	for _, connection := range this.Servers {
		if connection.ID == id {
			connection.Session = session
			return nil
		}
	}
	return ServerNotFoundError
}

// ClearServerSession forgets the api_server user session on the given connection,
// e.g. after logout.
func (this *WebUISession) ClearServerSession(id ServerID) error {
	return this.SetServerSession(id, nil)
}
