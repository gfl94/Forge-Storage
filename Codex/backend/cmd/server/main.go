package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/config"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/httpapi"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/observability"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/repository/sqlite"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/service"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/storage/azure"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger := observability.NewLogger(cfg.LogLevel)

	repo, err := sqlite.New(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("sqlite: %v", err)
	}
	defer repo.Close()

	authSvc := service.NewAuthService(repo, repo, cfg.SessionTTL)
	if err := authSvc.EnsureDefaultAdmin(context.Background(), cfg.DefaultAdminUser, cfg.DefaultAdminPass); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	cacheSvc := service.NewCacheService(repo, cfg.MetadataCacheDirTTL, cfg.MetadataCacheObjectTTL)

	provider, err := azure.New(cfg.AzureAccount, cfg.AzureKey, cfg.AzureContainer)
	if err != nil {
		log.Fatalf("azure provider: %v", err)
	}

	browseSvc := service.NewBrowseService(provider, cacheSvc, 100, 500)
	fileSvc := service.NewFileService(provider, cacheSvc, cfg.SignedURLTTL)

	handler := httpapi.NewHandler(cfg, logger, authSvc, browseSvc, fileSvc, repo.Ping, provider)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting server", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
