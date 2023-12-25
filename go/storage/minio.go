package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/model"
	"github.com/pascalhuerst/session-recorder/render"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

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

func (m *Minio) IsSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool {
	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks/", recorderID, sessionID)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: false})

	for range objectCh {
		return true
	}

	return false
}

func (m *Minio) CloseSession(ctx context.Context, recorderID, sessionID uuid.UUID, workDir string) error {
	err := os.MkdirAll(workDir, os.ModePerm)
	if err != nil {
		log.Err(err).Msg("Cannot create work dir")

		return err
	}

	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks", recorderID, sessionID)

	if !m.IsSessionClosed(ctx, recorderID, sessionID) {
		log.Debug().Str("object", chunksPrefix).Msg("Already closed")

		return nil
	}

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})

	toBeDeleted := make([]string, 0)

	file, err := os.Create(path.Join(workDir, "data.raw"))
	if err != nil {
		log.
			Err(err).Msg("Cannot create file")
	}
	defer file.Close()

	for objectInfo := range objectCh {
		if objectInfo.Err != nil {
			log.Err(objectInfo.Err).Msg("Cannot list objects")

			return objectInfo.Err
		}

		object, err := m.client.GetObject(ctx, bucketName, objectInfo.Key, minio.GetObjectOptions{})
		if err != nil {
			log.Err(err).Msg("Cannot get object")
		}

		_, err = io.Copy(file, object)
		if err != nil {
			log.Err(err).Msg("Cannot copy object")
		}

		toBeDeleted = append(toBeDeleted, objectInfo.Key)
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return render.CreateAudioFile(path.Join(workDir, "data.raw"), workDir, "flac")
	})

	eg.Go(func() error {
		return render.CreateAudioFile(path.Join(workDir, "data.raw"), workDir, "mp3")
	})

	eg.Go(func() error {
		return render.CreateAudioFile(path.Join(workDir, "data.raw"), workDir, "ogg")
	})

	if err := eg.Wait(); err != nil {
		log.Err(err).Msg("Cannot create audio files")

		return err
	}

	eg, _ = errgroup.WithContext(ctx)

	eg.Go(func() error {
		return render.CreateWaveform(path.Join(workDir, "data.flac"), path.Join(workDir, "waveform.dat"), 300, 10000, 200)
	})

	eg.Go(func() error {
		return render.CreateWaveform(path.Join(workDir, "data.flac"), path.Join(workDir, "overview.png"), 300, 1000, 200)
	})

	eg.Go(func() error {
		return render.CreateWaveform(path.Join(workDir, "data.flac"), path.Join(workDir, "full.png"), 300, 10000, 200)
	})

	if err := eg.Wait(); err != nil {
		log.Err(err).Msg("Cannot create waveform files")

		return err
	}

	for _, f := range []string{"data.flac", "data.mp3", "data.ogg", "data.raw", "full.png", "overview.png", "waveform.dat"} {
		objectName := fmt.Sprintf("%s/sessions/%s/%s", recorderID, sessionID, f)
		file := path.Join(workDir, f)

		i, err := m.client.FPutObject(ctx, bucketName, objectName, file, minio.PutObjectOptions{})
		if err != nil {
			log.Err(err).Str("object", f).Msg("Cannot put object")

			continue
		}

		log.Info().Str("object", f).Int64("size", i.Size).Msg("Successfully uploaded")
	}

	os.RemoveAll(workDir)

	for _, objectName := range toBeDeleted {
		if err := m.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
			log.Err(err).Str("object", objectName).Msg("Cannot remove object")
		}
	}

	if err = m.client.RemoveObject(ctx, bucketName, chunksPrefix, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		log.Err(err).Str("object", chunksPrefix).Msg("Cannot remove object")
	}

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
		err = m.CloseSession(ctx, recorderID, sessionID, os.TempDir())
		if err != nil {
			return fmt.Errorf("cannot close session: %w", err)
		}
	}

	return nil
}
