package middleware

import (
	"chat/src/webui_server/services/sessions"
	"context"
	"net/http"
)

type contextKey struct{}

func withSession(ctx context.Context, session *sessions.WebUISession) context.Context {
	return context.WithValue(ctx, contextKey{}, session)
}

// SessionFromContext returns the WebUI session placed there by RequireSession.
func SessionFromContext(ctx context.Context) (*sessions.WebUISession, bool) {
	session, ok := ctx.Value(contextKey{}).(*sessions.WebUISession)
	return session, ok
}

// RequireSession ensures the request carries a WebUI session and should wrap every
// page route.
//
// Intended behaviour:
//   - read the cookie named SessionCookieName;
//   - if it names a known session, reuse it; otherwise create a new one via
//     sessionsService.Create and send a Set-Cookie back to the browser
//     (HttpOnly, SameSite=Lax, Path=/);
//   - store the session in the request context with withSession and call next.
//
// The session id in the cookie is the only thing the browser stores; all api
// server tokens live in the session on the server.
func RequireSession(sessionsService *sessions.SessionsService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var session *sessions.WebUISession

		if cookie, err := r.Cookie(SessionCookieName); err == nil {
			if existing, ok := sessionsService.Get(sessions.SessionID(cookie.Value)); ok {
				session = existing
			}
		}

		if session == nil {
			created, err := sessionsService.Create(r.Context())
			if err != nil {
				http.Error(w, "could not create session", http.StatusInternalServerError)
				return
			}
			// Secure is intentionally omitted: the WebUI has no TLS by default.
			// Set it to true when serving behind HTTPS.
			http.SetCookie(w, &http.Cookie{
				Name:     SessionCookieName,
				Value:    string(created.ID),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			session = created
		}

		next(w, r.WithContext(withSession(r.Context(), session)))
	}
}
