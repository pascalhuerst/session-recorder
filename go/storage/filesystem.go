package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/model"
	"github.com/rs/zerolog/log"
)

var (
	errSetupFailed = errors.New("cannot setup filesystem storage")
	errEnsureDir   = errors.New("cannot ensure directory exists")
)

const (
	sessionsDir   = "sessions"
	chunksDir     = "chunks"
	recordingsDir = "recordings"
)

type filesystemChunk struct {
	number    int
	sessionID uuid.UUID
}

type Filesystem struct {
	basepath string

	chunks     map[uuid.UUID]*filesystemChunk
	chunksLock sync.Mutex
}

//session-recorder/2bb00a6a-c468-41b0-b8b8-40e3cd22450e/sessions/1dd208c5-300e-40fb-81d6-0a8b5d962c6b/chunks

func NewFilesystem(basepath string) (*Filesystem, error) {
	if err := ensureDirExists(basepath); err != nil {
		return nil, fmt.Errorf("%w: %w", errSetupFailed, err)
	}

	return &Filesystem{basepath: basepath}, nil
}

func dirExists(path string) bool {
	_, err := os.Stat(path)

	return !os.IsNotExist(err)
}

func ensureDirExists(path string) error {
	if !dirExists(path) {
		return os.MkdirAll(path, os.ModeSetuid|os.ModeSetgid)
	}

	return nil
}

func (fs *Filesystem) getChunksPath(recorderID, sessionID uuid.UUID) string {
	return filepath.Join(fs.basepath, recorderID.String(), sessionsDir, sessionID.String(), chunksDir)
}

func (fs *Filesystem) initSession(recorderID, sessionID uuid.UUID) {
	log.Info().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Starting new session")

	fs.chunks[recorderID] = &filesystemChunk{
		number:    0,
		sessionID: sessionID,
	}
}

func (fs *Filesystem) renderClosedSession(recorderID, sessionID uuid.UUID) error {
	return nil
}

// The following implements the Storage interface

func (fs *Filesystem) SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, _ string, samples []int16) error {
	fs.chunksLock.Lock()
	defer fs.chunksLock.Unlock()

	chunkDir := fs.getChunksPath(recorderID, sessionID)
	if err := ensureDirExists(chunkDir); err != nil {
		return fmt.Errorf("%w: %w", errEnsureDir, err)
	}

	if _, ok := fs.chunks[sessionID]; !ok {
		fs.initSession(recorderID, sessionID)
	}

	chunk := fs.chunks[sessionID]
	if chunk.sessionID != sessionID {
		if err := fs.CloseSession(ctx, recorderID, chunk.sessionID); err != nil {
			log.Error().Err(err).Stringer("recorder-id", recorderID).Stringer("session-id", chunk.sessionID).Msg("Cannot close session")
		}

		go func(recorderID, sessionID uuid.UUID) {
			if err := fs.renderClosedSession(recorderID, chunk.sessionID); err != nil {
				log.Err(err).Stringer("recorder-id", recorderID).Stringer("session-id", chunk.sessionID).Msg("Cannot render closed session")
			}
		}(recorderID, chunk.sessionID)
	}

	chunkFile := filepath.Join(chunkDir, fmt.Sprintf("%08d.raw", chunk.number))

	f, err := os.Create(chunkFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := binary.Write(f, binary.LittleEndian, samples); err != nil {
		return err
	}

	chunk.number++
	fs.chunks[sessionID] = chunk

	return nil
}

func (fs *Filesystem) GetRecorderIDs(ctx context.Context) ([]uuid.UUID, error) {

	return []uuid.UUID{}, nil
}

func (fs *Filesystem) PutRecorderMetadata(ctx context.Context, recorderID uuid.UUID, recorder *model.RecorderMetadata) error {
	return nil
}

func (fs *Filesystem) GetRecorderMetadata(ctx context.Context, recorderID uuid.UUID) (*model.RecorderMetadata, error) {
	return nil, nil
}

func (fs *Filesystem) GetSessionIDs(ctx context.Context, recorderID uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func (fs *Filesystem) IsSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool {
	return false
}

func (fs *Filesystem) CloseSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	return nil
}

func (fs *Filesystem) CloseOpenSessions(ctx context.Context, recorderID uuid.UUID) error {
	return nil
}

func (fs *Filesystem) GetSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID) (*model.SessionMetadata, error) {
	return nil, nil
}

func (fs *Filesystem) PutSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID, session *model.SessionMetadata) error {
	return nil
}
