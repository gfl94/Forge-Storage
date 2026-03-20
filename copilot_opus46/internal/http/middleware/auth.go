package middleware

import (
	"context"
	"net/http"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/repository"
)

// Auth is middleware that checks for a valid session cookie and populates the context.
type Auth struct {
	sessions   repository.SessionRepository
	users      repository.UserRepository
	cookieName string
}

// NewAuth creates a new Auth middleware.
func NewAuth(sessions repository.SessionRepository, users repository.UserRepository, cookieName string) *Auth {
	return &Auth{sessions: sessions, users: users, cookieName: cookieName}
}

// Required rejects requests without a valid session.
func (a *Auth) Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(a.cookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, `{"error":{"code":"unauthorized","message":"Authentication required."}}`, http.StatusUnauthorized)
			return
		}

		session, err := a.sessions.GetSession(r.Context(), cookie.Value)
		if err != nil || session == nil {
			http.Error(w, `{"error":{"code":"unauthorized","message":"Invalid or expired session."}}`, http.StatusUnauthorized)
			return
		}

		// Touch session for activity tracking
		_ = a.sessions.Touch(r.Context(), session.ID)

		ctx := context.WithValue(r.Context(), SessionIDKey, session.ID)
		ctx = context.WithValue(ctx, UserIDKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID returns the authenticated user ID from the context.
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetSessionID returns the session ID from the context.
func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}
