package storage

import (
	"context"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
)

// Provider defines the interface for storage operations
type Provider interface {
	// List lists objects at a given path with optional pagination
	List(ctx context.Context, path string, opts model.ListOptions) (*model.ListResult, error)

	// Stat returns metadata for a single object
	Stat(ctx context.Context, path string) (*model.ObjectInfo, error)

	// CreateUploadTarget generates signed upload instructions
	CreateUploadTarget(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error)

	// CreateDownloadTarget generates signed download URL
	CreateDownloadTarget(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error)

	// Delete removes an object (for future use)
	Delete(ctx context.Context, path string) error
}
