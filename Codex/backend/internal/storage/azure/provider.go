package azure

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/storage"
)

// Provider implements the storage.Provider interface for Azure Blob Storage.
type Provider struct {
	containerCli *container.Client
	cred         *azblob.SharedKeyCredential
	container    string
	account      string
	accountURL   string
}

// New creates a new Azure provider instance.
func New(account, key, containerName string) (*Provider, error) {
	cred, err := azblob.NewSharedKeyCredential(account, key)
	if err != nil {
		return nil, fmt.Errorf("azure shared key: %w", err)
	}
	baseURL := fmt.Sprintf("https://%s.blob.core.windows.net", account)
	containerURL := fmt.Sprintf("%s/%s", baseURL, containerName)
	containerClient, err := container.NewClientWithSharedKeyCredential(containerURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure container client: %w", err)
	}
	return &Provider{
		containerCli: containerClient,
		cred:         cred,
		container:    containerName,
		account:      account,
		accountURL:   baseURL,
	}, nil
}

// List returns objects under a prefix with directory semantics.
func (p *Provider) List(ctx context.Context, opts storage.ListOptions) (storage.ListResult, error) {
	prefix := strings.TrimPrefix(opts.Path, "/")
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	max := opts.Limit
	if max <= 0 {
		max = 100
	}

	pager := p.containerCli.NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{
		Prefix:     to.Ptr(prefix),
		Marker:     optionalMarker(opts.Cursor),
		MaxResults: to.Ptr(max),
	})

	var items []storage.ListItem
	var nextCursor *string

	if pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return storage.ListResult{}, err
		}

		for _, pref := range page.Segment.BlobPrefixes {
			normalized := "/" + strings.TrimSuffix(*pref.Name, "/")
			items = append(items, storage.ListItem{
				Path:  normalized,
				IsDir: true,
			})
		}

		for _, blob := range page.Segment.BlobItems {
			modified := timePtr(blob.Properties.LastModified)
			mime := ""
			if blob.Properties.ContentType != nil {
				mime = *blob.Properties.ContentType
			}
			etag := ""
			if blob.Properties.ETag != nil {
				etag = string(*blob.Properties.ETag)
			}
			items = append(items, storage.ListItem{
				Path:       "/" + *blob.Name,
				IsDir:      false,
				Size:       toSafeInt64(blob.Properties.ContentLength),
				MimeType:   mime,
				ModifiedAt: modified,
				ETag:       etag,
			})
		}

		if page.NextMarker != nil && len(*page.NextMarker) > 0 {
			nextCursor = to.Ptr(*page.NextMarker)
		}
	}

	return storage.ListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

// Stat returns metadata for a blob.
func (p *Provider) Stat(ctx context.Context, path string) (*storage.ObjectInfo, error) {
	blobPath := strings.TrimPrefix(path, "/")
	blobClient := p.containerCli.NewBlobClient(blobPath)
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.ErrorCode == string(bloberror.BlobNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return nil, err
	}
	mime := ""
	if props.ContentType != nil {
		mime = *props.ContentType
	}
	etag := ""
	if props.ETag != nil {
		etag = string(*props.ETag)
	}
	return &storage.ObjectInfo{
		Path:       path,
		Size:       toSafeInt64(props.ContentLength),
		MimeType:   mime,
		ModifiedAt: timePtr(props.LastModified),
		ETag:       etag,
	}, nil
}

// CreateUploadTarget generates a signed upload target.
func (p *Provider) CreateUploadTarget(ctx context.Context, req model.UploadRequest, expiresAt time.Time) (*model.UploadTarget, error) {
	blobName := strings.TrimPrefix(req.Path, "/")
	permissions := sas.BlobPermissions{Create: true, Write: true}
	sig := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().Add(-5 * time.Minute),
		ExpiryTime:    expiresAt,
		Permissions:   permissions.String(),
		ContainerName: p.container,
		BlobName:      blobName,
	}
	query, err := sig.SignWithSharedKey(p.cred)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s?%s", p.accountURL, p.container, blobName, query.Encode())
	headers := map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}
	if req.MimeType != "" {
		headers["Content-Type"] = req.MimeType
	}
	return &model.UploadTarget{
		Provider:  "azure",
		Method:    "PUT",
		URL:       url,
		Headers:   headers,
		ExpiresAt: expiresAt,
	}, nil
}

// CreateDownloadTarget generates a signed download target.
func (p *Provider) CreateDownloadTarget(ctx context.Context, req model.DownloadRequest, expiresAt time.Time) (*model.DownloadTarget, error) {
	blobName := strings.TrimPrefix(req.Path, "/")
	permissions := sas.BlobPermissions{Read: true}
	sig := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().Add(-5 * time.Minute),
		ExpiryTime:    expiresAt,
		Permissions:   permissions.String(),
		ContainerName: p.container,
		BlobName:      blobName,
	}
	query, err := sig.SignWithSharedKey(p.cred)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s?%s", p.accountURL, p.container, blobName, query.Encode())
	return &model.DownloadTarget{
		URL:       url,
		ExpiresAt: expiresAt,
	}, nil
}

// CheckReady ensures the container is reachable.
func (p *Provider) CheckReady(ctx context.Context) error {
	pager := p.containerCli.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		MaxResults: to.Ptr(int32(1)),
	})
	if pager.More() {
		if _, err := pager.NextPage(ctx); err != nil {
			return err
		}
	}
	return nil
}

func optionalMarker(cursor string) *string {
	if cursor == "" {
		return nil
	}
	return &cursor
}

func timePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

func toSafeInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

var _ storage.Provider = (*Provider)(nil)
