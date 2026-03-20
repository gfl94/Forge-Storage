// Package router sets up the chi router with all routes and middleware.
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/config"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/http/handler"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/http/middleware"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/repository/sqlite"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/service"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/storage"
)

// New creates the HTTP router with all routes and middleware wired up.
func New(
	cfg *config.Config,
	logger *slog.Logger,
	db *sqlite.DB,
	provider storage.Provider,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.SecurityHeaders)
	if cfg.Env == "development" {
		r.Use(middleware.CORS)
	}

	// Auth middleware
	authMW := middleware.NewAuth(db, db, cfg.SessionCookieName)

	// Services
	authService := service.NewAuthService(db, db, cfg, logger)
	browseService := service.NewBrowseService(provider, db, cfg, logger)
	fileService := service.NewFileService(provider, db, cfg, logger)
	uploadService := service.NewUploadService(provider, db, logger)
	downloadService := service.NewDownloadService(provider, logger)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, db, db, cfg)
	browseHandler := handler.NewBrowseHandler(browseService, fileService)
	transferHandler := handler.NewTransferHandler(uploadService, downloadService)
	healthHandler := handler.NewHealthHandler(db, provider)

	// Public routes
	r.Post("/api/login", authHandler.Login)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authMW.Required)

		r.Post("/api/logout", authHandler.Logout)
		r.Get("/api/session", authHandler.Session)
		r.Get("/api/list", browseHandler.List)
		r.Get("/api/meta", browseHandler.Meta)
		r.Post("/api/upload-url", transferHandler.UploadURL)
		r.Post("/api/download-url", transferHandler.DownloadURL)
		r.Get("/api/preview", transferHandler.Preview)
	})

	return r
}
