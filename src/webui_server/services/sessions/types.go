package sessions

import (
	"chat/src/shared/api/user"
	"sync"
)

// SessionID identifies one browser's WebUI session. Its string value is what goes
// into the session cookie.
type SessionID string

// ServerID identifies one connected api_server inside a single WebUI session. It
// is stable for the lifetime of the session and is what appears in webui URLs as
// {server_id}. It is deliberately independent of any api_server uid.
type ServerID uint32

// ServerConnection is one api_server instance added by the user, together with
// the credentials of the user signed in to it (if any).
type ServerConnection struct {
	ID   ServerID
	URL  string
	Name string

	// Session is nil until the user signs in on this connection. Each connection
	// keeps its own session, so different connections can be different users
	// (even against the same api_server).
	Session *user.UserSession
}

// WebUISession is the server-side state for one browser. All access to Servers
// must go through the methods in service.go so the lock is always held.
type WebUISession struct {
	ID SessionID

	mu      sync.RWMutex
	nextID  ServerID
	Servers []*ServerConnection
}

// SessionsService is the in-memory store of WebUI sessions. It has no persistent
// backing: sessions disappear when the process stops.
type SessionsService struct {
	mu       sync.RWMutex
	sessions map[SessionID]*WebUISession
}
