package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Session  SessionConfig
	SQLite   SQLiteConfig
	Cache    CacheConfig
	Storage  StorageConfig
	Azure    AzureConfig
	Log      LogConfig
}

// AppConfig holds general application settings
type AppConfig struct {
	Env     string
	Addr    string
	BaseURL string
}

// SessionConfig holds session-related settings
type SessionConfig struct {
	CookieName   string
	CookieSecure bool
	TTL          time.Duration
}

// SQLiteConfig holds SQLite database settings
type SQLiteConfig struct {
	Path string
}

// CacheConfig holds cache settings
type CacheConfig struct {
	DirectoryTTL time.Duration
	ObjectTTL    time.Duration
}

// StorageConfig holds storage provider settings
type StorageConfig struct {
	Provider       string
	SignedURLTTL   time.Duration
}

// AzureConfig holds Azure Blob Storage settings
type AzureConfig struct {
	StorageAccount string
	Container      string
	StorageKey     string
}

// LogConfig holds logging settings
type LogConfig struct {
	Level string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Env:     getEnv("APP_ENV", "production"),
			Addr:    getEnv("APP_ADDR", "127.0.0.1:8081"),
			BaseURL: getEnv("APP_BASE_URL", "http://localhost"),
		},
		Session: SessionConfig{
			CookieName:   getEnv("SESSION_COOKIE_NAME", "storage_session"),
			CookieSecure: getEnvBool("SESSION_COOKIE_SECURE", false),
			TTL:          getEnvDuration("SESSION_TTL_HOURS", 168) * time.Hour,
		},
		SQLite: SQLiteConfig{
			Path: getEnv("SQLITE_PATH", "./storage.db"),
		},
		Cache: CacheConfig{
			DirectoryTTL: getEnvDuration("METADATA_CACHE_DIR_TTL_SECONDS", 60) * time.Second,
			ObjectTTL:    getEnvDuration("METADATA_CACHE_OBJECT_TTL_SECONDS", 120) * time.Second,
		},
		Storage: StorageConfig{
			Provider:     getEnv("STORAGE_PROVIDER", "azure"),
			SignedURLTTL: getEnvDuration("SIGNED_URL_TTL_SECONDS", 300) * time.Second,
		},
		Azure: AzureConfig{
			StorageAccount: os.Getenv("AZURE_STORAGE_ACCOUNT"),
			Container:      os.Getenv("AZURE_STORAGE_CONTAINER"),
			StorageKey:     os.Getenv("AZURE_STORAGE_KEY"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	// Validate required fields
	if cfg.Storage.Provider == "azure" {
		if cfg.Azure.StorageAccount == "" {
			return nil, fmt.Errorf("AZURE_STORAGE_ACCOUNT is required when using Azure provider")
		}
		if cfg.Azure.Container == "" {
			return nil, fmt.Errorf("AZURE_STORAGE_CONTAINER is required when using Azure provider")
		}
		if cfg.Azure.StorageKey == "" {
			return nil, fmt.Errorf("AZURE_STORAGE_KEY is required when using Azure provider")
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue int64) time.Duration {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return time.Duration(parsed)
		}
	}
	return time.Duration(defaultValue)
}
