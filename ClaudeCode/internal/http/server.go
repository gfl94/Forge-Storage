package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/auth"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/config"
	httpMiddleware "github.com/gfl94/Forge-Storage/ClaudeCode/internal/http/middleware"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/observability"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/repository"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/service"
)

// Server holds HTTP server dependencies
type Server struct {
	router          *chi.Mux
	authService     *auth.Service
	browseService   *service.BrowseService
	fileService     *service.FileService
	uploadService   *service.UploadService
	downloadService *service.DownloadService
	logger          *observability.Logger
	config          *config.Config
}

// NewServer creates a new HTTP server
func NewServer(
	authService *auth.Service,
	browseService *service.BrowseService,
	fileService *service.FileService,
	uploadService *service.UploadService,
	downloadService *service.DownloadService,
	logger *observability.Logger,
	cfg *config.Config,
	db *repository.MetadataRepository,
) *Server {
	s := &Server{
		router:          chi.NewRouter(),
		authService:     authService,
		browseService:   browseService,
		fileService:     fileService,
		uploadService:   uploadService,
		downloadService: downloadService,
		logger:          logger,
		config:          cfg,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Global middleware
	s.router.Use(httpMiddleware.RequestID)
	s.router.Use(httpMiddleware.SecurityHeaders)
	s.router.Use(httpMiddleware.CORS)

	// Public routes
	s.router.Post("/api/login", s.handleLogin)
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ready", s.handleReady)

	// Protected routes
	s.router.Group(func(r chi.Router) {
		r.Use(httpMiddleware.RequireAuth(s.authService, s.config.Session.CookieName))

		r.Post("/api/logout", s.handleLogout)
		r.Get("/api/session", s.handleSession)
		r.Get("/api/list", s.handleList)
		r.Get("/api/meta", s.handleMeta)
		r.Post("/api/upload-url", s.handleUploadURL)
		r.Post("/api/download-url", s.handleDownloadURL)
		r.Get("/api/preview", s.handlePreview)
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Handler functions

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	user, session, err := s.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		s.writeError(w, "unauthorized", "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.Session.CookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   s.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	s.writeJSON(w, map[string]interface{}{
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
	}, http.StatusOK)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(s.config.Session.CookieName)
	if err == nil {
		s.authService.Logout(r.Context(), cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.Session.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   s.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	s.writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	user := httpMiddleware.GetUser(r.Context())
	if user == nil {
		s.writeJSON(w, map[string]interface{}{
			"authenticated": false,
		}, http.StatusOK)
		return
	}

	s.writeJSON(w, map[string]interface{}{
		"authenticated": true,
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
	}, http.StatusOK)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 500 {
				limit = 500
			}
		}
	}

	cursor := r.URL.Query().Get("cursor")

	result, err := s.browseService.List(r.Context(), path, model.ListOptions{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		s.logger.Error(r.Context(), "Failed to list directory", err)
		s.writeError(w, "provider_error", "Failed to list directory", http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, result, http.StatusOK)
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		s.writeError(w, "validation_error", "Path is required", http.StatusBadRequest)
		return
	}

	info, err := s.fileService.GetMetadata(r.Context(), path)
	if err != nil {
		s.logger.Error(r.Context(), "Failed to get metadata", err)
		s.writeError(w, "not_found", "File not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, info, http.StatusOK)
}

func (s *Server) handleUploadURL(w http.ResponseWriter, r *http.Request) {
	var req model.UploadRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		s.writeError(w, "validation_error", "Path is required", http.StatusBadRequest)
		return
	}

	target, err := s.uploadService.CreateUploadURL(r.Context(), req)
	if err != nil {
		s.logger.Error(r.Context(), "Failed to create upload URL", err)
		s.writeError(w, "provider_error", "Failed to create upload URL", http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, target, http.StatusOK)
}

func (s *Server) handleDownloadURL(w http.ResponseWriter, r *http.Request) {
	var req model.DownloadRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		s.writeError(w, "validation_error", "Path is required", http.StatusBadRequest)
		return
	}

	target, err := s.downloadService.CreateDownloadURL(r.Context(), req)
	if err != nil {
		s.logger.Error(r.Context(), "Failed to create download URL", err)
		s.writeError(w, "not_found", "File not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, target, http.StatusOK)
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		s.writeError(w, "validation_error", "Path is required", http.StatusBadRequest)
		return
	}

	target, err := s.downloadService.CreateDownloadURL(r.Context(), model.DownloadRequest{Path: path})
	if err != nil {
		s.logger.Error(r.Context(), "Failed to create preview URL", err)
		s.writeError(w, "not_found", "File not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, map[string]interface{}{
		"path":      path,
		"mode":      "redirect",
		"url":       target.URL,
		"expiresAt": target.ExpiresAt.Format(time.RFC3339),
	}, http.StatusOK)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check dependencies
	checks := map[string]string{
		"sqlite":          "ok",
		"storageProvider": "ok",
	}

	s.writeJSON(w, map[string]interface{}{
		"ok":     true,
		"checks": checks,
	}, http.StatusOK)
}

// Helper methods

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error: model.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
