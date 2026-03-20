package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	AppEnv              string
	Addr                string
	BaseURL             string
	SessionCookieName   string
	SessionCookieSecure bool
	SessionTTL          time.Duration

	SQLitePath             string
	MetadataCacheDirTTL    time.Duration
	MetadataCacheObjectTTL time.Duration

	StorageProvider string
	AzureAccount    string
	AzureKey        string
	AzureContainer  string
	SignedURLTTL    time.Duration

	LogLevel string

	DefaultAdminUser string
	DefaultAdminPass string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		Addr:                   getEnv("APP_ADDR", "127.0.0.1:8081"),
		BaseURL:                getEnv("APP_BASE_URL", "http://127.0.0.1:8081"),
		SessionCookieName:      getEnv("SESSION_COOKIE_NAME", "storage_session"),
		SessionCookieSecure:    getBoolEnv("SESSION_COOKIE_SECURE", false),
		SessionTTL:             getDurationHoursEnv("SESSION_TTL_HOURS", 24*time.Hour),
		SQLitePath:             getEnv("SQLITE_PATH", "./storage.db"),
		MetadataCacheDirTTL:    getDurationEnv("METADATA_CACHE_DIR_TTL_SECONDS", 60*time.Second),
		MetadataCacheObjectTTL: getDurationEnv("METADATA_CACHE_OBJECT_TTL_SECONDS", 120*time.Second),
		StorageProvider:        getEnv("STORAGE_PROVIDER", "azure"),
		AzureAccount:           os.Getenv("AZURE_STORAGE_ACCOUNT"),
		AzureKey:               os.Getenv("AZURE_STORAGE_KEY"),
		AzureContainer:         os.Getenv("AZURE_STORAGE_CONTAINER"),
		SignedURLTTL:           getDurationEnv("SIGNED_URL_TTL_SECONDS", 5*time.Minute),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		DefaultAdminUser:       getEnv("DEFAULT_ADMIN_USERNAME", "admin"),
		DefaultAdminPass:       os.Getenv("DEFAULT_ADMIN_PASSWORD"),
	}

	if cfg.StorageProvider == "azure" {
		if cfg.AzureAccount == "" || cfg.AzureKey == "" || cfg.AzureContainer == "" {
			return cfg, fmt.Errorf("azure storage requires AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY, AZURE_STORAGE_CONTAINER")
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		if n, err := strconv.Atoi(val); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}

func getDurationHoursEnv(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		if n, err := strconv.Atoi(val); err == nil {
			return time.Duration(n) * time.Hour
		}
	}
	return fallback
}
