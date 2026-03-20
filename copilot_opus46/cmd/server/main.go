// Package main is the entry point for the storage API server.
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

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/auth"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/config"
	router "github.com/gfl94/Forge-Storage/copilot_opus46/internal/http"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/observability"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/repository/sqlite"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/storage/azure"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Set up logger
	logger := observability.NewLogger(cfg.LogLevel)
	logger.Info("starting storage-api",
		"addr", cfg.Addr,
		"env", cfg.Env,
		"provider", cfg.StorageProvider,
	)

	// Open SQLite database
	db, err := sqlite.New(cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Seed default admin user if none exists
	if err := seedDefaultUser(db); err != nil {
		return fmt.Errorf("seed user: %w", err)
	}

	// Initialize storage provider
	provider, err := azure.New(
		cfg.AzureStorageAccount,
		cfg.AzureStorageContainer,
		cfg.AzureStorageKey,
		cfg.SignedURLTTL,
	)
	if err != nil {
		return fmt.Errorf("init azure provider: %w", err)
	}

	// Build router
	handler := router.New(cfg, logger, db, provider)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

// seedDefaultUser creates the admin user if no users exist.
func seedDefaultUser(db *sqlite.DB) error {
	ctx := context.Background()
	existing, err := db.GetByUsername(ctx, "admin")
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // user already exists
	}

	defaultPassword := os.Getenv("ADMIN_PASSWORD")
	if defaultPassword == "" {
		defaultPassword = "changeme"
	}

	hash, err := auth.HashPassword(defaultPassword)
	if err != nil {
		return err
	}

	id, err := auth.GenerateID("u")
	if err != nil {
		return err
	}

	return db.CreateUser(ctx, &model.User{
		ID:           id,
		Username:     "admin",
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	})
}
