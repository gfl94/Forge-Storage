package model

import "time"

// User represents an application user.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Session represents an authenticated browser session.
type Session struct {
	ID         string
	UserID     string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// DirectoryEntry describes an item within a directory listing.
type DirectoryEntry struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Type       string     `json:"type"` // "directory" or "file"
	Size       int64      `json:"size,omitempty"`
	MimeType   string     `json:"mimeType,omitempty"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
}

// DirectoryListing is the response payload for a directory list operation.
type DirectoryListing struct {
	Path       string           `json:"path"`
	Entries    []DirectoryEntry `json:"entries"`
	NextCursor *string          `json:"nextCursor,omitempty"`
	Source     string           `json:"source"` // "cache" or "provider"
}

// ObjectMetadata represents normalized metadata for a single object.
type ObjectMetadata struct {
	Path        string     `json:"path"`
	Type        string     `json:"type"` // "file"
	Size        int64      `json:"size,omitempty"`
	MimeType    string     `json:"mimeType,omitempty"`
	ModifiedAt  *time.Time `json:"modifiedAt,omitempty"`
	ETag        string     `json:"etag,omitempty"`
	Previewable bool       `json:"previewable"`
}

// UploadRequest describes an upload target request.
type UploadRequest struct {
	Path     string
	Size     int64
	MimeType string
}

// UploadTarget contains the data the browser needs to upload directly to storage.
type UploadTarget struct {
	Provider  string            `json:"provider"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

// DownloadRequest describes a download target request.
type DownloadRequest struct {
	Path string
}

// DownloadTarget contains the data the browser needs to download or preview.
type DownloadTarget struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// DirectoryCacheEntry is the cached representation of a directory listing.
type DirectoryCacheEntry struct {
	Path       string
	Payload    []byte
	SourceETag string
	FetchedAt  time.Time
	ExpiresAt  time.Time
}

// ObjectCacheEntry is the cached representation of an object metadata payload.
type ObjectCacheEntry struct {
	Path      string
	Type      string
	SizeBytes int64
	MimeType  string
	Modified  *time.Time
	ETag      string
	Payload   []byte
	FetchedAt time.Time
	ExpiresAt time.Time
}
