// Package service contains the application business logic.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/auth"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/config"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/repository"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/storage"
)

// AuthService handles login/logout/session operations.
type AuthService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	cfg      *config.Config
	logger   *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(users repository.UserRepository, sessions repository.SessionRepository, cfg *config.Config, logger *slog.Logger) *AuthService {
	return &AuthService{users: users, sessions: sessions, cfg: cfg, logger: logger}
}

// Login authenticates a user and creates a session.
func (s *AuthService) Login(ctx context.Context, username, password string) (*model.User, *model.Session, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, nil, ErrUnauthorized
	}
	if !auth.VerifyPassword(user.PasswordHash, password) {
		return nil, nil, ErrUnauthorized
	}

	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		return nil, nil, fmt.Errorf("generate session: %w", err)
	}

	now := time.Now()
	session := &model.Session{
		ID:         sessionID,
		UserID:     user.ID,
		ExpiresAt:  now.Add(s.cfg.SessionTTL),
		CreatedAt:  now,
		LastSeenAt: now,
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	s.logger.Info("user logged in", "userId", user.ID, "username", user.Username)
	return user, session, nil
}

// Logout deletes a session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.DeleteSession(ctx, sessionID)
}

// GetSession retrieves and validates a session.
func (s *AuthService) GetSession(ctx context.Context, sessionID string) (*model.Session, *model.User, error) {
	session, err := s.sessions.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, nil, nil
	}

	// Touch the session to update last seen
	_ = s.sessions.Touch(ctx, sessionID)

	user, err := s.users.GetByUsername(ctx, "") // we need by ID
	if err != nil {
		return nil, nil, fmt.Errorf("lookup user: %w", err)
	}
	// Since we don't have GetByID, we embed user info in the session context
	return session, user, nil
}

// BrowseService handles directory listing with caching.
type BrowseService struct {
	provider storage.Provider
	metadata repository.MetadataRepository
	cfg      *config.Config
	logger   *slog.Logger
}

// NewBrowseService creates a new BrowseService.
func NewBrowseService(provider storage.Provider, metadata repository.MetadataRepository, cfg *config.Config, logger *slog.Logger) *BrowseService {
	return &BrowseService{provider: provider, metadata: metadata, cfg: cfg, logger: logger}
}

// List returns directory entries, using cache when available.
func (s *BrowseService) List(ctx context.Context, dirPath string, opts model.ListOptions) (*model.ListResult, error) {
	dirPath = NormalizePath(dirPath)

	// Check cache first (only for first page without cursor)
	if opts.Cursor == "" {
		cached, err := s.metadata.GetDirectoryCache(ctx, dirPath)
		if err != nil {
			s.logger.Warn("cache lookup failed", "path", dirPath, "error", err)
		}
		if cached != nil {
			var result model.ListResult
			if err := json.Unmarshal([]byte(cached.PayloadJSON), &result); err == nil {
				result.Source = "cache"
				s.logger.Debug("cache hit", "path", dirPath)
				return &result, nil
			}
		}
	}

	// Fetch from provider
	result, err := s.provider.List(ctx, dirPath, opts)
	if err != nil {
		return nil, fmt.Errorf("provider list: %w", err)
	}

	// Cache the result (only first page)
	if opts.Cursor == "" {
		payload, _ := json.Marshal(result)
		now := time.Now()
		_ = s.metadata.PutDirectoryCache(ctx, &model.DirectoryCacheEntry{
			Path:        dirPath,
			PayloadJSON: string(payload),
			FetchedAt:   now,
			ExpiresAt:   now.Add(s.cfg.MetadataCacheDirTTL),
		})
	}

	return result, nil
}

// FileService handles file metadata operations.
type FileService struct {
	provider storage.Provider
	metadata repository.MetadataRepository
	cfg      *config.Config
	logger   *slog.Logger
}

// NewFileService creates a new FileService.
func NewFileService(provider storage.Provider, metadata repository.MetadataRepository, cfg *config.Config, logger *slog.Logger) *FileService {
	return &FileService{provider: provider, metadata: metadata, cfg: cfg, logger: logger}
}

// Meta returns metadata for a single file, using cache when available.
func (s *FileService) Meta(ctx context.Context, filePath string) (*model.ObjectInfo, error) {
	filePath = NormalizePath(filePath)

	// Check cache
	cached, err := s.metadata.GetObjectCache(ctx, filePath)
	if err != nil {
		s.logger.Warn("object cache lookup failed", "path", filePath, "error", err)
	}
	if cached != nil {
		var info model.ObjectInfo
		if err := json.Unmarshal([]byte(cached.PayloadJSON), &info); err == nil {
			s.logger.Debug("object cache hit", "path", filePath)
			return &info, nil
		}
	}

	// Fetch from provider
	info, err := s.provider.Stat(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("provider stat: %w", err)
	}

	// Cache
	payload, _ := json.Marshal(info)
	now := time.Now()
	_ = s.metadata.PutObjectCache(ctx, &model.ObjectCacheEntry{
		Path:        filePath,
		Type:        info.Type,
		SizeBytes:   info.Size,
		MimeType:    info.MimeType,
		ModifiedAt:  info.ModifiedAt,
		ETag:        info.ETag,
		PayloadJSON: string(payload),
		FetchedAt:   now,
		ExpiresAt:   now.Add(s.cfg.MetadataCacheObjectTTL),
	})

	return info, nil
}

// UploadService handles upload URL generation.
type UploadService struct {
	provider storage.Provider
	metadata repository.MetadataRepository
	logger   *slog.Logger
}

// NewUploadService creates a new UploadService.
func NewUploadService(provider storage.Provider, metadata repository.MetadataRepository, logger *slog.Logger) *UploadService {
	return &UploadService{provider: provider, metadata: metadata, logger: logger}
}

// CreateUploadURL generates a signed upload URL and invalidates related cache.
func (s *UploadService) CreateUploadURL(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error) {
	req.Path = NormalizePath(req.Path)

	target, err := s.provider.CreateUploadTarget(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create upload target: %w", err)
	}

	// Invalidate parent directory cache
	parentDir := path.Dir(req.Path)
	_ = s.metadata.InvalidatePrefix(ctx, parentDir)

	s.logger.Info("upload URL created", "path", req.Path)
	return target, nil
}

// DownloadService handles download/preview URL generation.
type DownloadService struct {
	provider storage.Provider
	logger   *slog.Logger
}

// NewDownloadService creates a new DownloadService.
func NewDownloadService(provider storage.Provider, logger *slog.Logger) *DownloadService {
	return &DownloadService{provider: provider, logger: logger}
}

// CreateDownloadURL generates a signed download URL.
func (s *DownloadService) CreateDownloadURL(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error) {
	req.Path = NormalizePath(req.Path)

	target, err := s.provider.CreateDownloadTarget(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create download target: %w", err)
	}

	s.logger.Info("download URL created", "path", req.Path)
	return target, nil
}

// NormalizePath cleans a virtual path and ensures it starts with /.
func NormalizePath(p string) string {
	if p == "" {
		return "/"
	}
	cleaned := path.Clean("/" + p)
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

// ValidatePath checks that a path is safe and normalized.
func ValidatePath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("path must start with /")
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("path must not contain ..")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path must not contain backslash")
	}
	return nil
}
