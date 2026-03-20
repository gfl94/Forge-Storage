package handler

import (
	"errors"
	"net/http"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/config"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/http/middleware"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/repository"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/service"
)

// AuthHandler handles login/logout/session endpoints.
type AuthHandler struct {
	authService *service.AuthService
	sessions    repository.SessionRepository
	users       repository.UserRepository
	cfg         *config.Config
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService, sessions repository.SessionRepository, users repository.UserRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authService: authService, sessions: sessions, users: users, cfg: cfg}
}

// Login handles POST /api/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body.")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Username and password are required.")
		return
	}

	user, session, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid credentials.")
			return
		}
		writeError(w, http.StatusInternalServerError, "database_error", "Login failed.")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.cfg.SessionTTL.Seconds()),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// Logout handles POST /api/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID := middleware.GetSessionID(r.Context())
	if sessionID != "" {
		_ = h.authService.Logout(r.Context(), sessionID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Session handles GET /api/session.
func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	sessionID := middleware.GetSessionID(r.Context())
	userID := middleware.GetUserID(r.Context())

	if sessionID == "" || userID == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"id": userID,
		},
	})
}
