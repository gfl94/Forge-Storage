package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env to test defaults
	for _, k := range []string{"APP_ENV", "APP_ADDR", "SESSION_TTL_HOURS", "SQLITE_PATH"} {
		os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.Addr != "127.0.0.1:8081" {
		t.Errorf("Addr = %q, want 127.0.0.1:8081", cfg.Addr)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Errorf("SessionTTL = %v, want %v", cfg.SessionTTL, 168*time.Hour)
	}
	if cfg.SQLitePath != "storage.db" {
		t.Errorf("SQLitePath = %q, want storage.db", cfg.SQLitePath)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	os.Setenv("APP_ADDR", "0.0.0.0:9090")
	os.Setenv("SESSION_TTL_HOURS", "24")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("APP_ADDR")
		os.Unsetenv("SESSION_TTL_HOURS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "production" {
		t.Errorf("Env = %q, want production", cfg.Env)
	}
	if cfg.Addr != "0.0.0.0:9090" {
		t.Errorf("Addr = %q, want 0.0.0.0:9090", cfg.Addr)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, want %v", cfg.SessionTTL, 24*time.Hour)
	}
}
