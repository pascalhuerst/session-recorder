package fileshare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
)

// FileIOSharerConfig holds configuration for the file.io service
type FileIOSharerConfig struct {
	Endpoint string // Default: https://file.io
}

// FileIOSharer uploads files to file.io and returns the share link.
// Files on file.io are deleted after first download or after expiration.
type FileIOSharer struct {
	storage  FileStorage
	endpoint string
	client   *http.Client
}

// fileIOResponse represents the response from file.io API
type fileIOResponse struct {
	Success bool   `json:"success"`
	Status  int    `json:"status"`
	Key     string `json:"key"`
	Link    string `json:"link"`
	Expiry  string `json:"expiry"`
	Message string `json:"message"`
}

// NewFileIOSharer creates a new FileIOSharer
func NewFileIOSharer(s FileStorage, config FileIOSharerConfig) *FileIOSharer {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://file.io"
	}

	return &FileIOSharer{
		storage:  s,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 5 * time.Minute, // Large file uploads may take time
		},
	}
}

// ShareSessionFile uploads a session file to file.io and returns the share link
func (f *FileIOSharer) ShareSessionFile(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (ShareResult, error) {
	// Read file from primary storage
	reader, size, err := f.storage.GetSessionFileReader(ctx, asset)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot get file from storage: %w", err)
	}
	defer reader.Close()

	// Determine filename
	filename := options.DownloadFilename
	if filename == "" {
		filename = string(asset.Filename)
	}

	return f.uploadToFileIO(ctx, reader, size, filename, options.Expires)
}

// ShareSegmentFile uploads a segment file to file.io and returns the share link
func (f *FileIOSharer) ShareSegmentFile(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (ShareResult, error) {
	// Read file from primary storage
	reader, size, err := f.storage.GetSegmentFileReader(ctx, asset)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot get file from storage: %w", err)
	}
	defer reader.Close()

	// Determine filename
	filename := options.DownloadFilename
	if filename == "" {
		filename = string(asset.Filename)
	}

	return f.uploadToFileIO(ctx, reader, size, filename, options.Expires)
}

func (f *FileIOSharer) uploadToFileIO(ctx context.Context, reader io.Reader, size int64, filename string, expires time.Duration) (ShareResult, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file field
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot create form file: %w", err)
	}

	// Copy file content to form
	if _, err := io.Copy(part, reader); err != nil {
		return ShareResult{}, fmt.Errorf("cannot copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return ShareResult{}, fmt.Errorf("cannot close multipart writer: %w", err)
	}

	// Calculate expiry in days (file.io uses days by default)
	expiryDays := int(expires.Hours()/24 + 0.5) // Round to nearest day
	if expiryDays < 1 {
		expiryDays = 1
	}

	// Build request URL with expiry
	url := fmt.Sprintf("%s/?expires=%dd", f.endpoint, expiryDays)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot upload to file.io: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result fileIOResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ShareResult{}, fmt.Errorf("cannot parse file.io response: %w", err)
	}

	if !result.Success {
		return ShareResult{}, fmt.Errorf("file.io upload failed: %s", result.Message)
	}

	log.Debug().
		Str("key", result.Key).
		Str("link", result.Link).
		Str("expiry", result.Expiry).
		Int64("size", size).
		Msg("Uploaded file to file.io")

	return ShareResult{
		URL:       result.Link,
		ExpiresAt: time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour),
	}, nil
}
