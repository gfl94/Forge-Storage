// Package model defines the core domain types shared across the application.
package model

import "time"

// User represents an authenticated user.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Session represents an active user session.
type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// EntryType distinguishes files from directories.
type EntryType string

const (
	EntryTypeFile      EntryType = "file"
	EntryTypeDirectory EntryType = "directory"
)

// Entry is a single item (file or directory) returned in a listing.
type Entry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       EntryType `json:"type"`
	Size       int64     `json:"size,omitempty"`
	MimeType   string    `json:"mimeType,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt,omitempty"`
}

// ObjectInfo holds detailed metadata about a single object.
type ObjectInfo struct {
	Path        string    `json:"path"`
	Type        EntryType `json:"type"`
	Size        int64     `json:"size"`
	MimeType    string    `json:"mimeType"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	ETag        string    `json:"etag"`
	Previewable bool      `json:"previewable"`
}

// ListOptions controls listing behavior.
type ListOptions struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// ListResult is the output of a provider list operation.
type ListResult struct {
	Path       string  `json:"path"`
	Entries    []Entry `json:"entries"`
	NextCursor string  `json:"nextCursor"`
	Source     string  `json:"source"` // "cache" or "provider"
}

// UploadRequest describes what the client wants to upload.
type UploadRequest struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
}

// UploadTarget is the signed upload instruction returned to the browser.
type UploadTarget struct {
	Provider  string            `json:"provider"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

// DownloadRequest describes what the client wants to download.
type DownloadRequest struct {
	Path string `json:"path"`
}

// DownloadTarget is the signed download URL returned to the browser.
type DownloadTarget struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// DirectoryCacheEntry stores a cached directory listing in the database.
type DirectoryCacheEntry struct {
	Path        string    `json:"path"`
	PayloadJSON string    `json:"payloadJson"`
	SourceETag  string    `json:"sourceEtag,omitempty"`
	FetchedAt   time.Time `json:"fetchedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// ObjectCacheEntry stores cached object metadata in the database.
type ObjectCacheEntry struct {
	Path        string    `json:"path"`
	Type        EntryType `json:"type"`
	SizeBytes   int64     `json:"sizeBytes"`
	MimeType    string    `json:"mimeType"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	ETag        string    `json:"etag"`
	PayloadJSON string    `json:"payloadJson"`
	FetchedAt   time.Time `json:"fetchedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// AuditEvent records a user action for auditing purposes.
type AuditEvent struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId,omitempty"`
	Action       string    `json:"action"`
	Path         string    `json:"path,omitempty"`
	MetadataJSON string    `json:"metadataJson,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
