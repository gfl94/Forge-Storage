package service

import (
	"context"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/storage"
)

// FileService handles object metadata, upload target, and download target behavior.
type FileService struct {
	provider  storage.Provider
	cache     *CacheService
	signedTTL time.Duration
}

// NewFileService constructs a FileService.
func NewFileService(provider storage.Provider, cache *CacheService, signedTTL time.Duration) *FileService {
	return &FileService{
		provider:  provider,
		cache:     cache,
		signedTTL: signedTTL,
	}
}

// Metadata returns metadata for a path with caching.
func (f *FileService) Metadata(ctx context.Context, path string) (*model.ObjectMetadata, error) {
	if cached, ok, err := f.cache.GetObject(ctx, path); err != nil {
		return nil, err
	} else if ok {
		return cached, nil
	}

	info, err := f.provider.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	meta := &model.ObjectMetadata{
		Path:        path,
		Type:        "file",
		Size:        info.Size,
		MimeType:    info.MimeType,
		ModifiedAt:  info.ModifiedAt,
		ETag:        info.ETag,
		Previewable: isPreviewable(path, info.MimeType),
	}

	_ = f.cache.PutObject(ctx, path, *meta)

	return meta, nil
}

// UploadTarget produces signed upload instructions and invalidates cache.
func (f *FileService) UploadTarget(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error) {
	target, err := f.provider.CreateUploadTarget(ctx, req, time.Now().Add(f.signedTTL))
	if err != nil {
		return nil, err
	}
	_ = f.cache.InvalidatePrefix(ctx, parentPath(req.Path))
	return target, nil
}

// DownloadTarget produces a signed download URL.
func (f *FileService) DownloadTarget(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error) {
	return f.provider.CreateDownloadTarget(ctx, req, time.Now().Add(f.signedTTL))
}

// PreviewTarget returns download target plus preview mode.
func (f *FileService) PreviewTarget(ctx context.Context, path string) (*model.DownloadTarget, bool, error) {
	meta, err := f.Metadata(ctx, path)
	if err != nil {
		return nil, false, err
	}
	if meta == nil {
		return nil, false, nil
	}
	target, err := f.DownloadTarget(ctx, model.DownloadRequest{Path: path})
	if err != nil {
		return nil, false, err
	}
	return target, meta.Previewable, nil
}

func isPreviewable(path, mimeType string) bool {
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	if mimeType == "application/pdf" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	if mimeType == "" && (ext == ".txt" || ext == ".md") {
		return true
	}
	if mimeType == "" {
		if detected := mime.TypeByExtension(ext); detected != "" {
			return strings.HasPrefix(detected, "image/") || strings.HasPrefix(detected, "text/") || detected == "application/pdf"
		}
	}
	return false
}

func parentPath(p string) string {
	if p == "/" {
		return "/"
	}
	dir := filepath.Dir(p)
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	return dir
}
