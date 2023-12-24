package storage

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunkID string, samples []int16) error
	CloseSession(ctx context.Context, sessionID uuid.UUID) error
	GetRecorders(ctx context.Context) ([]uuid.UUID, error)
	GetSessions(ctx context.Context, recorderID uuid.UUID) ([]uuid.UUID, error)
}
