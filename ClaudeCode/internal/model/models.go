package model

import "time"

// User represents an authenticated user
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Session represents a user session
type Session struct {
	ID         string
	UserID     string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// ObjectType represents the type of storage object
type ObjectType string

const (
	ObjectTypeFile      ObjectType = "file"
	ObjectTypeDirectory ObjectType = "directory"
)

// ObjectInfo represents metadata about a storage object
type ObjectInfo struct {
	Name         string
	Path         string
	Type         ObjectType
	Size         int64
	MimeType     string
	ModifiedAt   time.Time
	ETag         string
	Previewable  bool
}

// ListResult represents a directory listing result
type ListResult struct {
	Path       string
	Entries    []ObjectInfo
	NextCursor string
	Source     string // "cache" or "provider"
}

// DirectoryCacheEntry represents a cached directory listing
type DirectoryCacheEntry struct {
	Path        string
	PayloadJSON string
	SourceETag  string
	FetchedAt   time.Time
	ExpiresAt   time.Time
}

// ObjectCacheEntry represents cached object metadata
type ObjectCacheEntry struct {
	Path        string
	Type        ObjectType
	SizeBytes   int64
	MimeType    string
	ModifiedAt  time.Time
	ETag        string
	PayloadJSON string
	FetchedAt   time.Time
	ExpiresAt   time.Time
}

// UploadRequest represents a request to generate an upload URL
type UploadRequest struct {
	Path     string
	Size     int64
	MimeType string
}

// UploadTarget represents upload instructions for the client
type UploadTarget struct {
	Provider  string
	Method    string
	URL       string
	Headers   map[string]string
	ExpiresAt time.Time
}

// DownloadRequest represents a request to generate a download URL
type DownloadRequest struct {
	Path string
}

// DownloadTarget represents download instructions for the client
type DownloadTarget struct {
	URL       string
	ExpiresAt time.Time
}

// ListOptions contains options for listing objects
type ListOptions struct {
	Cursor string
	Limit  int
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
