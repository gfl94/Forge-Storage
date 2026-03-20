package repository

import (
	"context"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
)

// UserRepository defines operations for user persistence
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
}

// SessionRepository defines operations for session persistence
type SessionRepository interface {
	CreateSession(ctx context.Context, session *model.Session) error
	Get(ctx context.Context, id string) (*model.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}

// MetadataRepository defines operations for metadata caching
type MetadataRepository interface {
	GetDirectoryCache(ctx context.Context, path string) (*model.DirectoryCacheEntry, error)
	PutDirectoryCache(ctx context.Context, entry *model.DirectoryCacheEntry) error
	GetObjectCache(ctx context.Context, path string) (*model.ObjectCacheEntry, error)
	PutObjectCache(ctx context.Context, entry *model.ObjectCacheEntry) error
	InvalidatePrefix(ctx context.Context, path string) error
}
