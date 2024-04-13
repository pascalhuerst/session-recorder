package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/model"
	"github.com/pascalhuerst/session-recorder/render"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	bucketName   = "session-recorder"
	minChunkSize = 5 * 1024 * 1024 // As per s3 documentation
)

const publicAccessFormula = `
{
	"Version": "2012-10-17",
	"Statement": [
	  {
		"Effect": "Allow",
		"Principal": {
		  "AWS": [
			"*"
		  ]
		},
		"Action": [
		  "s3:GetBucketLocation",
		  "s3:ListBucket",
		  "s3:ListBucketMultipartUploads"
		],
		"Resource": [
		  "arn:aws:s3:::%s"
		]
	  },
	  {
		"Effect": "Allow",
		"Principal": {
		  "AWS": [
			"*"
		  ]
		},
		"Action": [
		  "s3:AbortMultipartUpload",
		  "s3:DeleteObject",
		  "s3:GetObject",
		  "s3:ListMultipartUploadParts",
		  "s3:PutObject"
		],
		"Resource": [
		  "arn:aws:s3:::%s/*"
		]
	  }
	]
  }
`

type minioChunk struct {
	number    int
	sessionID uuid.UUID
	buffer    *bytes.Buffer
}

type Minio struct {
	endpoint  string
	accessKey string
	secretLey string

	client *minio.Client

	// Key is recorder ID
	chunks     map[uuid.UUID]*minioChunk
	chunksLock sync.Mutex
}

func NewMinioStorage(endpoint, accessKey, secretKey string) (*Minio, error) {
	ctx := context.Background()

	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create minio client: %w", err)
	}

	exists, err := c.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot check if bucket exists: %w", err)
	}

	if !exists {
		err = c.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("cannot create bucket: %w", err)
		}

		policy := fmt.Sprintf(publicAccessFormula, bucketName, bucketName)
		if err := c.SetBucketPolicy(ctx, bucketName, policy); err != nil {
			log.Error().Err(err).Str("bucket", bucketName).Msg("Cannot set bucket policy")
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
		chunks:    make(map[uuid.UUID]*minioChunk),
	}, nil
}

func (m *Minio) initSession(recorderID, sessionID uuid.UUID) {
	log.Info().Str("recorder-id", recorderID.String()).Str("session-id", sessionID.String()).Msg("Starting new session")

	m.chunks[recorderID] = &minioChunk{
		number:    0,
		sessionID: sessionID,
		buffer:    new(bytes.Buffer),
	}
}

func (m *Minio) SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, _ string, samples []int16) error {
	if _, ok := m.chunks[recorderID]; !ok {
		m.initSession(recorderID, sessionID)
	}

	chunk := m.chunks[recorderID]
	if chunk.sessionID != sessionID {
		if err := m.CloseSession(ctx, recorderID, chunk.sessionID); err != nil {
			log.Fatal().Err(err).Msg("Cannot close session")
		}

		go func(recorderID, sessionID uuid.UUID) {
			if err := m.renderClosedSession(recorderID, chunk.sessionID); err != nil {
				log.Err(err).Msg("Cannot render session")
			}
		}(recorderID, chunk.sessionID)

		m.initSession(recorderID, sessionID)
	}

	binary.Write(chunk.buffer, binary.LittleEndian, samples)

	log.Info().Str("recorder-id", recorderID.String()).Str("session-id", sessionID.String()).Msgf("Added %d|%d (%.1f %%) samples", chunk.buffer.Len(), minChunkSize, float64(chunk.buffer.Len())/float64(minChunkSize)*100.0)

	// To be able to concatinate the chunks, we need to make sure that the chunk size is at least 5MB
	if chunk.buffer.Len() >= minChunkSize {
		log.Debug().Str("recorder-id", recorderID.String()).Str("session-id", sessionID.String()).Msg("Chunk is full")

		objectName := fmt.Sprintf("%s/sessions/%s/chunks/%s", recorderID, sessionID, fmt.Sprintf("%016d", chunk.number))
		chunk.number++

		_, err := m.client.PutObject(ctx, bucketName, objectName, chunk.buffer, int64(chunk.buffer.Len()), minio.PutObjectOptions{})
		if err != nil {
			return fmt.Errorf("cannot put object: %w", err)
		}
	}

	return nil
}

func (m *Minio) IsSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool {
	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks/", recorderID, sessionID)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: false})

	for range objectCh {
		return true
	}

	return false
}

func (m *Minio) CloseSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	// If we have samples left for this session, let's push those first
	if chunk, ok := m.chunks[recorderID]; ok {
		objectName := fmt.Sprintf("%s/sessions/%s/chunks/%s", recorderID, sessionID, fmt.Sprintf("%016d", chunk.number))

		if chunk.buffer.Len() > 0 {
			if chunk.buffer.Len() < minChunkSize {
				// If the last chunk is smaller than 5MB, we need to pad it with zeros
				chunk.buffer.Write(make([]byte, minChunkSize-chunk.buffer.Len()))
			}

			_, err := m.client.PutObject(ctx, bucketName, objectName, chunk.buffer, int64(chunk.buffer.Len()), minio.PutObjectOptions{})
			if err != nil {
				return fmt.Errorf("cannot put object: %w", err)
			}
		}
	}

	delete(m.chunks, recorderID)

	log.Info().Str("recorder-id", recorderID.String()).Str("session-id", sessionID.String()).Msg("Session closed")

	return nil
}

func (m *Minio) renderClosedSession(recorderID, sessionID uuid.UUID) error {
	ctx := context.Background()

	// Do the ComposeObject dance
	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks", recorderID, sessionID)

	if !m.IsSessionClosed(ctx, recorderID, sessionID) {
		return nil
	}

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})
	srcs := make([]minio.CopySrcOptions, 0)

	for objectInfo := range objectCh {
		if objectInfo.Err != nil {
			log.Err(objectInfo.Err).Msg("Cannot list objects")

			return objectInfo.Err
		}

		srcs = append(srcs, minio.CopySrcOptions{
			Bucket: bucketName,
			Object: objectInfo.Key,
		})
	}

	dst := minio.CopyDestOptions{
		Bucket: bucketName,
		Object: fmt.Sprintf("%s/sessions/%s/data.raw", recorderID, sessionID),
	}

	_, err := m.client.ComposeObject(ctx, dst, srcs...)
	if err != nil {
		log.Err(err).Msg("Cannot compose object")

		return err
	}

	var rawData *minio.Object
	if rawData, err = m.client.GetObject(ctx, bucketName, dst.Object, minio.GetObjectOptions{}); err != nil {
		log.Err(err).Str("object", dst.Object).Msg("Cannot get object")

		return err
	}

	readers, writer, closer := makeReaders(4)
	eg, _ := errgroup.WithContext(ctx)

	// Writer copies from raw data to all the readers
	eg.Go(func() error {
		defer closer.Close()

		_, err := io.Copy(writer, rawData)
		if err != nil {
			log.Err(err).Msg("Cannot setup multiple readers")

			return err
		}

		return nil
	})

	// Create waveform dat file. Reads from reader 2
	eg.Go(func() error {
		waveformData, err := render.CreateWaveform(readers[0], 300, 10000, 200)
		if err != nil {
			return fmt.Errorf("cannot create waveform: %w", err)
		}

		waveformObject := fmt.Sprintf("%s/sessions/%s/waveform.dat", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, waveformObject, waveformData, int64(waveformData.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", waveformObject).Msg("Cannot put object")

			return err
		}

		return nil
	})

	// Create waveform png overview file. Reads from reader 1
	eg.Go(func() error {
		overviewData, err := render.CreateOverview(readers[1], 300, 1000, 200)
		if err != nil {
			return fmt.Errorf("cannot create waveform overview: %w", err)
		}

		overviewObject := fmt.Sprintf("%s/sessions/%s/overview.png", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, overviewObject, overviewData, int64(overviewData.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", overviewObject).Msg("Cannot put object")

			return err
		}

		return nil
	})

	// Create flac file. Reads from reader 2
	eg.Go(func() error {
		var flacBuffer *bytes.Buffer
		if flacBuffer, err = render.Flac(readers[2]); err != nil {
			log.Err(err).Msg("Cannot convert to flac")

			return err
		}

		flacObject := fmt.Sprintf("%s/sessions/%s/data.flac", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, flacObject, flacBuffer, int64(flacBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", flacObject).Msg("Cannot put object")

			return err
		}

		return nil
	})

	// Create ogg file. Reads from reader 3
	eg.Go(func() error {
		ext := "ogg"

		var buffer *bytes.Buffer
		if buffer, err = render.CreateAudioFile(readers[3], ext); err != nil {
			log.Err(err).Msg("Cannot convert to ogg")

			return err
		}

		object := fmt.Sprintf("%s/sessions/%s/data.%s", recorderID, sessionID, ext)
		if _, err := m.client.PutObject(ctx, bucketName, object, buffer, int64(buffer.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", object).Msg("Cannot put object")

			return err
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Err(err).Msg("An error occured while rendering")

		return err
	}

	if err = m.client.RemoveObject(ctx, bucketName, chunksPrefix, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		log.Err(err).Str("object", chunksPrefix).Msg("Cannot remove object")
	}

	log.Info().Str("recorder-id", recorderID.String()).Str("session-id", sessionID.String()).Msg("Session rendered")

	return nil
}

func (m *Minio) GetRecorderIDs(ctx context.Context) ([]uuid.UUID, error) {
	recorders := make([]uuid.UUID, 0)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: "", Recursive: false})

	for object := range objectCh {
		if object.Err != nil {
			log.Err(object.Err).Msg("Cannot list objects")

			return nil, object.Err
		}

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

func (m *Minio) GetSessionIDs(ctx context.Context, recorderID uuid.UUID) ([]uuid.UUID, error) {
	sessions := map[uuid.UUID]struct{}{}

	prefix := fmt.Sprintf("%s/sessions", recorderID.String())

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})

	for object := range objectCh {
		if object.Err != nil {
			log.Err(object.Err).Msg("Cannot list objects")

			return nil, object.Err
		}

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

func (m *Minio) GetSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID) (*model.SessionMetadata, error) {
	objectName := fmt.Sprintf("%s/sessions/%s/metadata.json", recorderID, sessionID)

	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get object: %w", err)
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(obj)

	if err != nil {
		return nil, fmt.Errorf("cannot read object: %w", err)
	}

	metadata := &model.SessionMetadata{}
	err = json.Unmarshal(buffer.Bytes(), metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	return metadata, nil
}

func (m *Minio) PutSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID, session *model.SessionMetadata) error {
	objectName := fmt.Sprintf("%s/sessions/%s/metadata.json", recorderID, sessionID)

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(session)
	if err != nil {
		return fmt.Errorf("cannot marshal metadata: %w", err)
	}

	_, err = m.client.PutObject(ctx, bucketName, objectName, buffer, int64(buffer.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot put object: %w", err)
	}

	return nil
}

func (m *Minio) PutRecorderMetadata(ctx context.Context, recorderID uuid.UUID, recorder *model.RecorderMetadata) error {
	onjectName := fmt.Sprintf("%s/metadata.json", recorderID)

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(recorder)
	if err != nil {
		return fmt.Errorf("cannot marshal metadata: %w", err)
	}

	_, err = m.client.PutObject(ctx, bucketName, onjectName, buffer, int64(buffer.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot put object: %w", err)
	}

	return nil
}

func (m *Minio) GetRecorderMetadata(ctx context.Context, recorderID uuid.UUID) (*model.RecorderMetadata, error) {
	objectName := fmt.Sprintf("%s/metadata.json", recorderID)

	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get object: %w", err)
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(obj)

	if err != nil {
		return nil, fmt.Errorf("cannot read object: %w", err)
	}

	metadata := &model.RecorderMetadata{}
	err = json.Unmarshal(buffer.Bytes(), metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	return metadata, nil
}

func (m *Minio) CloseOpenSessions(ctx context.Context, recorderID uuid.UUID) error {
	sessionIDs, err := m.GetSessionIDs(ctx, recorderID)
	if err != nil {
		return fmt.Errorf("cannot get session IDs: %w", err)
	}

	for _, sessionID := range sessionIDs {
		err = m.CloseSession(ctx, recorderID, sessionID)
		if err != nil {
			return fmt.Errorf("cannot close session: %w", err)
		}

		// Render the session assynchonously
		go func(recorderID, sessionID uuid.UUID) {
			if err := m.renderClosedSession(recorderID, sessionID); err != nil {
				log.Err(err).Msg("Cannot render session")
			}
		}(recorderID, sessionID)
	}

	return nil
}
