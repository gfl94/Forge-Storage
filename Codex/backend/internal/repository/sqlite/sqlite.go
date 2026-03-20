package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/repository"
)

// Repository bundles the SQLite-backed repositories.
type Repository struct {
	db *sql.DB
}

// New creates the SQLite repository and applies migrations.
func New(dbPath string) (*Repository, error) {
	if dir := filepath.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=1", dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

// Close closes the database.
func (r *Repository) Close() error {
	return r.db.Close()
}

// Ping checks database connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);`,
		`CREATE TABLE IF NOT EXISTS directory_cache (
			path TEXT PRIMARY KEY,
			payload_json TEXT NOT NULL,
			source_etag TEXT,
			fetched_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS object_cache (
			path TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			size_bytes INTEGER,
			mime_type TEXT,
			modified_at TEXT,
			etag TEXT,
			payload_json TEXT NOT NULL,
			fetched_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
	}
	return nil
}

// Users

// GetByUsername returns a user by username.
func (r *Repository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username)
	var u model.User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err == nil {
		u.CreatedAt = parsed
	}
	return &u, nil
}

// GetByID returns a user by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, username, password_hash, created_at FROM users WHERE id = ?`, id)
	var u model.User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		u.CreatedAt = t
	}
	return &u, nil
}

// Create inserts a new user.
func (r *Repository) Create(ctx context.Context, user model.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash, created_at)
		VALUES (?, ?, ?, ?)
	`, user.ID, user.Username, user.PasswordHash, user.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// Sessions

// Create inserts a new session.
func (r *Repository) CreateSession(ctx context.Context, session model.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.ExpiresAt.UTC().Format(time.RFC3339Nano),
		session.CreatedAt.UTC().Format(time.RFC3339Nano), session.LastSeenAt.UTC().Format(time.RFC3339Nano))
	return err
}

// Get returns a session by id.
func (r *Repository) GetSession(ctx context.Context, id string) (*model.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, created_at, last_seen_at FROM sessions WHERE id = ?
	`, id)
	var s model.Session
	var expires, created, lastSeen string
	if err := row.Scan(&s.ID, &s.UserID, &expires, &created, &lastSeen); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339Nano, expires); err == nil {
		s.ExpiresAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
		s.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, lastSeen); err == nil {
		s.LastSeenAt = t
	}
	return &s, nil
}

// Delete removes a session.
func (r *Repository) DeleteSession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteByUser removes all sessions for a user.
func (r *Repository) DeleteSessionsByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// UpdateLastSeen updates last_seen_at and optionally expiry.
func (r *Repository) UpdateLastSeen(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id = ?`, at.UTC().Format(time.RFC3339Nano), id)
	return err
}

// Metadata cache

// GetDirectoryCache returns a cached directory listing.
func (r *Repository) GetDirectoryCache(ctx context.Context, path string) (*model.DirectoryCacheEntry, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT path, payload_json, source_etag, fetched_at, expires_at FROM directory_cache WHERE path = ?
	`, path)
	var entry model.DirectoryCacheEntry
	var fetched, expires string
	if err := row.Scan(&entry.Path, &entry.Payload, &entry.SourceETag, &fetched, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339Nano, fetched); err == nil {
		entry.FetchedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, expires); err == nil {
		entry.ExpiresAt = t
	}
	return &entry, nil
}

// PutDirectoryCache upserts a directory cache entry.
func (r *Repository) PutDirectoryCache(ctx context.Context, entry model.DirectoryCacheEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO directory_cache (path, payload_json, source_etag, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			payload_json=excluded.payload_json,
			source_etag=excluded.source_etag,
			fetched_at=excluded.fetched_at,
			expires_at=excluded.expires_at
	`, entry.Path, entry.Payload, entry.SourceETag, entry.FetchedAt.UTC().Format(time.RFC3339Nano), entry.ExpiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

// GetObjectCache returns a cached object entry.
func (r *Repository) GetObjectCache(ctx context.Context, path string) (*model.ObjectCacheEntry, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT path, type, size_bytes, mime_type, modified_at, etag, payload_json, fetched_at, expires_at
		FROM object_cache WHERE path = ?
	`, path)
	var entry model.ObjectCacheEntry
	var modified, fetched, expires string
	if err := row.Scan(&entry.Path, &entry.Type, &entry.SizeBytes, &entry.MimeType, &modified, &entry.ETag, &entry.Payload, &fetched, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if modified != "" {
		if t, err := time.Parse(time.RFC3339Nano, modified); err == nil {
			entry.Modified = &t
		}
	}
	if t, err := time.Parse(time.RFC3339Nano, fetched); err == nil {
		entry.FetchedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, expires); err == nil {
		entry.ExpiresAt = t
	}
	return &entry, nil
}

// PutObjectCache upserts an object cache entry.
func (r *Repository) PutObjectCache(ctx context.Context, entry model.ObjectCacheEntry) error {
	modified := ""
	if entry.Modified != nil {
		modified = entry.Modified.UTC().Format(time.RFC3339Nano)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO object_cache (path, type, size_bytes, mime_type, modified_at, etag, payload_json, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			type=excluded.type,
			size_bytes=excluded.size_bytes,
			mime_type=excluded.mime_type,
			modified_at=excluded.modified_at,
			etag=excluded.etag,
			payload_json=excluded.payload_json,
			fetched_at=excluded.fetched_at,
			expires_at=excluded.expires_at
	`, entry.Path, entry.Type, entry.SizeBytes, entry.MimeType, modified, entry.ETag, entry.Payload,
		entry.FetchedAt.UTC().Format(time.RFC3339Nano), entry.ExpiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

// InvalidatePrefix deletes cache entries under the given path prefix.
func (r *Repository) InvalidatePrefix(ctx context.Context, path string) error {
	likePattern := path
	if likePattern == "/" {
		likePattern = "/%"
	} else if likePattern != "" {
		likePattern = path + "%"
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM directory_cache WHERE path LIKE ?`, likePattern); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM object_cache WHERE path LIKE ?`, likePattern); err != nil {
		return err
	}
	return nil
}

// Expose interface compliance
var (
	_ repository.UserRepository     = (*Repository)(nil)
	_ repository.SessionRepository  = (*Repository)(nil)
	_ repository.MetadataRepository = (*Repository)(nil)
)
