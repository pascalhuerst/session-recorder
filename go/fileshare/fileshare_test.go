/**
 * Test Plan: fileshare
 *
 * Scenario: Factory creates DirectS3Sharer by default
 *   Given FILE_SHARE_METHOD is not set or set to "direct"
 *   When NewFileSharer is called with a storage instance
 *   Then a DirectS3Sharer is returned
 *
 * Scenario: Factory creates FileIOSharer when configured
 *   Given FILE_SHARE_METHOD is set to "fileio"
 *   When NewFileSharer is called with a storage instance
 *   Then a FileIOSharer is returned
 *
 * Scenario: Factory returns error for unknown method
 *   Given FILE_SHARE_METHOD is set to an unknown value
 *   When NewFileSharer is called
 *   Then an error is returned
 *
 * Scenario: DirectS3Sharer delegates to storage
 *   Given a DirectS3Sharer with a mock storage
 *   When ShareSessionFile is called
 *   Then it calls storage.GetPresignedURL
 *   And returns the URL with correct expiry time
 */
package fileshare

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/storage"
)

// mockFileStorage implements FileStorage for testing
type mockFileStorage struct {
	presignedURL        string
	presignedURLErr     error
	segmentPresignedURL string
	segmentPresignedErr error
}

func (m *mockFileStorage) GetPresignedURL(ctx context.Context, asset storage.AssetOptions, options storage.SigningOptions) (string, error) {
	return m.presignedURL, m.presignedURLErr
}

func (m *mockFileStorage) GetSegmentPresignedURL(ctx context.Context, asset storage.SegmentAssetOptions, options storage.SigningOptions) (string, error) {
	return m.segmentPresignedURL, m.segmentPresignedErr
}

func (m *mockFileStorage) GetSessionFileReader(ctx context.Context, asset storage.AssetOptions) (io.ReadCloser, int64, error) {
	return nil, 0, nil
}

func (m *mockFileStorage) GetSegmentFileReader(ctx context.Context, asset storage.SegmentAssetOptions) (io.ReadCloser, int64, error) {
	return nil, 0, nil
}

func TestNewFileSharer_DefaultDirect(t *testing.T) {
	// Unset FILE_SHARE_METHOD to test default behavior
	os.Unsetenv("FILE_SHARE_METHOD")

	mock := &mockFileStorage{}
	sharer, err := NewFileSharer(mock)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := sharer.(*DirectS3Sharer); !ok {
		t.Errorf("expected DirectS3Sharer, got %T", sharer)
	}
}

func TestNewFileSharer_ExplicitDirect(t *testing.T) {
	os.Setenv("FILE_SHARE_METHOD", "direct")
	defer os.Unsetenv("FILE_SHARE_METHOD")

	mock := &mockFileStorage{}
	sharer, err := NewFileSharer(mock)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := sharer.(*DirectS3Sharer); !ok {
		t.Errorf("expected DirectS3Sharer, got %T", sharer)
	}
}

func TestNewFileSharer_FileIO(t *testing.T) {
	os.Setenv("FILE_SHARE_METHOD", "fileio")
	defer os.Unsetenv("FILE_SHARE_METHOD")

	mock := &mockFileStorage{}
	sharer, err := NewFileSharer(mock)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := sharer.(*FileIOSharer); !ok {
		t.Errorf("expected FileIOSharer, got %T", sharer)
	}
}

func TestNewFileSharer_UnknownMethod(t *testing.T) {
	os.Setenv("FILE_SHARE_METHOD", "unknown")
	defer os.Unsetenv("FILE_SHARE_METHOD")

	mock := &mockFileStorage{}
	_, err := NewFileSharer(mock)

	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestDirectS3Sharer_ShareSessionFile(t *testing.T) {
	expectedURL := "https://example.com/presigned-url"
	mock := &mockFileStorage{presignedURL: expectedURL}
	sharer := NewDirectS3Sharer(mock)

	ctx := context.Background()
	asset := storage.AssetOptions{
		RecorderID: uuid.New(),
		SessionID:  uuid.New(),
		Filename:   storage.FILENAME_FLAC,
	}
	options := storage.SigningOptions{
		Expires:  time.Hour * 24,
		Download: true,
	}

	result, err := sharer.ShareSessionFile(ctx, asset, options)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, result.URL)
	}

	// Check expiry is approximately correct (within 1 second)
	expectedExpiry := time.Now().Add(options.Expires)
	if result.ExpiresAt.Before(expectedExpiry.Add(-time.Second)) || result.ExpiresAt.After(expectedExpiry.Add(time.Second)) {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, result.ExpiresAt)
	}
}

func TestDirectS3Sharer_ShareSegmentFile(t *testing.T) {
	expectedURL := "https://example.com/segment-presigned-url"
	mock := &mockFileStorage{segmentPresignedURL: expectedURL}
	sharer := NewDirectS3Sharer(mock)

	ctx := context.Background()
	asset := storage.SegmentAssetOptions{
		RecorderID: uuid.New(),
		SessionID:  uuid.New(),
		SegmentID:  uuid.New(),
		Filename:   storage.SEGMENT_FILENAME_FLAC,
	}
	options := storage.SigningOptions{
		Expires:  time.Hour * 24,
		Download: true,
	}

	result, err := sharer.ShareSegmentFile(ctx, asset, options)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, result.URL)
	}
}
