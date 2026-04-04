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

func (f *fakeObject) Close() error                    { return nil }
func (f *fakeObject) Stat() (minio.ObjectInfo, error) { return f.info, nil }

// fakeMultipartUpload tracks an in-progress multipart upload.
type fakeMultipartUpload struct {
	object string
	parts  map[int][]byte // partNumber -> data
}

// FakeMinioClient is an in-memory implementation of MinioClient for testing.
// It stores objects in maps keyed by "bucket/object". Thread-safe.
type FakeMinioClient struct {
	mu           sync.Mutex
	buckets      map[string]bool
	objects      map[string][]byte              // key: "bucket/object"
	uploads      map[string]*fakeMultipartUpload // uploadID -> upload
	nextUploadID int
}

func NewFakeMinioClient() *FakeMinioClient {
	return &FakeMinioClient{
		buckets: make(map[string]bool),
		objects: make(map[string][]byte),
		uploads: make(map[string]*fakeMultipartUpload),
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

// Multipart upload operations

func (f *FakeMinioClient) NewMultipartUpload(_ context.Context, _ string, object string, _ minio.PutObjectOptions) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextUploadID++
	uploadID := fmt.Sprintf("upload-%d", f.nextUploadID)
	f.uploads[uploadID] = &fakeMultipartUpload{
		object: object,
		parts:  make(map[int][]byte),
	}
	return uploadID, nil
}

func (f *FakeMinioClient) PutObjectPart(_ context.Context, _, _, uploadID string, partID int, data io.Reader, _ int64, _ minio.PutObjectPartOptions) (minio.ObjectPart, error) {
	partData, err := io.ReadAll(data)
	if err != nil {
		return minio.ObjectPart{}, fmt.Errorf("cannot read part data: %w", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	upload, ok := f.uploads[uploadID]
	if !ok {
		return minio.ObjectPart{}, fmt.Errorf("upload %q not found", uploadID)
	}
	upload.parts[partID] = partData
	return minio.ObjectPart{
		PartNumber: partID,
		ETag:       fmt.Sprintf("etag-%s-%d", uploadID, partID),
		Size:       int64(len(partData)),
	}, nil
}

func (f *FakeMinioClient) CompleteMultipartUpload(_ context.Context, bucketName, object, uploadID string, parts []minio.CompletePart, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	upload, ok := f.uploads[uploadID]
	if !ok {
		return minio.UploadInfo{}, fmt.Errorf("upload %q not found", uploadID)
	}

	// Sort parts by part number and concatenate
	sortedParts := make([]minio.CompletePart, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool {
		return sortedParts[i].PartNumber < sortedParts[j].PartNumber
	})

	var combined []byte
	for _, p := range sortedParts {
		data, exists := upload.parts[p.PartNumber]
		if !exists {
			return minio.UploadInfo{}, fmt.Errorf("part %d not found in upload %q", p.PartNumber, uploadID)
		}
		combined = append(combined, data...)
	}

	f.objects[f.key(bucketName, object)] = combined
	delete(f.uploads, uploadID)
	return minio.UploadInfo{Bucket: bucketName, Key: object, Size: int64(len(combined))}, nil
}

func (f *FakeMinioClient) AbortMultipartUpload(_ context.Context, _, _, uploadID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.uploads, uploadID)
	return nil
}

func (f *FakeMinioClient) ListObjectParts(_ context.Context, _, _, uploadID string, partNumberMarker, maxParts int) (minio.ListObjectPartsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	upload, ok := f.uploads[uploadID]
	if !ok {
		return minio.ListObjectPartsResult{}, fmt.Errorf("upload %q not found", uploadID)
	}

	// Collect and sort part numbers
	var partNumbers []int
	for pn := range upload.parts {
		if pn > partNumberMarker {
			partNumbers = append(partNumbers, pn)
		}
	}
	sort.Ints(partNumbers)

	truncated := false
	if maxParts > 0 && len(partNumbers) > maxParts {
		partNumbers = partNumbers[:maxParts]
		truncated = true
	}

	var objectParts []minio.ObjectPart
	for _, pn := range partNumbers {
		objectParts = append(objectParts, minio.ObjectPart{
			PartNumber: pn,
			ETag:       fmt.Sprintf("etag-%s-%d", uploadID, pn),
			Size:       int64(len(upload.parts[pn])),
		})
	}

	nextMarker := 0
	if len(partNumbers) > 0 {
		nextMarker = partNumbers[len(partNumbers)-1]
	}

	return minio.ListObjectPartsResult{
		UploadID:             uploadID,
		ObjectParts:          objectParts,
		IsTruncated:          truncated,
		NextPartNumberMarker: nextMarker,
	}, nil
}

func (f *FakeMinioClient) ListMultipartUploads(_ context.Context, bucketName, prefix, _, _, _ string, maxUploads int) (minio.ListMultipartUploadsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var uploads []minio.ObjectMultipartInfo
	for uploadID, upload := range f.uploads {
		if strings.HasPrefix(upload.object, prefix) {
			uploads = append(uploads, minio.ObjectMultipartInfo{
				Key:      upload.object,
				UploadID: uploadID,
			})
			if maxUploads > 0 && len(uploads) >= maxUploads {
				break
			}
		}
	}

	return minio.ListMultipartUploadsResult{
		Bucket:  bucketName,
		Uploads: uploads,
	}, nil
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

// RemoveUpload deletes a multipart upload by ID (for sabotaging uploads in tests).
func (f *FakeMinioClient) RemoveUpload(uploadID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.uploads, uploadID)
}

// MultipartUploadCount returns the number of in-progress multipart uploads (for test assertions).
func (f *FakeMinioClient) MultipartUploadCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.uploads)
}

// MultipartPartCount returns the number of parts in a given upload (for test assertions).
func (f *FakeMinioClient) MultipartPartCount(uploadID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	upload, ok := f.uploads[uploadID]
	if !ok {
		return 0
	}
	return len(upload.parts)
}
