package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

// fakeObject implements ObjectHandle backed by an in-memory byte slice.
type fakeObject struct {
	*bytes.Reader
	info minio.ObjectInfo
}

func newFakeObject(key string, data []byte) *fakeObject {
	return &fakeObject{
		Reader: bytes.NewReader(data),
		info:   minio.ObjectInfo{Key: key, Size: int64(len(data))},
	}
}

func (f *fakeObject) Close() error                          { return nil }
func (f *fakeObject) Stat() (minio.ObjectInfo, error)       { return f.info, nil }

// FakeMinioClient is an in-memory implementation of MinioClient for testing.
// It stores objects in maps keyed by "bucket/object". Thread-safe.
type FakeMinioClient struct {
	mu      sync.Mutex
	buckets map[string]bool
	objects map[string][]byte // key: "bucket/object"
}

func NewFakeMinioClient() *FakeMinioClient {
	return &FakeMinioClient{
		buckets: make(map[string]bool),
		objects: make(map[string][]byte),
	}
}

func (f *FakeMinioClient) key(bucket, object string) string {
	return bucket + "/" + object
}

func (f *FakeMinioClient) MakeBucket(_ context.Context, bucketName string, _ minio.MakeBucketOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.buckets[bucketName] {
		return fmt.Errorf("bucket %q already exists", bucketName)
	}
	f.buckets[bucketName] = true
	return nil
}

func (f *FakeMinioClient) BucketExists(_ context.Context, bucketName string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buckets[bucketName], nil
}

func (f *FakeMinioClient) ListObjects(_ context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	ch := make(chan minio.ObjectInfo)
	go func() {
		defer close(ch)
		f.mu.Lock()
		prefix := opts.Prefix
		var keys []string
		for k := range f.objects {
			fullPrefix := bucketName + "/"
			if strings.HasPrefix(k, fullPrefix+prefix) {
				objectKey := strings.TrimPrefix(k, fullPrefix)

				// Non-recursive: only return items at the current prefix level
				if !opts.Recursive {
					remaining := strings.TrimPrefix(objectKey, prefix)
					if idx := strings.Index(remaining, "/"); idx >= 0 {
						// This is a "directory" — return prefix entry
						dirKey := prefix + remaining[:idx+1]
						// Deduplicate directories
						found := false
						for _, existing := range keys {
							if existing == dirKey {
								found = true
								break
							}
						}
						if !found {
							keys = append(keys, dirKey)
						}
						continue
					}
				}

				keys = append(keys, objectKey)
			}
		}
		f.mu.Unlock()

		sort.Strings(keys)
		for _, key := range keys {
			f.mu.Lock()
			data, exists := f.objects[bucketName+"/"+key]
			f.mu.Unlock()
			info := minio.ObjectInfo{Key: key}
			if exists {
				info.Size = int64(len(data))
			}
			ch <- info
		}
	}()
	return ch
}

func (f *FakeMinioClient) GetObject(_ context.Context, bucketName, objectName string, _ minio.GetObjectOptions) (ObjectHandle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[f.key(bucketName, objectName)]
	if !ok {
		return nil, fmt.Errorf("object %q not found in bucket %q", objectName, bucketName)
	}
	// Return a copy so concurrent reads don't interfere
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	return newFakeObject(objectName, dataCopy), nil
}

func (f *FakeMinioClient) PutObject(_ context.Context, bucketName, objectName string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("cannot read data: %w", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[f.key(bucketName, objectName)] = data
	return minio.UploadInfo{Bucket: bucketName, Key: objectName, Size: int64(len(data))}, nil
}

func (f *FakeMinioClient) RemoveObject(_ context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := f.key(bucketName, objectName)
	if _, ok := f.objects[k]; ok {
		delete(f.objects, k)
		return nil
	}

	// If ForceDelete, also remove matching prefix (directory-like delete)
	if opts.ForceDelete {
		prefix := k
		for key := range f.objects {
			if strings.HasPrefix(key, prefix) {
				delete(f.objects, key)
			}
		}
	}

	return nil
}

func (f *FakeMinioClient) StatObject(_ context.Context, bucketName, objectName string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[f.key(bucketName, objectName)]
	if !ok {
		return minio.ObjectInfo{}, fmt.Errorf("object %q not found in bucket %q", objectName, bucketName)
	}
	return minio.ObjectInfo{
		Key:  objectName,
		Size: int64(len(data)),
	}, nil
}

func (f *FakeMinioClient) ComposeObject(_ context.Context, dst minio.CopyDestOptions, srcs ...minio.CopySrcOptions) (minio.UploadInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var combined []byte
	for _, src := range srcs {
		data, ok := f.objects[f.key(src.Bucket, src.Object)]
		if !ok {
			return minio.UploadInfo{}, fmt.Errorf("source object %q not found", src.Object)
		}
		combined = append(combined, data...)
	}

	f.objects[f.key(dst.Bucket, dst.Object)] = combined
	return minio.UploadInfo{Bucket: dst.Bucket, Key: dst.Object, Size: int64(len(combined))}, nil
}

func (f *FakeMinioClient) PresignedGetObject(_ context.Context, bucketName, objectName string, _ time.Duration, reqParams url.Values) (*url.URL, error) {
	// Return a deterministic fake URL
	u := &url.URL{
		Scheme:   "http",
		Host:     "fake-minio:9000",
		Path:     fmt.Sprintf("/%s/%s", bucketName, objectName),
		RawQuery: reqParams.Encode(),
	}
	return u, nil
}

// ObjectCount returns the number of objects stored (for test assertions).
func (f *FakeMinioClient) ObjectCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.objects)
}

// ObjectExists checks if a specific object exists (for test assertions).
func (f *FakeMinioClient) ObjectExists(bucket, object string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[f.key(bucket, object)]
	return ok
}

// GetObjectData returns the raw bytes for an object (for test assertions).
func (f *FakeMinioClient) GetObjectData(bucket, object string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[f.key(bucket, object)]
	return data, ok
}
