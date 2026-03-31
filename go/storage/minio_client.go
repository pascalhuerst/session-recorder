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
	ComposeObject(ctx context.Context, dst minio.CopyDestOptions, srcs ...minio.CopySrcOptions) (minio.UploadInfo, error)
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
}

// realMinioClient wraps *minio.Client to satisfy MinioClient.
// The only adaptation needed is GetObject: the real client returns
// *minio.Object while the interface returns ObjectHandle.
type realMinioClient struct {
	c *minio.Client
}

func newRealMinioClient(c *minio.Client) MinioClient {
	return &realMinioClient{c: c}
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

func (r *realMinioClient) ComposeObject(ctx context.Context, dst minio.CopyDestOptions, srcs ...minio.CopySrcOptions) (minio.UploadInfo, error) {
	return r.c.ComposeObject(ctx, dst, srcs...)
}

func (r *realMinioClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	return r.c.PresignedGetObject(ctx, bucketName, objectName, expires, reqParams)
}
