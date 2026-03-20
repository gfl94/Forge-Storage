package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/config"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/pathutil"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/service"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/storage"
)

// Handler wires HTTP routes to services.
type Handler struct {
	cfg       config.Config
	auth      *service.AuthService
	browse    *service.BrowseService
	files     *service.FileService
	rawLogger *slog.Logger
	dbHealth  func(ctx context.Context) error
	provider  storage.Provider
}

// NewHandler constructs the HTTP handler.
func NewHandler(cfg config.Config, logger *slog.Logger, auth *service.AuthService, browse *service.BrowseService, files *service.FileService, dbHealth func(ctx context.Context) error, provider storage.Provider) *Handler {
	return &Handler{
		cfg:       cfg,
		auth:      auth,
		browse:    browse,
		files:     files,
		rawLogger: logger,
		dbHealth:  dbHealth,
		provider:  provider,
	}
}

// Router builds the chi router with middleware and routes.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(requestLogger(h.rawLogger))
	r.Use(h.sessionLoader)

	r.Route("/api", func(api chi.Router) {
		api.Post("/login", h.handleLogin)
		api.Get("/session", h.handleSession)
		api.Group(func(pr chi.Router) {
			pr.Use(h.requireAuth)
			pr.Post("/logout", h.handleLogout)
			pr.Get("/list", h.handleList)
			pr.Get("/meta", h.handleMeta)
			pr.Post("/upload-url", h.handleUploadURL)
			pr.Post("/download-url", h.handleDownloadURL)
			pr.Get("/preview", h.handlePreview)
		})
	})

	r.Get("/health", h.handleHealth)
	r.Get("/ready", h.handleReady)

	return r
}

// sessionLoader attaches user/session to context when present.
func (h *Handler) sessionLoader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.cfg.SessionCookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		user, session, err := h.auth.ValidateSession(r.Context(), cookie.Value)
		if err == service.ErrUnauthorized {
			clearCookie(w, h.cfg.SessionCookieName)
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Unable to validate session")
			return
		}
		if user != nil && session != nil {
			ctx := withUser(r.Context(), user)
			ctx = withSession(ctx, session)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth enforces authentication for protected routes.
func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userFromContext(r.Context()) == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleLogin authenticates and issues session cookie.
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid payload")
		return
	}
	user, session, err := h.auth.Login(r.Context(), body.Username, body.Password)
	if err == service.ErrUnauthorized {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Unable to login")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cfg.SessionCookieName)
	if err == nil {
		_ = h.auth.Logout(r.Context(), cookie.Value)
	}
	clearCookie(w, h.cfg.SessionCookieName)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleSession(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context()).(*model.User)
	if user == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		rawPath = "/"
	}
	norm, err := pathutil.Normalize(rawPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", "Path is invalid")
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")
	listing, err := h.browse.List(r.Context(), norm, limit, cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Unable to list path")
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (h *Handler) handleMeta(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	norm, err := pathutil.Normalize(rawPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", "Path is invalid")
		return
	}
	meta, err := h.files.Metadata(r.Context(), norm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Unable to fetch metadata")
		return
	}
	if meta == nil {
		writeError(w, http.StatusNotFound, "not_found", "Object not found")
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handler) handleUploadURL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path     string `json:"path"`
		Size     int64  `json:"size"`
		MimeType string `json:"mimeType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid payload")
		return
	}
	norm, err := pathutil.Normalize(body.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", "Path is invalid")
		return
	}
	target, err := h.files.UploadTarget(r.Context(), model.UploadRequest{
		Path:     norm,
		Size:     body.Size,
		MimeType: body.MimeType,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Unable to create upload target")
		return
	}
	writeJSON(w, http.StatusOK, target)
}

func (h *Handler) handleDownloadURL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid payload")
		return
	}
	norm, err := pathutil.Normalize(body.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", "Path is invalid")
		return
	}
	target, err := h.files.DownloadTarget(r.Context(), model.DownloadRequest{Path: norm})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Unable to create download target")
		return
	}
	writeJSON(w, http.StatusOK, target)
}

func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	norm, err := pathutil.Normalize(rawPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", "Path is invalid")
		return
	}
	target, previewable, err := h.files.PreviewTarget(r.Context(), norm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Unable to create preview target")
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "not_found", "Object not found")
		return
	}
	mode := "download"
	if previewable {
		mode = "redirect"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":      norm,
		"mode":      mode,
		"url":       target.URL,
		"expiresAt": target.ExpiresAt,
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	sqliteStatus := "ok"
	if h.dbHealth != nil {
		if err := h.dbHealth(ctx); err != nil {
			sqliteStatus = "error"
		}
	}
	checks["sqlite"] = sqliteStatus

	providerStatus := "ok"
	if h.provider != nil {
		if err := h.provider.CheckReady(ctx); err != nil {
			providerStatus = "error"
		}
	}
	checks["storageProvider"] = providerStatus

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     sqliteStatus == "ok" && providerStatus == "ok",
		"checks": checks,
	})
}

func parseLimit(val string) int32 {
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return 0
	}
	return int32(n)
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
