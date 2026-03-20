package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/repository"
)

// CacheService wraps access to the metadata cache with TTL handling.
type CacheService struct {
	repo      repository.MetadataRepository
	dirTTL    time.Duration
	objectTTL time.Duration
}

// NewCacheService creates a CacheService.
func NewCacheService(repo repository.MetadataRepository, dirTTL, objectTTL time.Duration) *CacheService {
	return &CacheService{
		repo:      repo,
		dirTTL:    dirTTL,
		objectTTL: objectTTL,
	}
}

// GetDirectory returns a cached directory if still fresh.
func (c *CacheService) GetDirectory(ctx context.Context, path string) (*model.DirectoryListing, bool, error) {
	entry, err := c.repo.GetDirectoryCache(ctx, path)
	if err != nil || entry == nil {
		return nil, false, err
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false, nil
	}
	var listing model.DirectoryListing
	if err := json.Unmarshal(entry.Payload, &listing); err != nil {
		return nil, false, err
	}
	listing.Source = "cache"
	return &listing, true, nil
}

// PutDirectory caches a directory listing.
func (c *CacheService) PutDirectory(ctx context.Context, path string, listing model.DirectoryListing) error {
	payload, err := json.Marshal(model.DirectoryListing{
		Path:       listing.Path,
		Entries:    listing.Entries,
		NextCursor: listing.NextCursor,
		Source:     "cache",
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	entry := model.DirectoryCacheEntry{
		Path:      path,
		Payload:   payload,
		FetchedAt: now,
		ExpiresAt: now.Add(c.dirTTL),
	}
	return c.repo.PutDirectoryCache(ctx, entry)
}

// GetObject returns cached object metadata if fresh.
func (c *CacheService) GetObject(ctx context.Context, path string) (*model.ObjectMetadata, bool, error) {
	entry, err := c.repo.GetObjectCache(ctx, path)
	if err != nil || entry == nil {
		return nil, false, err
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false, nil
	}
	var meta model.ObjectMetadata
	if err := json.Unmarshal(entry.Payload, &meta); err != nil {
		return nil, false, err
	}
	return &meta, true, nil
}

// PutObject caches object metadata.
func (c *CacheService) PutObject(ctx context.Context, path string, meta model.ObjectMetadata) error {
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	entry := model.ObjectCacheEntry{
		Path:      path,
		Type:      meta.Type,
		SizeBytes: meta.Size,
		MimeType:  meta.MimeType,
		Modified:  meta.ModifiedAt,
		ETag:      meta.ETag,
		Payload:   payload,
		FetchedAt: now,
		ExpiresAt: now.Add(c.objectTTL),
	}
	return c.repo.PutObjectCache(ctx, entry)
}

// InvalidatePrefix removes cached entries under prefix.
func (c *CacheService) InvalidatePrefix(ctx context.Context, path string) error {
	return c.repo.InvalidatePrefix(ctx, path)
}
