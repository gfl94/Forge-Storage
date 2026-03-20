package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/repository"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/storage"
)

// BrowseService handles folder browsing and listing operations
type BrowseService struct {
	provider   storage.Provider
	metaRepo   repository.MetadataRepository
	cacheTTL   time.Duration
}

// NewBrowseService creates a new browse service
func NewBrowseService(provider storage.Provider, metaRepo repository.MetadataRepository, cacheTTL time.Duration) *BrowseService {
	return &BrowseService{
		provider: provider,
		metaRepo: metaRepo,
		cacheTTL: cacheTTL,
	}
}

// List lists objects at a given path with caching
func (s *BrowseService) List(ctx context.Context, path string, opts model.ListOptions) (*model.ListResult, error) {
	// Normalize path
	path = normalizePath(path)

	// Check cache first
	if cached, err := s.metaRepo.GetDirectoryCache(ctx, path); err == nil {
		if time.Now().Before(cached.ExpiresAt) {
			// Cache hit and still fresh
			var result model.ListResult
			if err := json.Unmarshal([]byte(cached.PayloadJSON), &result); err == nil {
				result.Source = "cache"
				return &result, nil
			}
		}
	}

	// Cache miss or stale, fetch from provider
	result, err := s.provider.List(ctx, path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list from provider: %w", err)
	}

	// Update cache
	payloadJSON, _ := json.Marshal(result)
	cacheEntry := &model.DirectoryCacheEntry{
		Path:        path,
		PayloadJSON: string(payloadJSON),
		FetchedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(s.cacheTTL),
	}
	s.metaRepo.PutDirectoryCache(ctx, cacheEntry)

	result.Source = "provider"
	return result, nil
}

// FileService handles file metadata operations
type FileService struct {
	provider storage.Provider
	metaRepo repository.MetadataRepository
	cacheTTL time.Duration
}

// NewFileService creates a new file service
func NewFileService(provider storage.Provider, metaRepo repository.MetadataRepository, cacheTTL time.Duration) *FileService {
	return &FileService{
		provider: provider,
		metaRepo: metaRepo,
		cacheTTL: cacheTTL,
	}
}

// GetMetadata retrieves metadata for a file with caching
func (s *FileService) GetMetadata(ctx context.Context, path string) (*model.ObjectInfo, error) {
	// Normalize path
	path = normalizePath(path)

	// Check cache first
	if cached, err := s.metaRepo.GetObjectCache(ctx, path); err == nil {
		if time.Now().Before(cached.ExpiresAt) {
			// Cache hit and still fresh
			return &model.ObjectInfo{
				Name:        strings.TrimPrefix(path, "/"),
				Path:        path,
				Type:        cached.Type,
				Size:        cached.SizeBytes,
				MimeType:    cached.MimeType,
				ModifiedAt:  cached.ModifiedAt,
				ETag:        cached.ETag,
				Previewable: isPreviewable(cached.MimeType),
			}, nil
		}
	}

	// Cache miss or stale, fetch from provider
	info, err := s.provider.Stat(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from provider: %w", err)
	}

	// Update cache
	payloadJSON, _ := json.Marshal(info)
	cacheEntry := &model.ObjectCacheEntry{
		Path:        path,
		Type:        info.Type,
		SizeBytes:   info.Size,
		MimeType:    info.MimeType,
		ModifiedAt:  info.ModifiedAt,
		ETag:        info.ETag,
		PayloadJSON: string(payloadJSON),
		FetchedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(s.cacheTTL),
	}
	s.metaRepo.PutObjectCache(ctx, cacheEntry)

	return info, nil
}

// UploadService handles upload operations
type UploadService struct {
	provider storage.Provider
	metaRepo repository.MetadataRepository
}

// NewUploadService creates a new upload service
func NewUploadService(provider storage.Provider, metaRepo repository.MetadataRepository) *UploadService {
	return &UploadService{
		provider: provider,
		metaRepo: metaRepo,
	}
}

// CreateUploadURL generates a signed upload URL
func (s *UploadService) CreateUploadURL(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error) {
	// Normalize and validate path
	req.Path = normalizePath(req.Path)
	if err := validatePath(req.Path); err != nil {
		return nil, err
	}

	// Generate signed upload target
	target, err := s.provider.CreateUploadTarget(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload target: %w", err)
	}

	// Invalidate parent directory cache
	parentPath := getParentPath(req.Path)
	s.metaRepo.InvalidatePrefix(ctx, parentPath)

	return target, nil
}

// DownloadService handles download operations
type DownloadService struct {
	provider storage.Provider
}

// NewDownloadService creates a new download service
func NewDownloadService(provider storage.Provider) *DownloadService {
	return &DownloadService{
		provider: provider,
	}
}

// CreateDownloadURL generates a signed download URL
func (s *DownloadService) CreateDownloadURL(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error) {
	// Normalize and validate path
	req.Path = normalizePath(req.Path)
	if err := validatePath(req.Path); err != nil {
		return nil, err
	}

	// Generate signed download target
	target, err := s.provider.CreateDownloadTarget(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create download target: %w", err)
	}

	return target, nil
}

// Helper functions

func normalizePath(path string) string {
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove trailing slash unless it's the root
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Clean up multiple slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains '..'")
	}

	return nil
}

func getParentPath(path string) string {
	if path == "/" {
		return "/"
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) <= 1 {
		return "/"
	}

	return "/" + strings.Join(parts[:len(parts)-1], "/")
}

func isPreviewable(mimeType string) bool {
	previewable := []string{
		"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/svg+xml",
		"application/pdf",
		"text/plain", "text/html", "text/css", "text/javascript",
		"application/json", "application/xml",
	}

	for _, t := range previewable {
		if strings.HasPrefix(mimeType, t) {
			return true
		}
	}

	return false
}
