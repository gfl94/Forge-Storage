package service

import (
	"context"
	"path"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/storage"
)

// BrowseService handles directory listing and metadata lookup.
type BrowseService struct {
	provider     storage.Provider
	cache        *CacheService
	defaultLimit int32
	maxLimit     int32
}

// NewBrowseService constructs a BrowseService.
func NewBrowseService(provider storage.Provider, cache *CacheService, defaultLimit, maxLimit int32) *BrowseService {
	return &BrowseService{
		provider:     provider,
		cache:        cache,
		defaultLimit: defaultLimit,
		maxLimit:     maxLimit,
	}
}

// List returns a directory listing with cache support.
func (b *BrowseService) List(ctx context.Context, pathStr string, limit int32, cursor string) (*model.DirectoryListing, error) {
	if limit <= 0 {
		limit = b.defaultLimit
	}
	if b.maxLimit > 0 && limit > b.maxLimit {
		limit = b.maxLimit
	}

	if cached, ok, err := b.cache.GetDirectory(ctx, pathStr); err != nil {
		return nil, err
	} else if ok && cursor == "" {
		return cached, nil
	}

	res, err := b.provider.List(ctx, storage.ListOptions{
		Path:   pathStr,
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	listing := &model.DirectoryListing{
		Path:       pathStr,
		Entries:    make([]model.DirectoryEntry, 0, len(res.Items)),
		NextCursor: res.NextCursor,
		Source:     "provider",
	}

	for _, item := range res.Items {
		entry := model.DirectoryEntry{
			Name: path.Base(item.Path),
			Path: item.Path,
			Type: chooseType(item.IsDir),
		}
		if !item.IsDir {
			entry.Size = item.Size
			entry.MimeType = item.MimeType
			entry.ModifiedAt = item.ModifiedAt
		}
		listing.Entries = append(listing.Entries, entry)
	}

	if cursor == "" {
		_ = b.cache.PutDirectory(ctx, pathStr, *listing)
	}

	return listing, nil
}

func chooseType(isDir bool) string {
	if isDir {
		return "directory"
	}
	return "file"
}
