package auth

import (
	userservice "chat/src/api_server/services/users"
	"chat/src/shared_api"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func withSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, contextKey{}, session)
}

// SessionFromContext returns the session stored by RequireUser. ok is false when
// the route was not wrapped with RequireUser.
func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(contextKey{}).(Session)
	return session, ok
}

// RequireUser rejects the request with 401 unless it carries a valid
// "Authorization: Bearer <token>" header, and hands the resolved session to next.
func RequireUser(users *userservice.UsersService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := tokenFromRequest(r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		uid, err := users.GetUserUIDByToken(token)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		next(w, r.WithContext(withSession(r.Context(), Session{Token: token, UID: uid})))
	}
}

// tokenFromRequest extracts the bearer token of the request.
func tokenFromRequest(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", userservice.TokenNotFoundError
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	if token == "" {
		return "", userservice.TokenNotFoundError
	}
	return token, nil
}

// writeAuthError answers an authentication failure. Every auth failure is a
// missing or unknown token, so it always maps to 401.
func writeAuthError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(struct {
		Error *shared_api.Error `json:"error,omitempty"`
	}{Error: &shared_api.Error{Field: "token", Message: err.Error()}})
}
