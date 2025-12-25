package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
)

type ChunkSinkHandler struct {
	recorderBroadcaster *broadcast.RecorderBroadcaster
	sessionStorage      storage.Storage
}

func NewChunkSinkHandler(sessionStorage storage.Storage, recorderBroadcaster *broadcast.RecorderBroadcaster) *ChunkSinkHandler {
	return &ChunkSinkHandler{
		sessionStorage:      sessionStorage,
		recorderBroadcaster: recorderBroadcaster,
	}
}

// Called when a chunk-source sends status updates
func (h *ChunkSinkHandler) setRecorderStatus(ctx context.Context, status *cmpb.RecorderStatus) error {
	recorderID, err := uuid.Parse(status.RecorderID)
	if err != nil {
		log.Err(err).Str("recorder-id", status.RecorderID).Msg("Cannot parse recorder ID")
		return err
	}

	h.sessionStorage.EnsureRecorderExists(ctx, recorderID, status.RecorderName)

	// Broadcast to all connected clients (non-blocking, with per-subscriber buffer)
	h.recorderBroadcaster.Broadcast(&sspb.Recorder{
		RecorderID:   recorderID.String(),
		RecorderName: status.RecorderName,
		Info: &sspb.Recorder_Status{
			Status: status,
		},
	})

	return nil
}

// Called when a chunk-source sends chunks
func (h *ChunkSinkHandler) setChunks(ctx context.Context, chunks *cspb.Chunks) error {
	chunkID := fmt.Sprintf("%016d", chunks.ChunkCount)
	sessionID, err := uuid.Parse(chunks.SessionID)
	if err != nil {
		log.Err(err).Str("sesstion-id", chunks.SessionID).Msg("Cannot parse session ID")

		return err
	}

	recorderID, err := uuid.Parse(chunks.RecorderID)
	if err != nil {
		log.Err(err).Str("recorder-id", chunks.RecorderID).Msg("Cannot parse recorder ID")

		return err
	}

	// We have s16 samples, but stored int u32
	samples := make([]int16, 0, len(chunks.Data))
	for _, sample := range chunks.Data {
		samples = append(samples, int16(sample))
	}

	timeCreated := chunks.TimeCreated.AsTime()

	if err = h.sessionStorage.SafeChunks(ctx, recorderID, sessionID, chunkID, timeCreated, samples); err != nil {
		log.Err(err).Msg("Cannot save chunks")
	}

	return nil
}
