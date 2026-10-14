package main

import (
	"context"
	"encoding/binary"
	"errors"
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

// defaultCloseDelay is how long a session stays open after its recorder
// reported silence, and after each chunk that still arrives for it.
//
// The recorder sends its "no signal" status directly, but only queues its
// final chunk for a sender task - so the status regularly overtakes the last
// audio, and after a reconnect it can overtake a whole send buffer of it.
// Closing on the status itself cut that audio off (and, before the storage
// refused chunks for closed sessions, let it start the session over).
const defaultCloseDelay = 2 * time.Second

type ChunkSinkHandler struct {
	recorderBroadcaster *broadcast.RecorderBroadcaster
	sessionStorage      storage.Storage
	lock                sync.Mutex
	recorderStates      map[uuid.UUID]*recorderState
	connectedRecorders  map[uuid.UUID]struct{}
	// See defaultCloseDelay; a field so tests can shorten it.
	closeDelay time.Duration
}

type recorderState struct {
	lastSession uuid.UUID
	// Driven by the recorder's status alone - a chunk says nothing about
	// whether the recorder is still recording.
	recording bool
	// The session waiting to be closed once closeDelay has passed without
	// another chunk for it, and the timer that will do it.
	pendingSession uuid.UUID
	pendingClose   *time.Timer
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
		closeDelay:          defaultCloseDelay,
	}
}

// Called when a recorder sends status updates
func (h *ChunkSinkHandler) setRecorderStatus(ctx context.Context, status *cmpb.RecorderStatus) error {
	recorderID, err := uuid.Parse(status.RecorderID)
	if err != nil {
		log.Err(err).Str("recorder-id", status.RecorderID).Msg("Cannot parse recorder ID")
		return err
	}

	h.lock.Lock()
	state := h.recorderStates[recorderID]
	if state == nil {
		state = &recorderState{}
		h.recorderStates[recorderID] = state
	}
	wasRecording := state.recording
	nowRecording := status.SignalStatus == cmpb.SignalStatus_SIGNAL
	if wasRecording && !nowRecording && state.lastSession != uuid.Nil {
		// Not closed here: the session's last chunk is usually still on its
		// way (see defaultCloseDelay).
		h.scheduleCloseLocked(recorderID, state, state.lastSession)
	}
	state.recording = nowRecording
	h.lock.Unlock()

	h.sessionStorage.EnsureRecorderExists(ctx, recorderID, status.RecorderName)

	// Broadcast to all connected clients (non-blocking, with per-subscriber buffer).
	// The status is relayed as a whole, as the recorder sent it: clients read
	// more of it than the backend does (the audio fingerprints), so nothing
	// here picks fields out of it or builds a new one. The message is not
	// modified after this point - the broadcaster caches it and every
	// subscriber stream serializes this same instance.
	h.recorderBroadcaster.Broadcast(&sspb.Recorder{
		RecorderID:   recorderID.String(),
		RecorderName: status.RecorderName,
		Info: &sspb.Recorder_Status{
			Status: status,
		},
	})

	return nil
}

// scheduleCloseLocked arranges for sessionID to be closed once closeDelay has
// passed. Caller holds h.lock.
func (h *ChunkSinkHandler) scheduleCloseLocked(recorderID uuid.UUID, state *recorderState, sessionID uuid.UUID) {
	if state.pendingClose != nil {
		if state.pendingSession == sessionID {
			return
		}
		// A different session is still waiting: do not let it wait any longer.
		state.pendingClose.Stop()
		go h.closeAfterSilence(recorderID, state.pendingSession)
	}
	state.pendingSession = sessionID
	state.pendingClose = time.AfterFunc(h.closeDelay, func() {
		h.closeSilentSession(recorderID, sessionID)
	})
}

// closeSilentSession is the pending close firing. It does nothing if that
// close was cancelled, or already carried out, in the meantime.
func (h *ChunkSinkHandler) closeSilentSession(recorderID, sessionID uuid.UUID) {
	h.lock.Lock()
	state := h.recorderStates[recorderID]
	if state == nil || state.pendingSession != sessionID {
		h.lock.Unlock()
		return
	}
	state.pendingSession = uuid.Nil
	state.pendingClose = nil
	if state.lastSession == sessionID {
		state.lastSession = uuid.Nil
	}
	h.lock.Unlock()

	h.closeAfterSilence(recorderID, sessionID)
}

func (h *ChunkSinkHandler) closeAfterSilence(recorderID, sessionID uuid.UUID) {
	if closed, closeErr := h.closeRecorderSession(context.Background(), recorderID, sessionID); closeErr != nil {
		log.Err(closeErr).
			Str("recorder-id", recorderID.String()).
			Str("session-id", sessionID.String()).
			Msg("Cannot close session after recorder became silent")
	} else if closed {
		log.Info().
			Str("recorder-id", recorderID.String()).
			Str("session-id", sessionID.String()).
			Msg("Closed session after recorder became silent")
	}
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

// Called when a recorder sends chunks
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

	// data is packed little-endian s16 PCM (2 bytes per sample).
	raw := chunks.Data
	if len(raw)%2 != 0 {
		log.Warn().
			Str("recorder-id", recorderID.String()).
			Int("bytes", len(raw)).
			Msg("Chunk data has odd byte length; dropping trailing byte")
		raw = raw[:len(raw)-1]
	}
	samples := make([]int16, len(raw)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(raw[i*2:]))
	}

	timeCreated := chunks.TimeCreated.AsTime()

	if err = h.sessionStorage.SafeChunks(ctx, recorderID, sessionID, chunkID, timeCreated, samples); err != nil {
		if errors.Is(err, storage.ErrSessionClosed) {
			// Audio for a session that is already closed (it arrived later
			// than closeDelay after the recorder went silent). The storage
			// logged it and dropped it; nothing here changes for it - in
			// particular the session must not become "current" again.
			return nil
		}
		log.Err(err).Msg("Cannot save chunks")
		return nil
	}

	// Only a chunk the storage took makes its session the recorder's current
	// one, and it never says the recorder is recording: the status does.
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
	if state.pendingClose != nil {
		if state.pendingSession == sessionID {
			// Still audio coming in for the session about to be closed
			// (a recorder emptying its send buffer): wait for it to end.
			state.pendingClose.Reset(h.closeDelay)
		} else if state.pendingSession == previousSession {
			// The recorder has moved on to a new session; the storage
			// closes the old one itself when it switches.
			state.pendingClose.Stop()
			state.pendingSession = uuid.Nil
			state.pendingClose = nil
		}
	}
	h.lock.Unlock()

	if previousSession != uuid.Nil && previousSession != sessionID {
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
	h.lock.Lock()
	defer h.lock.Unlock()

	delete(h.connectedRecorders, recorderID)
	if state, ok := h.recorderStates[recorderID]; ok {
		// A pending close stays scheduled and fires as planned.
		state.recording = false
		state.lastSession = uuid.Nil
	}
}
