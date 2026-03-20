// Package sqlite implements the repository interfaces using SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps an *sql.DB and implements all repository interfaces.
type DB struct {
	db *sql.DB
}

// New opens a SQLite database and runs migrations.
func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite handles one writer at a time

	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *DB) Close() error {
	return s.db.Close()
}

// Ping checks that the database is reachable.
func (s *DB) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE TABLE IF NOT EXISTS directory_cache (
			path TEXT PRIMARY KEY,
			payload_json TEXT NOT NULL,
			source_etag TEXT,
			fetched_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		)`,
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
		)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			action TEXT NOT NULL,
			path TEXT,
			metadata_json TEXT,
			created_at TEXT NOT NULL
		)`,
	}
	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}

// ── UserRepository ──────────────────────────────────────────────────

// GetByUsername returns the user with the given username, or nil if not found.
func (s *DB) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username)

	var u model.User
	var createdAt string
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &u, nil
}

// CreateUser inserts a new user record.
func (s *DB) CreateUser(ctx context.Context, user *model.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// ── SessionRepository ───────────────────────────────────────────────

// CreateSession inserts a new session.
func (s *DB) CreateSession(ctx context.Context, session *model.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at, created_at, last_seen_at) VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.UserID,
		session.ExpiresAt.Format(time.RFC3339),
		session.CreatedAt.Format(time.RFC3339),
		session.LastSeenAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSession returns a session by ID, or nil if not found or expired.
func (s *DB) GetSession(ctx context.Context, id string) (*model.Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, created_at, last_seen_at FROM sessions WHERE id = ?`, id)

	var sess model.Session
	var expiresAt, createdAt, lastSeenAt string
	err := row.Scan(&sess.ID, &sess.UserID, &expiresAt, &createdAt, &lastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	sess.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sess.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)

	if time.Now().After(sess.ExpiresAt) {
		// expired – clean up and return nil
		_ = s.DeleteSession(ctx, id)
		return nil, nil
	}
	return &sess, nil
}

// DeleteSession removes a session by ID.
func (s *DB) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// TouchSession updates the last_seen_at timestamp for a session.
func (s *DB) Touch(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired sessions.
func (s *DB) DeleteExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ?`, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

// ── MetadataRepository ──────────────────────────────────────────────

// GetDirectoryCache retrieves a cached directory listing, or nil if not found/expired.
func (s *DB) GetDirectoryCache(ctx context.Context, path string) (*model.DirectoryCacheEntry, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT path, payload_json, source_etag, fetched_at, expires_at
		 FROM directory_cache WHERE path = ?`, path)

	var e model.DirectoryCacheEntry
	var fetchedAt, expiresAt string
	err := row.Scan(&e.Path, &e.PayloadJSON, &e.SourceETag, &fetchedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get directory cache: %w", err)
	}
	e.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	e.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	if time.Now().After(e.ExpiresAt) {
		return nil, nil // stale
	}
	return &e, nil
}

// PutDirectoryCache upserts a directory cache entry.
func (s *DB) PutDirectoryCache(ctx context.Context, entry *model.DirectoryCacheEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO directory_cache (path, payload_json, source_etag, fetched_at, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		entry.Path, entry.PayloadJSON, entry.SourceETag,
		entry.FetchedAt.Format(time.RFC3339),
		entry.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("put directory cache: %w", err)
	}
	return nil
}

// GetObjectCache retrieves cached object metadata, or nil if not found/expired.
func (s *DB) GetObjectCache(ctx context.Context, path string) (*model.ObjectCacheEntry, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT path, type, size_bytes, mime_type, modified_at, etag, payload_json, fetched_at, expires_at
		 FROM object_cache WHERE path = ?`, path)

	var e model.ObjectCacheEntry
	var modifiedAt, fetchedAt, expiresAt string
	err := row.Scan(&e.Path, &e.Type, &e.SizeBytes, &e.MimeType, &modifiedAt, &e.ETag,
		&e.PayloadJSON, &fetchedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get object cache: %w", err)
	}
	e.ModifiedAt, _ = time.Parse(time.RFC3339, modifiedAt)
	e.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	e.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	if time.Now().After(e.ExpiresAt) {
		return nil, nil
	}
	return &e, nil
}

// PutObjectCache upserts an object cache entry.
func (s *DB) PutObjectCache(ctx context.Context, entry *model.ObjectCacheEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO object_cache (path, type, size_bytes, mime_type, modified_at, etag, payload_json, fetched_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Path, entry.Type, entry.SizeBytes, entry.MimeType,
		entry.ModifiedAt.Format(time.RFC3339), entry.ETag, entry.PayloadJSON,
		entry.FetchedAt.Format(time.RFC3339),
		entry.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("put object cache: %w", err)
	}
	return nil
}

// InvalidatePrefix removes all cache entries whose path starts with the given prefix.
func (s *DB) InvalidatePrefix(ctx context.Context, path string) error {
	prefix := path + "%"
	if _, err := s.db.ExecContext(ctx, `DELETE FROM directory_cache WHERE path LIKE ?`, prefix); err != nil {
		return fmt.Errorf("invalidate directory cache: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM object_cache WHERE path LIKE ?`, prefix); err != nil {
		return fmt.Errorf("invalidate object cache: %w", err)
	}
	// Also invalidate the exact path (parent directory)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM directory_cache WHERE path = ?`, path); err != nil {
		return fmt.Errorf("invalidate exact directory cache: %w", err)
	}
	return nil
}
