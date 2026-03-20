// Package azure implements the storage.Provider interface for Azure Blob Storage.
package azure

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
)

// Provider implements storage.Provider for Azure Blob Storage.
type Provider struct {
	accountName   string
	containerName string
	accountKey    string
	signedURLTTL  time.Duration

	containerURL azblob.ContainerURL
	credential   *azblob.SharedKeyCredential
}

// New creates a new Azure Blob Storage provider.
func New(accountName, containerName, accountKey string, signedURLTTL time.Duration) (*Provider, error) {
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("azure credential: %w", err)
	}

	pipeline := azblob.NewPipeline(cred, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	containerURL := azblob.NewContainerURL(*u, pipeline)

	return &Provider{
		accountName:   accountName,
		containerName: containerName,
		accountKey:    accountKey,
		signedURLTTL:  signedURLTTL,
		containerURL:  containerURL,
		credential:    cred,
	}, nil
}

// normalizePath strips leading slash for Azure blob names.
func normalizePath(p string) string {
	return strings.TrimPrefix(path.Clean("/"+p), "/")
}

// List lists objects under the given prefix path.
func (p *Provider) List(ctx context.Context, dirPath string, opts model.ListOptions) (*model.ListResult, error) {
	prefix := normalizePath(dirPath)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	limit := opts.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	marker := azblob.Marker{}
	if opts.Cursor != "" {
		marker.Val = &opts.Cursor
	}

	resp, err := p.containerURL.ListBlobsHierarchySegment(ctx, marker, "/", azblob.ListBlobsSegmentOptions{
		Prefix:     prefix,
		MaxResults: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("azure list: %w", err)
	}

	var entries []model.Entry

	// Virtual directories (prefixes)
	for _, blobPrefix := range resp.Segment.BlobPrefixes {
		name := strings.TrimPrefix(blobPrefix.Name, prefix)
		name = strings.TrimSuffix(name, "/")
		if name == "" {
			continue
		}
		entries = append(entries, model.Entry{
			Name: name,
			Path: "/" + strings.TrimSuffix(blobPrefix.Name, "/"),
			Type: model.EntryTypeDirectory,
		})
	}

	// Blobs (files)
	for _, blob := range resp.Segment.BlobItems {
		name := strings.TrimPrefix(blob.Name, prefix)
		if name == "" {
			continue
		}
		entry := model.Entry{
			Name: name,
			Path: "/" + blob.Name,
			Type: model.EntryTypeFile,
			Size: *blob.Properties.ContentLength,
		}
		if blob.Properties.ContentType != nil {
			entry.MimeType = *blob.Properties.ContentType
		}
		if blob.Properties.LastModified != (time.Time{}) {
			entry.ModifiedAt = blob.Properties.LastModified
		}
		entries = append(entries, entry)
	}

	var nextCursor string
	if resp.NextMarker.Val != nil {
		nextCursor = *resp.NextMarker.Val
	}

	return &model.ListResult{
		Path:       "/" + strings.TrimSuffix(prefix, "/"),
		Entries:    entries,
		NextCursor: nextCursor,
		Source:     "provider",
	}, nil
}

// Stat fetches metadata for a single blob.
func (p *Provider) Stat(ctx context.Context, filePath string) (*model.ObjectInfo, error) {
	blobName := normalizePath(filePath)
	blobURL := p.containerURL.NewBlobURL(blobName)

	props, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, fmt.Errorf("azure stat: %w", err)
	}

	mimeType := props.ContentType()
	info := &model.ObjectInfo{
		Path:        "/" + blobName,
		Type:        model.EntryTypeFile,
		Size:        props.ContentLength(),
		MimeType:    mimeType,
		ModifiedAt:  props.LastModified(),
		ETag:        string(props.ETag()),
		Previewable: isPreviewable(mimeType),
	}
	return info, nil
}

// CreateUploadTarget generates a SAS URL for uploading a blob.
func (p *Provider) CreateUploadTarget(ctx context.Context, req model.UploadRequest) (*model.UploadTarget, error) {
	blobName := normalizePath(req.Path)
	blobURL := p.containerURL.NewBlockBlobURL(blobName)

	now := time.Now().UTC()
	expiry := now.Add(p.signedURLTTL)

	sasQueryParams, err := azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    expiry,
		Permissions:   azblob.BlobSASPermissions{Write: true, Create: true}.String(),
		ContainerName: p.containerName,
		BlobName:      blobName,
	}.NewSASQueryParameters(p.credential)
	if err != nil {
		return nil, fmt.Errorf("azure upload sas: %w", err)
	}

	parts := azblob.NewBlobURLParts(blobURL.URL())
	parts.SAS = sasQueryParams
	sasURL := parts.URL()

	headers := map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}
	if req.MimeType != "" {
		headers["Content-Type"] = req.MimeType
	}

	return &model.UploadTarget{
		Provider:  "azure",
		Method:    "PUT",
		URL:       sasURL.String(),
		Headers:   headers,
		ExpiresAt: expiry,
	}, nil
}

// CreateDownloadTarget generates a SAS URL for downloading a blob.
func (p *Provider) CreateDownloadTarget(ctx context.Context, req model.DownloadRequest) (*model.DownloadTarget, error) {
	blobName := normalizePath(req.Path)
	blobURL := p.containerURL.NewBlobURL(blobName)

	now := time.Now().UTC()
	expiry := now.Add(p.signedURLTTL)

	sasQueryParams, err := azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    expiry,
		Permissions:   azblob.BlobSASPermissions{Read: true}.String(),
		ContainerName: p.containerName,
		BlobName:      blobName,
	}.NewSASQueryParameters(p.credential)
	if err != nil {
		return nil, fmt.Errorf("azure download sas: %w", err)
	}

	parts := azblob.NewBlobURLParts(blobURL.URL())
	parts.SAS = sasQueryParams
	sasURL := parts.URL()

	return &model.DownloadTarget{
		URL:       sasURL.String(),
		ExpiresAt: expiry,
	}, nil
}

// Ping checks that the container is accessible.
func (p *Provider) Ping(ctx context.Context) error {
	_, err := p.containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if err != nil {
		return fmt.Errorf("azure ping: %w", err)
	}
	return nil
}

// isPreviewable determines if a MIME type can be previewed in a browser.
func isPreviewable(mimeType string) bool {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return true
	case mimeType == "application/pdf":
		return true
	case strings.HasPrefix(mimeType, "text/"):
		return true
	case mimeType == "video/mp4":
		return true
	default:
		return false
	}
}
