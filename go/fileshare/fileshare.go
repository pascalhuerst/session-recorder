package fileshare

import (
	"context"
	"io"
	"time"

	"github.com/pascalhuerst/session-recorder/storage"
)

// ShareResult contains the download URL and expiration time
type ShareResult struct {
	URL       string
	ExpiresAt time.Time
}

// FileStorage defines the minimal storage interface needed by FileSharer implementations.
// This is a subset of storage.Storage to allow easier testing.
type FileStorage interface {
	GetPresignedURL(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (string, error)
	GetSegmentPresignedURL(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (string, error)
	GetSessionFileReader(ctx context.Context, asset storage.AssetOptions) (io.ReadCloser, int64, error)
	GetSegmentFileReader(ctx context.Context, asset storage.SegmentAssetOptions) (io.ReadCloser, int64, error)
}

// FileSharer generates shareable download links for files
type FileSharer interface {
	// ShareSessionFile generates a download URL for a session file
	ShareSessionFile(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (ShareResult, error)

	// ShareSegmentFile generates a download URL for a segment file
	ShareSegmentFile(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (ShareResult, error)
}
