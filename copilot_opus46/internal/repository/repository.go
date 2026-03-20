// Package repository defines interfaces for data persistence.
package repository

import (
	"context"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
)

// UserRepository manages user records.
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
}

// SessionRepository manages sessions.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, id string) (*model.Session, error)
	DeleteSession(ctx context.Context, id string) error
	Touch(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}

// MetadataRepository manages cached directory and object metadata.
type MetadataRepository interface {
	GetDirectoryCache(ctx context.Context, path string) (*model.DirectoryCacheEntry, error)
	PutDirectoryCache(ctx context.Context, entry *model.DirectoryCacheEntry) error
	GetObjectCache(ctx context.Context, path string) (*model.ObjectCacheEntry, error)
	PutObjectCache(ctx context.Context, entry *model.ObjectCacheEntry) error
	InvalidatePrefix(ctx context.Context, path string) error
}
