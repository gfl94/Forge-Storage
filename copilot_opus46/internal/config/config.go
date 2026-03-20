// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Env     string // "production" or "development"
	Addr    string // listen address, e.g. "127.0.0.1:8081"
	BaseURL string // external base URL

	SessionCookieName   string
	SessionCookieSecure bool
	SessionTTL          time.Duration

	SQLitePath string

	MetadataCacheDirTTL    time.Duration
	MetadataCacheObjectTTL time.Duration

	StorageProvider       string // "azure"
	AzureStorageAccount   string
	AzureStorageContainer string
	AzureStorageKey       string
	SignedURLTTL          time.Duration

	LogLevel string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Env:                 envOr("APP_ENV", "development"),
		Addr:                envOr("APP_ADDR", "127.0.0.1:8081"),
		BaseURL:             envOr("APP_BASE_URL", "http://localhost:8081"),
		SessionCookieName:   envOr("SESSION_COOKIE_NAME", "storage_session"),
		SessionCookieSecure: envBoolOr("SESSION_COOKIE_SECURE", false),
		SQLitePath:          envOr("SQLITE_PATH", "storage.db"),
		StorageProvider:     envOr("STORAGE_PROVIDER", "azure"),
		AzureStorageAccount: os.Getenv("AZURE_STORAGE_ACCOUNT"),
		AzureStorageContainer: envOr("AZURE_STORAGE_CONTAINER", "files"),
		AzureStorageKey:     os.Getenv("AZURE_STORAGE_KEY"),
		LogLevel:            envOr("LOG_LEVEL", "info"),
	}

	sessionTTLHours, err := envIntOr("SESSION_TTL_HOURS", 168)
	if err != nil {
		return nil, fmt.Errorf("SESSION_TTL_HOURS: %w", err)
	}
	cfg.SessionTTL = time.Duration(sessionTTLHours) * time.Hour

	dirTTL, err := envIntOr("METADATA_CACHE_DIR_TTL_SECONDS", 60)
	if err != nil {
		return nil, fmt.Errorf("METADATA_CACHE_DIR_TTL_SECONDS: %w", err)
	}
	cfg.MetadataCacheDirTTL = time.Duration(dirTTL) * time.Second

	objTTL, err := envIntOr("METADATA_CACHE_OBJECT_TTL_SECONDS", 120)
	if err != nil {
		return nil, fmt.Errorf("METADATA_CACHE_OBJECT_TTL_SECONDS: %w", err)
	}
	cfg.MetadataCacheObjectTTL = time.Duration(objTTL) * time.Second

	signedTTL, err := envIntOr("SIGNED_URL_TTL_SECONDS", 300)
	if err != nil {
		return nil, fmt.Errorf("SIGNED_URL_TTL_SECONDS: %w", err)
	}
	cfg.SignedURLTTL = time.Duration(signedTTL) * time.Second

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBoolOr(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envIntOr(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", v, err)
	}
	return n, nil
}
