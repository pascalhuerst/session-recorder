package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/minio/minio-go/v7"
)

/**
 * Test Plan: makeSureBucketExists error handling
 *
 * Scenario: MakeBucket succeeds on first call
 *   Given a fresh MinIO client with no existing buckets
 *   When makeSureBucketExists is called
 *   Then no error is returned
 *
 * Scenario: MakeBucket fails but bucket already exists
 *   Given MakeBucket returns an error
 *   And BucketExists returns true
 *   When makeSureBucketExists is called
 *   Then no error is returned (bucket already exists)
 *
 * Scenario: MakeBucket fails and bucket does not exist
 *   Given MakeBucket returns an error
 *   And BucketExists returns false
 *   When makeSureBucketExists is called
 *   Then the original MakeBucket error is wrapped and returned
 *
 * Scenario: MakeBucket fails and BucketExists also fails
 *   Given MakeBucket returns an error
 *   And BucketExists returns an error
 *   When makeSureBucketExists is called
 *   Then the BucketExists error is wrapped and returned
 */

// bucketStubClient is a minimal MinioClient stub that lets tests control
// MakeBucket and BucketExists outcomes independently.
type bucketStubClient struct {
	FakeMinioClient
	makeBucketErr   error
	bucketExistsVal bool
	bucketExistsErr error
}

func (b *bucketStubClient) MakeBucket(_ context.Context, _ string, _ minio.MakeBucketOptions) error {
	return b.makeBucketErr
}

func (b *bucketStubClient) BucketExists(_ context.Context, _ string) (bool, error) {
	return b.bucketExistsVal, b.bucketExistsErr
}

func TestMakeSureBucketExists_Success(t *testing.T) {
	stub := &bucketStubClient{
		FakeMinioClient: *NewFakeMinioClient(),
		makeBucketErr:   nil,
	}
	m := NewMinioStorageWithClient(stub, "fake:9000", "fake:9000", "fake:9000")

	if err := m.makeSureBucketExists(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestMakeSureBucketExists_MakeFails_BucketExists(t *testing.T) {
	stub := &bucketStubClient{
		FakeMinioClient: *NewFakeMinioClient(),
		makeBucketErr:   fmt.Errorf("bucket already owned"),
		bucketExistsVal: true,
		bucketExistsErr: nil,
	}
	m := NewMinioStorageWithClient(stub, "fake:9000", "fake:9000", "fake:9000")

	if err := m.makeSureBucketExists(context.Background()); err != nil {
		t.Errorf("expected no error when bucket exists, got %v", err)
	}
}

func TestMakeSureBucketExists_MakeFails_BucketNotExists(t *testing.T) {
	stub := &bucketStubClient{
		FakeMinioClient: *NewFakeMinioClient(),
		makeBucketErr:   fmt.Errorf("permission denied"),
		bucketExistsVal: false,
		bucketExistsErr: nil,
	}
	m := NewMinioStorageWithClient(stub, "fake:9000", "fake:9000", "fake:9000")

	err := m.makeSureBucketExists(context.Background())
	if err == nil {
		t.Fatal("expected error when bucket doesn't exist, got nil")
	}
	if want := "cannot create bucket"; !contains(err.Error(), want) {
		t.Errorf("error should contain %q, got %q", want, err.Error())
	}
	if !contains(err.Error(), "permission denied") {
		t.Errorf("error should wrap original MakeBucket error, got %q", err.Error())
	}
}

func TestMakeSureBucketExists_MakeFails_BucketExistsFails(t *testing.T) {
	stub := &bucketStubClient{
		FakeMinioClient: *NewFakeMinioClient(),
		makeBucketErr:   fmt.Errorf("permission denied"),
		bucketExistsVal: false,
		bucketExistsErr: fmt.Errorf("network timeout"),
	}
	m := NewMinioStorageWithClient(stub, "fake:9000", "fake:9000", "fake:9000")

	err := m.makeSureBucketExists(context.Background())
	if err == nil {
		t.Fatal("expected error when BucketExists fails, got nil")
	}
	if want := "cannot check if bucket exists"; !contains(err.Error(), want) {
		t.Errorf("error should contain %q, got %q", want, err.Error())
	}
	if !contains(err.Error(), "network timeout") {
		t.Errorf("error should wrap BucketExists error, got %q", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
