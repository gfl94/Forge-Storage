package azure

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
)

// Provider implements the storage.Provider interface for Azure Blob Storage
type Provider struct {
	client         *service.Client
	containerName  string
	accountName    string
	signedURLTTL   time.Duration
}

// NewProvider creates a new Azure Blob Storage provider
func NewProvider(accountName, accountKey, containerName string, signedURLTTL time.Duration) (*Provider, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	client, err := service.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Provider{
		client:        client,
		containerName: containerName,
		accountName:   accountName,
		signedURLTTL:  signedURLTTL,
	}, nil
}

// normalizePath converts a user path to an Azure blob name
func (p *Provider) normalizePath(path string) string {
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")
	return path
}

// denormalizePath converts an Azure blob name back to user path
func (p *Provider) denormalizePath(blobName string) string {
	if !strings.HasPrefix(blobName, "/") {
		return "/" + blobName
	}
	return blobName
}

// List lists objects at a given path
func (p *Provider) List(ctx context.Context, path string, opts model.ListOptions) (*model.ListResult, error) {
	prefix := p.normalizePath(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	containerClient := p.client.NewContainerClient(p.containerName)
	pager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var entries []model.ObjectInfo
	seen := make(map[string]bool)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}

			blobPath := p.denormalizePath(*blob.Name)

			// Skip if this is the prefix itself
			if strings.TrimSuffix(blobPath, "/") == strings.TrimSuffix(path, "/") {
				continue
			}

			// Extract the relative path after the prefix
			relativePath := strings.TrimPrefix(*blob.Name, prefix)

			// Check if this is a direct child or a nested item
			parts := strings.Split(strings.Trim(relativePath, "/"), "/")

			if len(parts) > 1 {
				// This is a nested item, add as a directory if not already seen
				dirName := parts[0]
				dirPath := path
				if !strings.HasSuffix(dirPath, "/") {
					dirPath += "/"
				}
				dirPath += dirName

				if !seen[dirPath] {
					seen[dirPath] = true
					entries = append(entries, model.ObjectInfo{
						Name: dirName,
						Path: dirPath,
						Type: model.ObjectTypeDirectory,
					})
				}
			} else {
				// This is a direct file
				var size int64
				if blob.Properties != nil && blob.Properties.ContentLength != nil {
					size = *blob.Properties.ContentLength
				}

				var mimeType string
				if blob.Properties != nil && blob.Properties.ContentType != nil {
					mimeType = *blob.Properties.ContentType
				} else {
					mimeType = mime.TypeByExtension(filepath.Ext(*blob.Name))
				}

				var modifiedAt time.Time
				if blob.Properties != nil && blob.Properties.LastModified != nil {
					modifiedAt = *blob.Properties.LastModified
				}

				var etag string
				if blob.Properties != nil && blob.Properties.ETag != nil {
					etag = string(*blob.Properties.ETag)
				}

				entries = append(entries, model.ObjectInfo{
					Name:       filepath.Base(*blob.Name),
					Path:       blobPath,
					Type:       model.ObjectTypeFile,
					Size:       size,
					MimeType:   mimeType,
					ModifiedAt: modifiedAt,
					ETag:       etag,
					Previewable: isPreviewable(mimeType),
				})
			}
		}
	}

	return &model.ListResult{
		Path:       path,
		Entries:    entries,
		NextCursor: "",
		Source:     "provider",
	}, nil
}

// Stat returns metadata for a single object
func (p *Provider) Stat(ctx context.Context, path string) (*model.ObjectInfo, error) {
	blobName := p.normalizePath(path)

	containerClient := p.client.NewContainerClient(p.containerName)
	blobClient := containerClient.NewBlobClient(blobName)

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob properties: %w", err)
	}

	var size int64
	if props.ContentLength != nil {
		size = *props.ContentLength
	}

	var mimeType string
	if props.ContentType != nil {
		mimeType = *props.ContentType
	} else {
		mimeType = mime.TypeByExtension(filepath.Ext(blobName))
	}

	var modifiedAt time.Time
	if props.LastModified != nil {
		modifiedAt = *props.LastModified
	}

	var etag string
	if props.ETag != nil {
		etag = string(*props.ETag)
	}

	return &model.ObjectInfo{
		Name:        filepath.Base(blobName),
		Path:        path,
		Type:        model.ObjectTypeFile,
		Size:        size,
		MimeType:    mimeType,
		ModifiedAt:  modifiedAt,
		ETag:        etag,
		Previewable: isPreviewable(mimeType),
	}, nil
}

// CreateUploadTarget generates signed upload instructions
func (p *Provider) CreateUploadTarget(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error) {
	blobName := p.normalizePath(req.Path)

	containerClient := p.client.NewContainerClient(p.containerName)
	blobClient := containerClient.NewBlobClient(blobName)

	// Create SAS token for upload
	now := time.Now()
	expiresAt := now.Add(p.signedURLTTL)

	sasURL, err := blobClient.GetSASURL(
		sas.BlobPermissions{Write: true, Create: true},
		expiresAt,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SAS URL: %w", err)
	}

	headers := map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}

	if req.MimeType != "" {
		headers["Content-Type"] = req.MimeType
	}

	return &model.UploadTarget{
		Provider:  "azure",
		Method:    "PUT",
		URL:       sasURL,
		Headers:   headers,
		ExpiresAt: expiresAt,
	}, nil
}

// CreateDownloadTarget generates signed download URL
func (p *Provider) CreateDownloadTarget(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error) {
	blobName := p.normalizePath(req.Path)

	containerClient := p.client.NewContainerClient(p.containerName)
	blobClient := containerClient.NewBlobClient(blobName)

	// Create SAS token for download
	now := time.Now()
	expiresAt := now.Add(p.signedURLTTL)

	sasURL, err := blobClient.GetSASURL(
		sas.BlobPermissions{Read: true},
		expiresAt,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SAS URL: %w", err)
	}

	return &model.DownloadTarget{
		URL:       sasURL,
		ExpiresAt: expiresAt,
	}, nil
}

// Delete removes an object
func (p *Provider) Delete(ctx context.Context, path string) error {
	blobName := p.normalizePath(path)

	containerClient := p.client.NewContainerClient(p.containerName)
	blobClient := containerClient.NewBlobClient(blobName)

	_, err := blobClient.Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	return nil
}

// isPreviewable determines if a file type can be previewed in the browser
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
