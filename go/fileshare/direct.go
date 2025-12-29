package fileshare

import (
	"context"
	"time"

	"github.com/pascalhuerst/session-recorder/storage"
)

// DirectS3Sharer uses the storage's presigned URL capability directly.
// Use this when the S3/MinIO instance is accessible via the web.
type DirectS3Sharer struct {
	storage FileStorage
}

// NewDirectS3Sharer creates a new DirectS3Sharer
func NewDirectS3Sharer(s FileStorage) *DirectS3Sharer {
	return &DirectS3Sharer{storage: s}
}

// ShareSessionFile generates a presigned URL for a session file
func (d *DirectS3Sharer) ShareSessionFile(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (ShareResult, error) {
	url, err := d.storage.GetPresignedURL(ctx, asset, options)
	if err != nil {
		return ShareResult{}, err
	}

	return ShareResult{
		URL:       url,
		ExpiresAt: time.Now().Add(options.Expires),
	}, nil
}

// ShareSegmentFile generates a presigned URL for a segment file
func (d *DirectS3Sharer) ShareSegmentFile(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (ShareResult, error) {
	url, err := d.storage.GetSegmentPresignedURL(ctx, asset, options)
	if err != nil {
		return ShareResult{}, err
	}

	return ShareResult{
		URL:       url,
		ExpiresAt: time.Now().Add(options.Expires),
	}, nil
}
