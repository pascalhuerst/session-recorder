package fileshare

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// ShareMethod represents the file sharing method to use
type ShareMethod string

const (
	ShareMethodDirect ShareMethod = "direct"
	ShareMethodS3Copy ShareMethod = "s3_copy"
	ShareMethodFileIO ShareMethod = "fileio"
)

// NewFileSharer creates a FileSharer based on environment configuration.
//
// Environment variables:
//   - FILE_SHARE_METHOD: "direct" (default), "s3_copy", or "fileio"
//
// For s3_copy method:
//   - FILE_SHARE_S3_ENDPOINT: S3 endpoint
//   - FILE_SHARE_S3_PUBLIC_ENDPOINT: Public endpoint for presigned URLs
//   - FILE_SHARE_S3_ACCESS_KEY: S3 access key
//   - FILE_SHARE_S3_SECRET_KEY: S3 secret key
//   - FILE_SHARE_S3_BUCKET: Bucket name (default: "shared-files")
//   - FILE_SHARE_S3_USE_SSL: Use SSL (default: "true")
//
// For fileio method:
//   - FILE_SHARE_FILEIO_ENDPOINT: file.io endpoint (default: "https://file.io")
func NewFileSharer(s FileStorage) (FileSharer, error) {
	method := ShareMethod(strings.ToLower(getEnv("FILE_SHARE_METHOD", "direct")))

	switch method {
	case ShareMethodDirect, "":
		log.Info().Msg("Using direct S3 presigned URLs for file sharing")
		return NewDirectS3Sharer(s), nil

	case ShareMethodS3Copy:
		log.Info().Msg("Using S3 copy for file sharing")
		return newS3CopySharer(s)

	case ShareMethodFileIO:
		log.Info().Msg("Using file.io for file sharing")
		return newFileIOSharer(s), nil

	default:
		return nil, fmt.Errorf("unknown file share method: %s", method)
	}
}

func newS3CopySharer(s FileStorage) (*CopyS3Sharer, error) {
	endpoint := os.Getenv("FILE_SHARE_S3_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("FILE_SHARE_S3_ENDPOINT is required for s3_copy method")
	}

	accessKey := os.Getenv("FILE_SHARE_S3_ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf("FILE_SHARE_S3_ACCESS_KEY is required for s3_copy method")
	}

	secretKey := os.Getenv("FILE_SHARE_S3_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("FILE_SHARE_S3_SECRET_KEY is required for s3_copy method")
	}

	config := CopyS3SharerConfig{
		Endpoint:       endpoint,
		PublicEndpoint: getEnv("FILE_SHARE_S3_PUBLIC_ENDPOINT", endpoint),
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		Bucket:         getEnv("FILE_SHARE_S3_BUCKET", "shared-files"),
		UseSSL:         getEnv("FILE_SHARE_S3_USE_SSL", "true") == "true",
	}

	return NewCopyS3Sharer(s, config)
}

func newFileIOSharer(s FileStorage) *FileIOSharer {
	config := FileIOSharerConfig{
		Endpoint: getEnv("FILE_SHARE_FILEIO_ENDPOINT", "https://file.io"),
	}

	return NewFileIOSharer(s, config)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
