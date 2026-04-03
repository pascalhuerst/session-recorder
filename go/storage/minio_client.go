package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// ObjectHandle abstracts an S3 object that can be read, closed, and stat'd.
// *minio.Object satisfies this interface.
type ObjectHandle interface {
	io.ReadCloser
	Stat() (minio.ObjectInfo, error)
}

// MinioClient abstracts the subset of *minio.Client methods used by the
// storage layer. In production the realMinioClient adapter wraps *minio.Client
// to satisfy this. In tests, an in-memory fake can be substituted.
type MinioClient interface {
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (ObjectHandle, error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)

	// Multipart upload operations
	NewMultipartUpload(ctx context.Context, bucket, object string, opts minio.PutObjectOptions) (string, error)
	PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data io.Reader, size int64, opts minio.PutObjectPartOptions) (minio.ObjectPart, error)
	CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, parts []minio.CompletePart, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error
	ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker, maxParts int) (minio.ListObjectPartsResult, error)
	ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (minio.ListMultipartUploadsResult, error)
}

// realMinioClient wraps *minio.Client to satisfy MinioClient.
// The only adaptation needed is GetObject: the real client returns
// *minio.Object while the interface returns ObjectHandle.
type realMinioClient struct {
	c    *minio.Client
	core *minio.Core
}

func newRealMinioClient(c *minio.Client) MinioClient {
	return &realMinioClient{
		c:    c,
		core: &minio.Core{Client: c},
	}
}

func (r *realMinioClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	return r.c.MakeBucket(ctx, bucketName, opts)
}

func (r *realMinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return r.c.BucketExists(ctx, bucketName)
}

func (r *realMinioClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return r.c.ListObjects(ctx, bucketName, opts)
}

func (r *realMinioClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (ObjectHandle, error) {
	return r.c.GetObject(ctx, bucketName, objectName, opts)
}

func (r *realMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return r.c.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}

func (r *realMinioClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	return r.c.RemoveObject(ctx, bucketName, objectName, opts)
}

func (r *realMinioClient) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	return r.c.StatObject(ctx, bucketName, objectName, opts)
}

func (r *realMinioClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	return r.c.PresignedGetObject(ctx, bucketName, objectName, expires, reqParams)
}

// Multipart upload operations — delegated to minio.Core

func (r *realMinioClient) NewMultipartUpload(ctx context.Context, bucket, object string, opts minio.PutObjectOptions) (string, error) {
	return r.core.NewMultipartUpload(ctx, bucket, object, opts)
}

func (r *realMinioClient) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data io.Reader, size int64, opts minio.PutObjectPartOptions) (minio.ObjectPart, error) {
	return r.core.PutObjectPart(ctx, bucket, object, uploadID, partID, data, size, opts)
}

func (r *realMinioClient) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, parts []minio.CompletePart, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return r.core.CompleteMultipartUpload(ctx, bucket, object, uploadID, parts, opts)
}

func (r *realMinioClient) AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error {
	return r.core.AbortMultipartUpload(ctx, bucket, object, uploadID)
}

func (r *realMinioClient) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker, maxParts int) (minio.ListObjectPartsResult, error) {
	return r.core.ListObjectParts(ctx, bucket, object, uploadID, partNumberMarker, maxParts)
}

func (r *realMinioClient) ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (minio.ListMultipartUploadsResult, error) {
	return r.core.ListMultipartUploads(ctx, bucket, prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
}
