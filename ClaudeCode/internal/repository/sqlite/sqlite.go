package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New creates a new SQLite database connection and initializes schema
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(1) // SQLite works best with single writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Ping checks if the database is accessible
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

	CREATE TABLE IF NOT EXISTS directory_cache (
		path TEXT PRIMARY KEY,
		payload_json TEXT NOT NULL,
		source_etag TEXT,
		fetched_at TEXT NOT NULL,
		expires_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_directory_cache_expires_at ON directory_cache(expires_at);

	CREATE TABLE IF NOT EXISTS object_cache (
		path TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		size_bytes INTEGER,
		mime_type TEXT,
		modified_at TEXT,
		etag TEXT,
		payload_json TEXT NOT NULL,
		fetched_at TEXT NOT NULL,
		expires_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_object_cache_expires_at ON object_cache(expires_at);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// UserRepository implementation

// GetByUsername retrieves a user by username
func (db *DB) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = ?`

	var user model.User
	var createdAt string

	err := db.conn.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &user, nil
}

// Create creates a new user (implements UserRepository)
func (db *DB) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`

	_, err := db.conn.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.CreatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// SessionRepository implementation

// CreateSession creates a new session
func (db *DB) CreateSession(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO sessions (id, user_id, expires_at, created_at, last_seen_at) VALUES (?, ?, ?, ?, ?)`

	_, err := db.conn.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.ExpiresAt.Format(time.RFC3339),
		session.CreatedAt.Format(time.RFC3339),
		session.LastSeenAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (db *DB) Get(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT id, user_id, expires_at, created_at, last_seen_at FROM sessions WHERE id = ?`

	var session model.Session
	var expiresAt, createdAt, lastSeenAt string

	err := db.conn.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&expiresAt,
		&createdAt,
		&lastSeenAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	session.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)

	return &session, nil
}

// Delete deletes a session by ID
func (db *DB) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = ?`

	_, err := db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteExpired deletes all expired sessions
func (db *DB) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at < ?`

	_, err := db.conn.ExecContext(ctx, query, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// MetadataRepository implementation

// GetDirectoryCache retrieves a cached directory listing
func (db *DB) GetDirectoryCache(ctx context.Context, path string) (*model.DirectoryCacheEntry, error) {
	query := `SELECT path, payload_json, source_etag, fetched_at, expires_at FROM directory_cache WHERE path = ?`

	var entry model.DirectoryCacheEntry
	var fetchedAt, expiresAt string
	var sourceETag sql.NullString

	err := db.conn.QueryRowContext(ctx, query, path).Scan(
		&entry.Path,
		&entry.PayloadJSON,
		&sourceETag,
		&fetchedAt,
		&expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cache entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query directory cache: %w", err)
	}

	entry.SourceETag = sourceETag.String
	entry.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	entry.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	return &entry, nil
}

// PutDirectoryCache stores a directory listing in cache
func (db *DB) PutDirectoryCache(ctx context.Context, entry *model.DirectoryCacheEntry) error {
	query := `INSERT OR REPLACE INTO directory_cache (path, payload_json, source_etag, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?)`

	var sourceETag interface{}
	if entry.SourceETag != "" {
		sourceETag = entry.SourceETag
	}

	_, err := db.conn.ExecContext(ctx, query,
		entry.Path,
		entry.PayloadJSON,
		sourceETag,
		entry.FetchedAt.Format(time.RFC3339),
		entry.ExpiresAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to put directory cache: %w", err)
	}

	return nil
}

// GetObjectCache retrieves cached object metadata
func (db *DB) GetObjectCache(ctx context.Context, path string) (*model.ObjectCacheEntry, error) {
	query := `SELECT path, type, size_bytes, mime_type, modified_at, etag, payload_json, fetched_at, expires_at FROM object_cache WHERE path = ?`

	var entry model.ObjectCacheEntry
	var sizeBytes sql.NullInt64
	var mimeType, modifiedAt, etag sql.NullString
	var fetchedAt, expiresAt string

	err := db.conn.QueryRowContext(ctx, query, path).Scan(
		&entry.Path,
		&entry.Type,
		&sizeBytes,
		&mimeType,
		&modifiedAt,
		&etag,
		&entry.PayloadJSON,
		&fetchedAt,
		&expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cache entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query object cache: %w", err)
	}

	entry.SizeBytes = sizeBytes.Int64
	entry.MimeType = mimeType.String
	entry.ETag = etag.String
	if modifiedAt.Valid {
		entry.ModifiedAt, _ = time.Parse(time.RFC3339, modifiedAt.String)
	}
	entry.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	entry.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	return &entry, nil
}

// PutObjectCache stores object metadata in cache
func (db *DB) PutObjectCache(ctx context.Context, entry *model.ObjectCacheEntry) error {
	query := `INSERT OR REPLACE INTO object_cache (path, type, size_bytes, mime_type, modified_at, etag, payload_json, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var modifiedAt interface{}
	if !entry.ModifiedAt.IsZero() {
		modifiedAt = entry.ModifiedAt.Format(time.RFC3339)
	}

	var mimeType interface{}
	if entry.MimeType != "" {
		mimeType = entry.MimeType
	}

	var etag interface{}
	if entry.ETag != "" {
		etag = entry.ETag
	}

	_, err := db.conn.ExecContext(ctx, query,
		entry.Path,
		entry.Type,
		entry.SizeBytes,
		mimeType,
		modifiedAt,
		etag,
		entry.PayloadJSON,
		entry.FetchedAt.Format(time.RFC3339),
		entry.ExpiresAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to put object cache: %w", err)
	}

	return nil
}

// InvalidatePrefix removes all cache entries matching a path prefix
func (db *DB) InvalidatePrefix(ctx context.Context, path string) error {
	// Delete from directory cache
	query1 := `DELETE FROM directory_cache WHERE path LIKE ? || '%'`
	if _, err := db.conn.ExecContext(ctx, query1, path); err != nil {
		return fmt.Errorf("failed to invalidate directory cache: %w", err)
	}

	// Delete from object cache
	query2 := `DELETE FROM object_cache WHERE path LIKE ? || '%'`
	if _, err := db.conn.ExecContext(ctx, query2, path); err != nil {
		return fmt.Errorf("failed to invalidate object cache: %w", err)
	}

	return nil
}
