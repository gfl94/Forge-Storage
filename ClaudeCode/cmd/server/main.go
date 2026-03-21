package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/auth"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/config"
	httpServer "github.com/gfl94/Forge-Storage/ClaudeCode/internal/http"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/observability"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/repository"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/repository/sqlite"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/service"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/storage/azure"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger := observability.NewLogger(cfg.Log.Level)
	logger.Info(context.Background(), "Starting Storage API service", "env", cfg.App.Env)

	// Initialize SQLite database
	db, err := sqlite.New(cfg.SQLite.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	logger.Info(context.Background(), "Database initialized", "path", cfg.SQLite.Path)

	// Initialize default admin user if not exists
	if err := initializeDefaultUser(context.Background(), db); err != nil {
		logger.Warn(context.Background(), "Failed to initialize default user", "error", err)
	}

	// Initialize storage provider
	var provider *azure.Provider
	if cfg.Storage.Provider == "azure" {
		provider, err = azure.NewProvider(
			cfg.Azure.StorageAccount,
			cfg.Azure.StorageKey,
			cfg.Azure.Container,
			cfg.Storage.SignedURLTTL,
		)
		if err != nil {
			return fmt.Errorf("failed to initialize Azure provider: %w", err)
		}
		logger.Info(context.Background(), "Azure provider initialized", "account", cfg.Azure.StorageAccount, "container", cfg.Azure.Container)
	} else {
		return fmt.Errorf("unsupported storage provider: %s", cfg.Storage.Provider)
	}

	// Initialize services
	authService := auth.NewService(db, db, cfg.Session.TTL)
	browseService := service.NewBrowseService(provider, db, cfg.Cache.DirectoryTTL)
	fileService := service.NewFileService(provider, db, cfg.Cache.ObjectTTL)
	uploadService := service.NewUploadService(provider, db)
	downloadService := service.NewDownloadService(provider)

	// Cast db to repository.MetadataRepository for the server
	var metaRepo repository.MetadataRepository = db

	// Initialize HTTP server
	server := httpServer.NewServer(
		authService,
		browseService,
		fileService,
		uploadService,
		downloadService,
		logger,
		cfg,
		&metaRepo,
	)

	// Start HTTP server
	httpSrv := &http.Server{
		Addr:         cfg.App.Addr,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(context.Background(), "HTTP server starting", "addr", cfg.App.Addr)
		serverErrors <- httpSrv.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Info(context.Background(), "Shutdown signal received", "signal", sig)

		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpSrv.Shutdown(ctx); err != nil {
			logger.Error(context.Background(), "Failed to shutdown gracefully", err)
			httpSrv.Close()
			return fmt.Errorf("shutdown error: %w", err)
		}

		logger.Info(context.Background(), "Server stopped gracefully")
	}

	return nil
}

func initializeDefaultUser(ctx context.Context, db *sqlite.DB) error {
	// Check if admin user already exists
	_, err := db.GetByUsername(ctx, "admin")
	if err == nil {
		// User already exists
		return nil
	}

	// Create default admin user
	// IMPORTANT: Change this password in production!
	passwordHash, err := auth.HashPassword("admin123")
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     "admin",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	if err := db.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create default user: %w", err)
	}

	log.Printf("Default admin user created (username: admin, password: admin123)")
	return nil
}
