// Package storage defines the StorageProvider interface for object storage backends.
package storage

import (
	"context"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
)

// Provider is the abstraction over cloud object storage.
type Provider interface {
	// List returns entries under the given normalized path prefix.
	List(ctx context.Context, path string, opts model.ListOptions) (*model.ListResult, error)

	// Stat returns metadata for a single object.
	Stat(ctx context.Context, path string) (*model.ObjectInfo, error)

	// CreateUploadTarget generates a signed upload URL and instructions.
	CreateUploadTarget(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error)

	// CreateDownloadTarget generates a signed download URL.
	CreateDownloadTarget(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error)

	// Ping checks provider connectivity.
	Ping(ctx context.Context) error
}
