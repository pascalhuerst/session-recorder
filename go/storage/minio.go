package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	bucketName = "session-recorder"
)

type Minio struct {
	endpoint  string
	accessKey string
	secretLey string

	client *minio.Client
}

func NewMinioStorage(endpoint, accessKey, secretKey string) (*Minio, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})

	exists, err := c.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot check if bucket exists: %w", err)
	}

	if !exists {
		err = c.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("cannot create bucket: %w", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("cannot connect to minio: %w", err)
	}

	return &Minio{
		endpoint:  endpoint,
		accessKey: accessKey,
		secretLey: secretKey,
		client:    c,
	}, nil
}

func (m *Minio) SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunkID string, samples []int16) error {
	objectName := fmt.Sprintf("%s/sessions/%s/chunks/%s", recorderID, sessionID, chunkID)

	buffer := new(bytes.Buffer)

	for _, chunk := range samples {
		binary.Write(buffer, binary.LittleEndian, chunk)
	}

	_, err := m.client.PutObject(ctx, bucketName, objectName, buffer, int64(buffer.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot put object: %w", err)
	}

	//log.Info().Msgf("Successfully uploaded (object=%s) %s/%s", objectName, sessionID, chunkID)

	return nil
}

func (m *Minio) CloseSession(ctx context.Context, sessionID uuid.UUID) error {
	return nil
}

func (m *Minio) GetRecorders(ctx context.Context) ([]uuid.UUID, error) {
	recorders := make([]uuid.UUID, 0)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: "", Recursive: false})

	for object := range objectCh {
		if object.Err != nil {
			log.Err(object.Err).Msg("Cannot list objects")

			return nil, object.Err
		}

		fmt.Printf("Recorder Object: %s\n", object.Key)

		idString, _ := strings.CutSuffix(object.Key, "/")
		recorderID, err := uuid.Parse(idString)
		if err != nil {
			log.Err(err).Str("recorder-id", object.Key).Msg("Cannot parse recorder ID")

			return nil, err
		}

		recorders = append(recorders, recorderID)
	}

	return recorders, nil
}

func (m *Minio) GetSessions(ctx context.Context, recorderID uuid.UUID) ([]uuid.UUID, error) {
	sessions := map[uuid.UUID]struct{}{}

	prefix := fmt.Sprintf("%s/sessions", recorderID.String())

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})

	for object := range objectCh {
		if object.Err != nil {
			log.Err(object.Err).Msg("Cannot list objects")

			return nil, object.Err
		}

		// 22a2a258-bc03-4836-a94a-d322d5e95b6a/sessions/3c48ad78-ddf7-4397-a3bd-1aaf769bb67d/chunks/0000000000000000
		tokens := strings.Split(object.Key, "/")
		idString := tokens[2]

		sessionID, err := uuid.Parse(idString)
		if err != nil {
			log.Err(err).Str("session-id", object.Key).Msg("Cannot parse session ID")

			return nil, err
		}

		sessions[sessionID] = struct{}{}
	}

	ret := make([]uuid.UUID, 0)
	for sessionID := range sessions {
		ret = append(ret, sessionID)
	}

	return ret, nil
}
