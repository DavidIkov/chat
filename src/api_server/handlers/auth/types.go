package auth

import (
	"chat/src/shared"
)

// contextKey is unexported so that no other package can collide with the value
// stored by this package.
type contextKey struct{}

// Session is the caller authenticated from the "Authorization" header. It keeps
// the raw token as well as the resolved user id: logging out removes the session
// behind the token, while the rest of the handlers need the user id.
type Session struct {
	Token string
	UID   shared.UID
}
