package main

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	lock                sync.Mutex
	recorderStates      map[uuid.UUID]*recorderState
	connectedRecorders  map[uuid.UUID]struct{}
}

type recorderState struct {
	lastSession uuid.UUID
	recording   bool
}

type sessionCloser interface {
	CloseRecordingSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
}

func NewChunkSinkHandler(sessionStorage storage.Storage, recorderBroadcaster *broadcast.RecorderBroadcaster) *ChunkSinkHandler {
	return &ChunkSinkHandler{
		sessionStorage:      sessionStorage,
		recorderBroadcaster: recorderBroadcaster,
		recorderStates:      make(map[uuid.UUID]*recorderState),
		connectedRecorders:  make(map[uuid.UUID]struct{}),
	}
}

// Called when a chunk-source sends status updates
func (h *ChunkSinkHandler) setRecorderStatus(ctx context.Context, status *cmpb.RecorderStatus) error {
	recorderID, err := uuid.Parse(status.RecorderID)
	if err != nil {
		log.Err(err).Str("recorder-id", status.RecorderID).Msg("Cannot parse recorder ID")
		return err
	}

	var sessionToClose uuid.UUID

	h.lock.Lock()
	state := h.recorderStates[recorderID]
	if state == nil {
		state = &recorderState{}
		h.recorderStates[recorderID] = state
	}
	wasRecording := state.recording
	nowRecording := status.SignalStatus == cmpb.SignalStatus_SIGNAL
	if wasRecording && !nowRecording && state.lastSession != uuid.Nil {
		sessionToClose = state.lastSession
		state.lastSession = uuid.Nil
	}
	state.recording = nowRecording
	h.lock.Unlock()

	if sessionToClose != uuid.Nil {
		if closed, closeErr := h.closeRecorderSession(ctx, recorderID, sessionToClose); closeErr != nil {
			log.Err(closeErr).
				Str("recorder-id", recorderID.String()).
				Str("session-id", sessionToClose.String()).
				Msg("Cannot close session after recorder became silent")
		} else if closed {
			log.Info().
				Str("recorder-id", recorderID.String()).
				Str("session-id", sessionToClose.String()).
				Msg("Closed session after recorder became silent")
		}
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

func (h *ChunkSinkHandler) closeRecorderSession(ctx context.Context, recorderID, sessionID uuid.UUID) (bool, error) {
	if sessionID == uuid.Nil {
		return false, nil
	}

	closer, ok := h.sessionStorage.(sessionCloser)
	if !ok {
		return false, nil
	}

	if err := closer.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		return false, err
	}

	return true, nil
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

	var previousSession uuid.UUID

	h.lock.Lock()
	state := h.recorderStates[recorderID]
	if state == nil {
		state = &recorderState{}
		h.recorderStates[recorderID] = state
	}
	if state.lastSession != uuid.Nil && state.lastSession != sessionID {
		previousSession = state.lastSession
	}
	state.lastSession = sessionID
	state.recording = true
	h.lock.Unlock()

	// We have s16 samples, but stored int u32
	samples := make([]int16, 0, len(chunks.Data))
	for _, sample := range chunks.Data {
		samples = append(samples, int16(sample))
	}

	timeCreated := time.Now()
	if chunks.TimeCreated != nil {
		t := chunks.TimeCreated.AsTime()
		if !t.IsZero() {
			timeCreated = t
		}
	}

	if err = h.sessionStorage.SafeChunks(ctx, recorderID, sessionID, chunkID, timeCreated, samples); err != nil {
		log.Err(err).Msg("Cannot save chunks")
	} else if previousSession != uuid.Nil && previousSession != sessionID {
		if closed, closeErr := h.closeRecorderSession(ctx, recorderID, previousSession); closeErr != nil {
			log.Err(closeErr).
				Str("recorder-id", recorderID.String()).
				Str("session-id", previousSession.String()).
				Msg("Cannot close previous session after session switch")
		} else if closed {
			log.Info().
				Str("recorder-id", recorderID.String()).
				Str("session-id", previousSession.String()).
				Msg("Closed previous session after session switch")
		}
	}

	return nil
}

func (h *ChunkSinkHandler) OnRecorderConnected(recorderID uuid.UUID) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if _, ok := h.recorderStates[recorderID]; !ok {
		h.recorderStates[recorderID] = &recorderState{}
	}
	h.connectedRecorders[recorderID] = struct{}{}
}

func (h *ChunkSinkHandler) OnRecorderDisconnected(recorderID uuid.UUID) {
	var sessionToClose uuid.UUID

	h.lock.Lock()
	delete(h.connectedRecorders, recorderID)
	if state, ok := h.recorderStates[recorderID]; ok {
		if state.recording && state.lastSession != uuid.Nil {
			sessionToClose = state.lastSession
		}
		state.recording = false
		state.lastSession = uuid.Nil
	}
	h.lock.Unlock()

	if sessionToClose != uuid.Nil {
		if closed, err := h.closeRecorderSession(context.Background(), recorderID, sessionToClose); err != nil {
			log.Err(err).
				Str("recorder-id", recorderID.String()).
				Str("session-id", sessionToClose.String()).
				Msg("Cannot close session after recorder disconnected")
		} else if closed {
			log.Info().
				Str("recorder-id", recorderID.String()).
				Str("session-id", sessionToClose.String()).
				Msg("Closed session after recorder disconnected")
		}
	}
}

func (h *ChunkSinkHandler) IsRecorderConnected(recorderID uuid.UUID) bool {
	h.lock.Lock()
	defer h.lock.Unlock()

	_, ok := h.connectedRecorders[recorderID]
	return ok
}
