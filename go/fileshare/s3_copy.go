package fileshare

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
)

// CopyS3SharerConfig holds configuration for the external S3 bucket
type CopyS3SharerConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
}

// CopyS3Sharer copies files to an external S3 bucket and generates presigned URLs.
// Use this when the primary storage is not publicly accessible.
type CopyS3Sharer struct {
	storage        FileStorage
	client         *minio.Client
	bucket         string
	endpoint       string
	publicEndpoint string
}

// NewCopyS3Sharer creates a new CopyS3Sharer
func NewCopyS3Sharer(s FileStorage, config CopyS3SharerConfig) (*CopyS3Sharer, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create minio client: %w", err)
	}

	sharer := &CopyS3Sharer{
		storage:        s,
		client:         client,
		bucket:         config.Bucket,
		endpoint:       config.Endpoint,
		publicEndpoint: config.PublicEndpoint,
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("cannot check if bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("cannot create bucket: %w", err)
		}
		log.Info().Str("bucket", config.Bucket).Msg("Created share bucket")
	}

	return sharer, nil
}

// ShareSessionFile copies a session file to external S3 and returns a presigned URL
func (c *CopyS3Sharer) ShareSessionFile(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (ShareResult, error) {
	// Read file from primary storage
	reader, size, err := c.storage.GetSessionFileReader(ctx, asset)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot get file from storage: %w", err)
	}
	defer reader.Close()

	// Generate unique object name with timestamp
	objectName := fmt.Sprintf("sessions/%s/%s/%s/%s",
		asset.RecorderID.String(),
		asset.SessionID.String(),
		uuid.New().String()[:8],
		asset.Filename)

	// Upload to external S3
	_, err = c.client.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot upload to share bucket: %w", err)
	}

	log.Debug().
		Str("object", objectName).
		Int64("size", size).
		Msg("Uploaded file to share bucket")

	// Generate presigned URL
	return c.generatePresignedURL(ctx, objectName, options)
}

// ShareSegmentFile copies a segment file to external S3 and returns a presigned URL
func (c *CopyS3Sharer) ShareSegmentFile(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (ShareResult, error) {
	// Read file from primary storage
	reader, size, err := c.storage.GetSegmentFileReader(ctx, asset)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot get file from storage: %w", err)
	}
	defer reader.Close()

	// Generate unique object name with timestamp
	objectName := fmt.Sprintf("segments/%s/%s/%s/%s/%s",
		asset.RecorderID.String(),
		asset.SessionID.String(),
		asset.SegmentID.String(),
		uuid.New().String()[:8],
		asset.Filename)

	// Upload to external S3
	_, err = c.client.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot upload to share bucket: %w", err)
	}

	log.Debug().
		Str("object", objectName).
		Int64("size", size).
		Msg("Uploaded segment to share bucket")

	// Generate presigned URL
	return c.generatePresignedURL(ctx, objectName, options)
}

func (c *CopyS3Sharer) generatePresignedURL(ctx context.Context, objectName string, options storage.SigningOptions) (ShareResult, error) {
	values := make(url.Values)

	if options.Download {
		filename := options.DownloadFilename
		if filename == "" {
			// Extract filename from object path
			parts := strings.Split(objectName, "/")
			filename = parts[len(parts)-1]
		}
		values.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	presignedURL, err := c.client.PresignedGetObject(ctx, c.bucket, objectName, options.Expires, values)
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Replace internal endpoint with public endpoint
	publicURL := presignedURL.String()
	if c.publicEndpoint != "" && c.publicEndpoint != c.endpoint {
		publicURL = strings.Replace(publicURL, c.endpoint, c.publicEndpoint, 1)
	}

	return ShareResult{
		URL:       publicURL,
		ExpiresAt: time.Now().Add(options.Expires),
	}, nil
}
