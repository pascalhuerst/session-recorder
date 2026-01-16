package fileshare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
)

// DropboxConfig holds configuration for Dropbox sharing
type DropboxConfig struct {
	AccessToken string // Dropbox API access token
	FolderPath  string // Folder path in Dropbox (e.g., "/SessionRecorder")
}

// DropboxSharer uploads files to Dropbox and returns shareable links
type DropboxSharer struct {
	storage     FileStorage
	accessToken string
	folderPath  string
	client      *http.Client
}

// NewDropboxSharer creates a new Dropbox sharer
func NewDropboxSharer(s FileStorage, config DropboxConfig) *DropboxSharer {
	folderPath := config.FolderPath
	if folderPath == "" {
		folderPath = "/SessionRecorder"
	}

	return &DropboxSharer{
		storage:     s,
		accessToken: config.AccessToken,
		folderPath:  folderPath,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// ShareSessionFile uploads a session file to Dropbox and returns the share link
func (d *DropboxSharer) ShareSessionFile(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (ShareResult, error) {
	log.Debug().
		Str("recorder-id", asset.RecorderID.String()).
		Str("session-id", asset.SessionID.String()).
		Str("filename", string(asset.Filename)).
		Msg("Dropbox: Starting session file share")

	reader, size, err := d.storage.GetSessionFileReader(ctx, asset)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot get file from storage: %w", err)
	}
	defer reader.Close()

	log.Debug().
		Int64("size", size).
		Msg("Dropbox: Got file reader from storage")

	filename := options.DownloadFilename
	if filename == "" {
		filename = string(asset.Filename)
	}

	return d.uploadAndShare(ctx, reader, size, filename, options.Expires)
}

// ShareSegmentFile uploads a segment file to Dropbox and returns the share link
func (d *DropboxSharer) ShareSegmentFile(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (ShareResult, error) {
	reader, size, err := d.storage.GetSegmentFileReader(ctx, asset)
	if err != nil {
		return ShareResult{}, fmt.Errorf("cannot get file from storage: %w", err)
	}
	defer reader.Close()

	filename := options.DownloadFilename
	if filename == "" {
		filename = string(asset.Filename)
	}

	return d.uploadAndShare(ctx, reader, size, filename, options.Expires)
}

type dropboxUploadArg struct {
	Path       string `json:"path"`
	Mode       string `json:"mode"`
	AutoRename bool   `json:"autorename"`
	Mute       bool   `json:"mute"`
}

type dropboxUploadResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path_display"`
}

type dropboxShareArg struct {
	Path     string                  `json:"path"`
	Settings dropboxShareSettings    `json:"settings"`
}

type dropboxShareSettings struct {
	RequestedVisibility string `json:"requested_visibility"`
	Audience            string `json:"audience"`
	Access              string `json:"access"`
}

type dropboxShareResponse struct {
	URL string `json:"url"`
}

type dropboxError struct {
	ErrorSummary string `json:"error_summary"`
}

func (d *DropboxSharer) uploadAndShare(ctx context.Context, reader io.Reader, size int64, filename string, expires time.Duration) (ShareResult, error) {
	// Build the destination path
	destPath := filepath.Join(d.folderPath, filepath.Base(filename))

	log.Debug().
		Str("dest-path", destPath).
		Str("filename", filename).
		Int64("size", size).
		Msg("Dropbox: Reading file content for upload")

	// Read file content
	content, err := io.ReadAll(reader)
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to read file content: %w", err)
	}

	log.Debug().
		Int("content-length", len(content)).
		Msg("Dropbox: Starting upload to Dropbox API")

	// Upload file to Dropbox
	uploadArg := dropboxUploadArg{
		Path:       destPath,
		Mode:       "overwrite",
		AutoRename: true,
		Mute:       true,
	}
	uploadArgJSON, _ := json.Marshal(uploadArg)

	uploadReq, err := http.NewRequestWithContext(ctx, "POST", "https://content.dropboxapi.com/2/files/upload", bytes.NewReader(content))
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to create upload request: %w", err)
	}

	uploadReq.Header.Set("Authorization", "Bearer "+d.accessToken)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	uploadReq.Header.Set("Dropbox-API-Arg", string(uploadArgJSON))

	uploadResp, err := d.client.Do(uploadReq)
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to upload to Dropbox: %w", err)
	}
	defer uploadResp.Body.Close()

	uploadBody, _ := io.ReadAll(uploadResp.Body)

	log.Debug().
		Int("status", uploadResp.StatusCode).
		Msg("Dropbox: Upload response received")

	if uploadResp.StatusCode != http.StatusOK {
		var errResp dropboxError
		json.Unmarshal(uploadBody, &errResp)
		log.Error().
			Int("status", uploadResp.StatusCode).
			Str("error", errResp.ErrorSummary).
			Str("body", string(uploadBody)).
			Msg("Dropbox upload failed")
		return ShareResult{}, fmt.Errorf("dropbox upload failed: %s", errResp.ErrorSummary)
	}

	var uploadResult dropboxUploadResponse
	if err := json.Unmarshal(uploadBody, &uploadResult); err != nil {
		return ShareResult{}, fmt.Errorf("failed to parse upload response: %w", err)
	}

	log.Debug().
		Str("file-id", uploadResult.ID).
		Str("path", uploadResult.Path).
		Msg("Dropbox: File uploaded, creating share link")

	// Create shared link
	shareArg := dropboxShareArg{
		Path: uploadResult.Path,
		Settings: dropboxShareSettings{
			RequestedVisibility: "public",
			Audience:            "public",
			Access:              "viewer",
		},
	}
	shareArgJSON, _ := json.Marshal(shareArg)

	shareReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.dropboxapi.com/2/sharing/create_shared_link_with_settings", bytes.NewReader(shareArgJSON))
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to create share request: %w", err)
	}

	shareReq.Header.Set("Authorization", "Bearer "+d.accessToken)
	shareReq.Header.Set("Content-Type", "application/json")

	shareResp, err := d.client.Do(shareReq)
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to create share link: %w", err)
	}
	defer shareResp.Body.Close()

	shareBody, _ := io.ReadAll(shareResp.Body)

	// Handle case where link already exists
	if shareResp.StatusCode == http.StatusConflict {
		// Link already exists, get existing link
		return d.getExistingLink(ctx, uploadResult.Path, expires)
	}

	if shareResp.StatusCode != http.StatusOK {
		var errResp dropboxError
		json.Unmarshal(shareBody, &errResp)
		log.Error().
			Int("status", shareResp.StatusCode).
			Str("error", errResp.ErrorSummary).
			Str("body", string(shareBody)).
			Msg("Dropbox share failed")
		return ShareResult{}, fmt.Errorf("dropbox share failed: %s", errResp.ErrorSummary)
	}

	var shareResult dropboxShareResponse
	if err := json.Unmarshal(shareBody, &shareResult); err != nil {
		return ShareResult{}, fmt.Errorf("failed to parse share response: %w", err)
	}

	// Convert to direct download link
	downloadURL := shareResult.URL
	if len(downloadURL) > 0 {
		// Replace dl=0 with dl=1 for direct download
		downloadURL = downloadURL[:len(downloadURL)-1] + "1"
	}

	log.Debug().
		Str("path", uploadResult.Path).
		Str("url", downloadURL).
		Int64("size", size).
		Msg("Uploaded file to Dropbox")

	return ShareResult{
		URL:       downloadURL,
		ExpiresAt: time.Now().Add(expires),
	}, nil
}

type dropboxListLinksArg struct {
	Path string `json:"path"`
}

type dropboxListLinksResponse struct {
	Links []dropboxShareResponse `json:"links"`
}

func (d *DropboxSharer) getExistingLink(ctx context.Context, path string, expires time.Duration) (ShareResult, error) {
	listArg := dropboxListLinksArg{Path: path}
	listArgJSON, _ := json.Marshal(listArg)

	listReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.dropboxapi.com/2/sharing/list_shared_links", bytes.NewReader(listArgJSON))
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to create list links request: %w", err)
	}

	listReq.Header.Set("Authorization", "Bearer "+d.accessToken)
	listReq.Header.Set("Content-Type", "application/json")

	listResp, err := d.client.Do(listReq)
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to list shared links: %w", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		return ShareResult{}, fmt.Errorf("failed to list shared links: status %d", listResp.StatusCode)
	}

	var listResult dropboxListLinksResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		return ShareResult{}, fmt.Errorf("failed to parse list links response: %w", err)
	}

	if len(listResult.Links) == 0 {
		return ShareResult{}, fmt.Errorf("no shared links found for path")
	}

	downloadURL := listResult.Links[0].URL
	if len(downloadURL) > 0 {
		downloadURL = downloadURL[:len(downloadURL)-1] + "1"
	}

	return ShareResult{
		URL:       downloadURL,
		ExpiresAt: time.Now().Add(expires),
	}, nil
}
