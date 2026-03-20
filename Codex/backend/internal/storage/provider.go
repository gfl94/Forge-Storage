package storage

import (
	"context"
	"time"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
)

// ListOptions controls provider listing behavior.
type ListOptions struct {
	Path   string
	Limit  int32
	Cursor string
}

// ListItem represents a normalized item returned by the provider.
type ListItem struct {
	Path       string
	IsDir      bool
	Size       int64
	MimeType   string
	ModifiedAt *time.Time
	ETag       string
}

// ListResult is the provider list response.
type ListResult struct {
	Items      []ListItem
	NextCursor *string
}

// ObjectInfo contains metadata for a single object.
type ObjectInfo struct {
	Path       string
	Size       int64
	MimeType   string
	ModifiedAt *time.Time
	ETag       string
}

// Provider is the abstraction for storage providers.
type Provider interface {
	List(ctx context.Context, opts ListOptions) (ListResult, error)
	Stat(ctx context.Context, path string) (*ObjectInfo, error)
	CreateUploadTarget(ctx context.Context, req model.UploadRequest, expiresAt time.Time) (*model.UploadTarget, error)
	CreateDownloadTarget(ctx context.Context, req model.DownloadRequest, expiresAt time.Time) (*model.DownloadTarget, error)
	CheckReady(ctx context.Context) error
}
