package repository

import (
	"context"
	"time"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
)

// UserRepository handles user persistence.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Create(ctx context.Context, user model.User) error
}

// SessionRepository handles session persistence.
type SessionRepository interface {
	CreateSession(ctx context.Context, session model.Session) error
	GetSession(ctx context.Context, id string) (*model.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteSessionsByUser(ctx context.Context, userID string) error
	UpdateLastSeen(ctx context.Context, id string, at time.Time) error
}

// MetadataRepository handles cached listings and object metadata.
type MetadataRepository interface {
	GetDirectoryCache(ctx context.Context, path string) (*model.DirectoryCacheEntry, error)
	PutDirectoryCache(ctx context.Context, entry model.DirectoryCacheEntry) error
	GetObjectCache(ctx context.Context, path string) (*model.ObjectCacheEntry, error)
	PutObjectCache(ctx context.Context, entry model.ObjectCacheEntry) error
	InvalidatePrefix(ctx context.Context, path string) error
}
