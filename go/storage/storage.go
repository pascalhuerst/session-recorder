package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/model"
	"github.com/rs/zerolog/log"
)

type Storage interface {
	SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunkID string, samples []int16) error
	GetRecorderIDs(ctx context.Context) ([]uuid.UUID, error)
	PutRecorderMetadata(ctx context.Context, recorderID uuid.UUID, recorder *model.RecorderMetadata) error
	GetRecorderMetadata(ctx context.Context, recorderID uuid.UUID) (*model.RecorderMetadata, error)

	GetSessionIDs(ctx context.Context, recorderID uuid.UUID) ([]uuid.UUID, error)
	IsSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool
	CloseSession(ctx context.Context, recorderID, sessionID uuid.UUID, workDir string) error
	CloseOpenSessions(ctx context.Context, recorderID uuid.UUID) error
	GetSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID) (*model.SessionMetadata, error)
	PutSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID, session *model.SessionMetadata) error
}

func NewSystem(s Storage) (*model.System, error) {
	ctx := context.Background()

	recorders := make(map[string]model.Recorder)

	recorderIDs, err := s.GetRecorderIDs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Cannot get recorders")

		return nil, err
	}

	for _, recorderID := range recorderIDs {
		recorder, err := makeRecorder(ctx, s, recorderID)
		if err != nil {
			log.Error().Err(err).Msg("Cannot make recorder")

			return nil, err
		}

		sessions, err := makeSessions(ctx, s, recorderID)
		if err != nil {
			log.Error().Err(err).Msg("Cannot make session")

			return nil, err
		}

		recorder.Sessions = sessions

		recorders[recorderID.String()] = recorder

	}

	return &model.System{
		Recorders: recorders,
	}, nil
}

func makeRecorder(ctx context.Context, s Storage, recorderID uuid.UUID) (model.Recorder, error) {
	md, err := s.GetRecorderMetadata(ctx, recorderID)
	if err != nil {
		log.Error().Err(err).Str("recorder-id", recorderID.String()).Msg("Cannot get recorder metadata")

		md = &model.RecorderMetadata{
			GenericMetadata: &model.GenericMetadata{},
		}
	}

	return model.Recorder{
		Metadata: md,
	}, nil
}

func makeSessions(ctx context.Context, s Storage, recorderID uuid.UUID) (map[string]model.Session, error) {
	sessionIDs, err := s.GetSessionIDs(ctx, recorderID)
	if err != nil {
		log.Error().Err(err).Str("recorder-id", recorderID.String()).Msg("Cannot get sessions")

		return nil, err
	}

	sessions := make(map[string]model.Session)

	for _, sessionID := range sessionIDs {
		md, err := s.GetSessionMetadata(ctx, recorderID, sessionID)
		if err != nil {
			log.Error().Err(err).Str("recorder-id", recorderID.String()).Str("session-id", sessionID.String()).Msg("Cannot get session metadata")

			continue
		}

		isClosed := s.IsSessionClosed(ctx, recorderID, sessionID)

		sessions[sessionID.String()] = model.Session{
			Metadata: md,
			IsClosed: isClosed,
		}
	}

	return sessions, nil
}
