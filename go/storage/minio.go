package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qmuntal/stateless"

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

// Default session timeout - if no chunks arrive for this duration, the session is automatically closed
const DefaultSessionTimeout = 30 * time.Second

// Session timeout check interval
const sessionTimeoutCheckInterval = 5 * time.Second

type Minio struct {
	system *System

	endpoint       string
	localEndpoint  string // For URLs consumed by UI (browser-accessible)
	publicEndpoint string // For URLs shared externally (email downloads)
	accessKey      string
	secretLey      string

	client MinioClient

	// Key is recorder ID
	chunks        map[uuid.UUID]*minioChunk
	lastChunkTime map[uuid.UUID]time.Time // Track when each recorder last received chunks
	dataLock      sync.Mutex

	sessionTimeout time.Duration
	stopTimeout    chan struct{}

	onAudioChunkCb OnAudioChunkCb

	eventBus *EventBus

	// State machines: one per session, keyed by sessionID. Machines are internally thread-safe.
	sessionMachines map[uuid.UUID]*stateless.StateMachine
	machineLock     sync.Mutex

	// Work queue for async rendering (sessions + segments)
	renderQueue *workQueue
}

func NewMinioStorage(endpoint, localEndpoint, publicEndpoint, accessKey, secretKey string) (*Minio, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create minio client: %w", err)
	}

	// If localEndpoint not specified, default to endpoint
	if localEndpoint == "" {
		localEndpoint = endpoint
	}
	// If publicEndpoint not specified, default to localEndpoint
	if publicEndpoint == "" {
		publicEndpoint = localEndpoint
	}

	return &Minio{
		endpoint:        endpoint,
		localEndpoint:   localEndpoint,
		publicEndpoint:  publicEndpoint,
		accessKey:       accessKey,
		secretLey:       secretKey,
		client:          newRealMinioClient(c),
		chunks:          make(map[uuid.UUID]*minioChunk),
		lastChunkTime:   make(map[uuid.UUID]time.Time),
		sessionTimeout:  DefaultSessionTimeout,
		stopTimeout:     make(chan struct{}),
		eventBus:        NewEventBus(),
		sessionMachines: make(map[uuid.UUID]*stateless.StateMachine),
		renderQueue:     newWorkQueue(DefaultMaxRenderWorkers),
	}, nil
}

// NewMinioStorageWithClient creates a Minio storage using a provided MinioClient.
// This is intended for testing with a fake/in-memory client.
func NewMinioStorageWithClient(client MinioClient, endpoint, localEndpoint, publicEndpoint string) *Minio {
	if localEndpoint == "" {
		localEndpoint = endpoint
	}
	if publicEndpoint == "" {
		publicEndpoint = localEndpoint
	}
	return &Minio{
		endpoint:        endpoint,
		localEndpoint:   localEndpoint,
		publicEndpoint:  publicEndpoint,
		client:          client,
		chunks:          make(map[uuid.UUID]*minioChunk),
		lastChunkTime:   make(map[uuid.UUID]time.Time),
		sessionTimeout:  DefaultSessionTimeout,
		stopTimeout:     make(chan struct{}),
		eventBus:        NewEventBus(),
		sessionMachines: make(map[uuid.UUID]*stateless.StateMachine),
		renderQueue:     newWorkQueue(DefaultMaxRenderWorkers),
	}
}

// SetSessionTimeout configures the session timeout duration.
// Sessions that don't receive chunks for this duration are automatically closed.
// Safe to call before Start. If called after Start, the new timeout takes effect
// on the next check cycle.
func (m *Minio) SetSessionTimeout(timeout time.Duration) {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()
	m.sessionTimeout = timeout
}

func (m *Minio) makeSureBucketExists(ctx context.Context) error {
	err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, err := m.client.BucketExists(ctx, bucketName)
		if err != nil {
			return fmt.Errorf("cannot check if bucket exists: %w", err)
		}

		if !exists {
			return fmt.Errorf("cannot create bucket: %w", err)
		}
	}

	return nil
}

func (m *Minio) Start(ctx context.Context) error {
	if err := m.makeSureBucketExists(ctx); err != nil {
		log.Err(err).Msg("Cannot create bucket")

		return err
	}

	var err error

	m.system, err = m.getSystemMetadata(ctx)
	if err != nil {
		log.Warn().Msg("Cannot get system metadata, creating...")

		system := &System{
			Recorders: make(map[uuid.UUID]Recorder),

			ID:   uuid.New(),
			Name: "Session Recorder",
		}

		if err := m.putSystemMetadata(ctx, system); err != nil {
			log.Err(err).Msg("Cannot put system metadata")

			return err
		}
	}

	recorderIDs, err := m.FindRecorderIDs(ctx)
	if err != nil {
		log.Err(err).Msg("Cannot read recorder IDs")

		return nil
	}

	if m.system.Recorders == nil {
		m.system.Recorders = make(map[uuid.UUID]Recorder)
	}

	for _, recorderID := range recorderIDs {
		recorderMetadata, err := m.getRecorderMetadata(ctx, recorderID)
		if err != nil {
			log.Warn().
				Err(err).
				Stringer("recorder-id", recorderID).
				Msg("Cannot get metadata, ignoring recorder")

			continue
		}

		if recorderMetadata.Sessions == nil {
			recorderMetadata.Sessions = make(map[uuid.UUID]Session)
		}

		m.system.Recorders[recorderID] = *recorderMetadata

		sessions, err := m.readSessionIDs(ctx, recorderID)
		if err != nil {
			log.Err(err).
				Stringer("recorder-id", recorderID).
				Msg("Cannot read session IDs")

			continue
		}

		for _, sessionID := range sessions {
			sessionMetadata, err := m.getSessionMetadata(ctx, recorderID, sessionID)
			if err != nil {
				log.Warn().
					Err(err).
					Stringer("session-id", sessionID).
					Msg("Cannot get metadata, ignoring session")

				continue
			}

			// Migration: Convert legacy IsClosed to State
			if sessionMetadata.State == SessionStateUnknown {
				if sessionMetadata.IsClosed {
					sessionMetadata.State = SessionStateFinished
				} else if m.isSessionClosed(ctx, recorderID, sessionID) {
					// No chunks folder means it was already rendered
					sessionMetadata.State = SessionStateFinished
				} else {
					// Has chunks folder, still recording
					sessionMetadata.State = SessionStateRecording
				}
				// Save migrated state
				if err := m.putSessionMetadata(ctx, recorderID, sessionID, sessionMetadata); err != nil {
					log.Warn().
						Err(err).
						Stringer("session-id", sessionID).
						Msg("Cannot save migrated session metadata")
				} else {
					log.Info().
						Stringer("session-id", sessionID).
						Str("state", sessionMetadata.State.String()).
						Msg("Migrated legacy session state")
				}
			}

			m.system.Recorders[recorderID].Sessions[sessionID] = *sessionMetadata

			// Create a state machine for each loaded session
			m.getOrCreateSessionMachine(recorderID, sessionID, sessionMetadata.State)
		}

		if err := m.closeSessions(ctx, recorderID); err != nil {
			log.Err(err).Msg("Cannot close sessions")

			continue
		}
	}

	// Start the session timeout checker
	go m.runSessionTimeoutChecker(ctx)

	return nil
}

// Stop stops the session timeout checker and drains the work queue.
func (m *Minio) Stop() {
	close(m.stopTimeout)
	if m.renderQueue != nil {
		m.renderQueue.stop()
	}
}

// getOrCreateSessionMachine returns the state machine for a session,
// creating one at the given initial state if it doesn't exist yet.
func (m *Minio) getOrCreateSessionMachine(recorderID, sessionID uuid.UUID, initialState SessionState) *stateless.StateMachine {
	m.machineLock.Lock()
	defer m.machineLock.Unlock()

	if sm, ok := m.sessionMachines[sessionID]; ok {
		return sm
	}

	sm := newSessionStateMachine(recorderID, sessionID, initialState, m.onSessionTransition)
	m.sessionMachines[sessionID] = sm
	return sm
}

// removeSessionMachine removes the state machine for a deleted session.
func (m *Minio) removeSessionMachine(sessionID uuid.UUID) {
	m.machineLock.Lock()
	defer m.machineLock.Unlock()
	delete(m.sessionMachines, sessionID)
}

// fireSessionTrigger fires a trigger on a session's state machine.
func (m *Minio) fireSessionTrigger(ctx context.Context, sessionID uuid.UUID, trigger sessionTrigger, errorMsg ...string) error {
	m.machineLock.Lock()
	sm, ok := m.sessionMachines[sessionID]
	m.machineLock.Unlock()

	if !ok {
		return fmt.Errorf("no state machine for session %s", sessionID)
	}

	// Store error message in context for triggerRenderFailure
	if trigger == triggerRenderFailure && len(errorMsg) > 0 {
		ctx = context.WithValue(ctx, renderErrorKey{}, errorMsg[0])
	}

	if err := sm.FireCtx(ctx, trigger); err != nil {
		return fmt.Errorf("cannot fire trigger %s on session %s: %w", trigger, sessionID, err)
	}

	return nil
}

// renderErrorKey is the context key for passing error messages through state transitions.
type renderErrorKey struct{}

// onSessionTransition is invoked during every session state transition.
// It persists the new state, updates the in-memory cache, and notifies subscribers.
// Called with the state machine's internal lock held, so transitions on the same session are serialized.
func (m *Minio) onSessionTransition(ctx context.Context, recorderID, sessionID uuid.UUID, trigger sessionTrigger, source, destination SessionState) {
	m.dataLock.Lock()

	recorder, ok := m.system.Recorders[recorderID]
	if !ok {
		m.dataLock.Unlock()
		log.Error().Stringer("recorder-id", recorderID).Msg("onSessionTransition: recorder not found")
		return
	}

	session, ok := recorder.Sessions[sessionID]
	if !ok {
		// Session might not be in the map yet (e.g., during initSession).
		// Create a minimal session entry — the caller will fill in details.
		session = Session{
			ID:         sessionID,
			RecorderID: recorderID,
			Segments:   make(map[uuid.UUID]Segment),
		}
	}

	previousState := session.State
	session.State = destination

	// Handle trigger-specific side effects
	switch trigger {
	case triggerRenderSuccess:
		session.IsClosed = true // backward compatibility
	case triggerRenderFailure:
		if errMsg, ok := ctx.Value(renderErrorKey{}).(string); ok {
			session.ErrorMessage = errMsg
		}
	case triggerRetryRender:
		session.ErrorMessage = ""
	}

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Str("trigger", string(trigger)).
			Msg("onSessionTransition: cannot persist state")
	}

	// Write updated session back to in-memory map (session is a value copy)
	m.system.Recorders[recorderID].Sessions[sessionID] = session

	m.dataLock.Unlock()

	// Notify outside all locks to prevent deadlocks
	m.eventBus.EmitSessionStateChanged(SessionStateChangedEvent{
		RecorderID:    recorderID,
		SessionID:     sessionID,
		PreviousState: previousState,
		NewState:      destination,
		Trigger:       string(trigger),
		ErrorMessage:  session.ErrorMessage,
		Session:       session,
	})
}

// runSessionTimeoutChecker periodically checks for stale sessions and closes them
func (m *Minio) runSessionTimeoutChecker(ctx context.Context) {
	ticker := time.NewTicker(sessionTimeoutCheckInterval)
	defer ticker.Stop()

	m.dataLock.Lock()
	timeout := m.sessionTimeout
	m.dataLock.Unlock()
	log.Info().
		Dur("timeout", timeout).
		Dur("check-interval", sessionTimeoutCheckInterval).
		Msg("Session timeout checker started")

	for {
		select {
		case <-m.stopTimeout:
			log.Info().Msg("Session timeout checker stopped")
			return
		case <-ctx.Done():
			log.Info().Msg("Session timeout checker stopped (context cancelled)")
			return
		case <-ticker.C:
			m.checkAndCloseStaleSession(ctx)
		}
	}
}

// checkAndCloseStaleSession checks for sessions that haven't received chunks recently and closes them
func (m *Minio) checkAndCloseStaleSession(ctx context.Context) {
	m.dataLock.Lock()

	now := time.Now()
	var staleRecorders []struct {
		recorderID uuid.UUID
		sessionID  uuid.UUID
		chunk      minioChunk
	}

	for recorderID, chunk := range m.chunks {
		lastTime, ok := m.lastChunkTime[recorderID]
		if !ok {
			continue
		}

		if now.Sub(lastTime) > m.sessionTimeout {
			log.Warn().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", chunk.sessionID).
				Dur("since-last-chunk", now.Sub(lastTime)).
				Msg("Session timed out, closing")

			staleRecorders = append(staleRecorders, struct {
				recorderID uuid.UUID
				sessionID  uuid.UUID
				chunk      minioChunk
			}{recorderID, chunk.sessionID, *chunk})

			// Remove from tracking maps
			delete(m.chunks, recorderID)
			delete(m.lastChunkTime, recorderID)
		}
	}

	m.dataLock.Unlock()

	// Transition and submit render jobs outside the lock
	for _, stale := range staleRecorders {
		// Transition to PROCESSING via FSM (callback acquires dataLock internally)
		if err := m.fireSessionTrigger(ctx, stale.sessionID, triggerCloseRecording); err != nil {
			log.Warn().Err(err).
				Stringer("session-id", stale.sessionID).
				Msg("Cannot transition timed-out session to PROCESSING")
		}

		chunkCopy := stale.chunk
		m.renderQueue.submitSessionRender(context.Background(), m, stale.recorderID, stale.sessionID, &chunkCopy)
	}
}

// EventBus returns the event bus for registering lifecycle event listeners.
func (m *Minio) EventBus() *EventBus {
	return m.eventBus
}

func (m *Minio) RegisterOnAudioChunkCallback(cb OnAudioChunkCb) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	m.onAudioChunkCb = cb

	return nil
}

func (m *Minio) initSession(ctx context.Context, recorderID, sessionID uuid.UUID, timeCreated time.Time) {
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Creating new session")

	// Before creating a new session, ensure all previous sessions are closed
	m.closeIntermediateSessions(ctx, recorderID)

	m.chunks[recorderID] = &minioChunk{
		number:    0,
		sessionID: sessionID,
		buffer:    new(bytes.Buffer),
	}

	// Create session directly at RECORDING state.
	// We skip the UNKNOWN→RECORDING FSM transition because initSession is called
	// while dataLock is held (from SafeChunks), and the FSM callback needs dataLock.
	// The FSM is initialized at RECORDING to stay in sync.
	session := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		StartTime:  timeCreated,
		EndTime:    time.Time{},
		Duration:   0,
		State:      SessionStateRecording,
		IsClosed:   false,
		Keep:       false,
		Segments:   make(map[uuid.UUID]Segment),
	}

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Cannot put session metadata")
		return
	}

	// Update in-memory map
	m.system.Recorders[recorderID].Sessions[sessionID] = session

	// Create state machine at RECORDING (already transitioned)
	m.getOrCreateSessionMachine(recorderID, sessionID, SessionStateRecording)

	// Notify about new recording session (emitted while dataLock is held by caller;
	// listeners must not call back into storage methods that acquire dataLock)
	m.eventBus.EmitSessionStateChanged(SessionStateChangedEvent{
		RecorderID:    recorderID,
		SessionID:     sessionID,
		PreviousState: SessionStateUnknown,
		NewState:      SessionStateRecording,
		Trigger:       string(triggerStartRecording),
		Session:       session,
	})
}

func (m *Minio) initRecorder(ctx context.Context, recorderID uuid.UUID, recorderName string) {
	log.Info().
		Stringer("recorder-id", recorderID).
		Msg("Creating new recorder")

	recorder := Recorder{
		ID:       recorderID,
		Name:     recorderName,
		Sessions: make(map[uuid.UUID]Session),
	}

	if err := m.putRecorderMetadata(ctx, recorderID,
		&Recorder{
			ID:       recorder.ID,
			Name:     recorder.Name,
			Sessions: make(map[uuid.UUID]Session),
		},
	); err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Msg("Cannot put recorder metadata")
	}
}

func (m *Minio) DeleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	m.removeSessionMachine(sessionID)

	return m.deleteSession(ctx, recorderID, sessionID)
}

func (m *Minio) deleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	chunksPrefix := fmt.Sprintf("%s/sessions/%s", recorderID, sessionID)

	log.Warn().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Deleting session")

	err := m.client.RemoveObject(ctx, bucketName, chunksPrefix, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		log.Err(err).Str("object", chunksPrefix).Msg("Cannot remove object")
	}

	if _, ok := m.system.Recorders[recorderID]; ok {
		delete(m.system.Recorders[recorderID].Sessions, sessionID)
	}

	return err
}

func (m *Minio) SetKeepSession(ctx context.Context, recorderID, sessionID uuid.UUID, keep bool) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		log.Warn().
			Stringer("recorder-id", recorderID).
			Msg("No recorder with this id")

		return fmt.Errorf("no recorder with this id")
	}

	if _, ok := m.system.Recorders[recorderID].Sessions[sessionID]; !ok {
		log.Warn().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("No session with this id")

		return fmt.Errorf("no session with this id")
	}

	session := m.system.Recorders[recorderID].Sessions[sessionID]
	session.Keep = keep

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Cannot put session metadata")

		return err
	}

	// Update in-memory cache
	m.system.Recorders[recorderID].Sessions[sessionID] = session

	return nil
}

func (m *Minio) SetName(ctx context.Context, recorderID, sessionID uuid.UUID, name string) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		log.Warn().
			Stringer("recorder-id", recorderID).
			Msg("No recorder with this id")

		return fmt.Errorf("no recorder with this id")
	}

	if _, ok := m.system.Recorders[recorderID].Sessions[sessionID]; !ok {
		log.Warn().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("No session with this id")

		return fmt.Errorf("no session with this id")
	}

	session := m.system.Recorders[recorderID].Sessions[sessionID]
	session.Name = name

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Cannot put session metadata")

		return err
	}

	// Update in-memory cache
	m.system.Recorders[recorderID].Sessions[sessionID] = session

	return nil
}

func (m *Minio) SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, _ string, timeCreated time.Time, samples []int16) error {
	// Collect deferred work that must happen outside dataLock
	var oldSessionID uuid.UUID
	var lastChunk *minioChunk
	var needsSessionSwitch bool

	m.dataLock.Lock()

	// Track when we last received chunks from this recorder
	m.lastChunkTime[recorderID] = time.Now()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		m.dataLock.Unlock()
		log.Warn().Stringer("recorder-id", recorderID).Msg("No recorder with this id")
		return fmt.Errorf("no recorder with this id")
	}

	if _, ok := m.chunks[recorderID]; !ok {
		// Check if session already exists (e.g., after backend restart).
		// If the recorder is sending chunks for a known session, resume it
		// regardless of current state — the recorder is the source of truth.
		if session, exists := m.system.Recorders[recorderID].Sessions[sessionID]; exists {
			chunkCount := m.countChunks(ctx, recorderID, sessionID)
			m.chunks[recorderID] = &minioChunk{
				number:    chunkCount,
				sessionID: sessionID,
				buffer:    new(bytes.Buffer),
			}

			if session.State != SessionStateRecording {
				// Session was in a non-RECORDING state (e.g., PROCESSING from a
				// previous crash, or FINISHED from a completed render). Reset to
				// RECORDING since the recorder is still actively sending chunks.
				previousState := session.State
				session.State = SessionStateRecording
				session.IsClosed = false
				session.ErrorMessage = ""
				if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
					log.Err(err).Msg("Cannot persist resumed session state")
				}
				m.system.Recorders[recorderID].Sessions[sessionID] = session

				// Replace FSM to match the corrected state
				m.removeSessionMachine(sessionID)
				m.getOrCreateSessionMachine(recorderID, sessionID, SessionStateRecording)

				m.eventBus.EmitSessionStateChanged(SessionStateChangedEvent{
					RecorderID:    recorderID,
					SessionID:     sessionID,
					PreviousState: previousState,
					NewState:      SessionStateRecording,
					Trigger:       "resumed-recording",
					Session:       session,
				})

				log.Info().
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Str("previous-state", previousState.String()).
					Int("existing-chunks", chunkCount).
					Msg("Resumed session from non-RECORDING state")
			} else {
				log.Info().
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Int("existing-chunks", chunkCount).
					Msg("Resuming existing RECORDING session")
			}
		} else {
			m.initSession(ctx, recorderID, sessionID, timeCreated)
		}
	}

	chunk := m.chunks[recorderID]

	// If we have a new sessionID, we need to close the old one
	if chunk.sessionID != sessionID {
		oldSessionID = chunk.sessionID
		lc := *chunk
		lastChunk = &lc
		needsSessionSwitch = true

		m.initSession(ctx, recorderID, sessionID, timeCreated)
		chunk = m.chunks[recorderID]
	}

	binary.Write(chunk.buffer, binary.LittleEndian, samples)

	// Broadcast audio samples for real-time streaming
	if m.onAudioChunkCb != nil {
		m.onAudioChunkCb(recorderID, sessionID, samples, chunk.number, timeCreated)
	}

	log.Debug().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msgf("Added %d|%d (%.1f %%) samples", chunk.buffer.Len(), minChunkSize, float64(chunk.buffer.Len())/float64(minChunkSize)*100.0)

	// To be able to concatinate the chunks, we need to make sure that the chunk size is at least 5MB
	if chunk.buffer.Len() >= minChunkSize {
		log.Debug().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Chunk is full")

		objectName := fmt.Sprintf("%s/sessions/%s/chunks/%s", recorderID, sessionID, fmt.Sprintf("%016d", chunk.number))
		chunk.number++

		_, err := m.client.PutObject(ctx, bucketName, objectName, chunk.buffer, int64(chunk.buffer.Len()), minio.PutObjectOptions{})
		if err != nil {
			m.dataLock.Unlock()
			return fmt.Errorf("cannot put object: %w", err)
		}
	}

	m.dataLock.Unlock()

	// Handle session switch outside dataLock — fireSessionTrigger's callback acquires it
	if needsSessionSwitch {
		if err := m.fireSessionTrigger(ctx, oldSessionID, triggerCloseRecording); err != nil {
			log.Warn().Err(err).
				Stringer("session-id", oldSessionID).
				Msg("Cannot transition old session to PROCESSING (may already be closed)")
		}

		// Submit render to work queue (non-blocking)
		m.renderQueue.submitSessionRender(context.Background(), m, recorderID, oldSessionID, lastChunk)
	}

	return nil
}

func (m *Minio) isSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool {
	// Check metadata first — if the session is already FINISHED or ERROR, it's closed
	// regardless of whether chunk objects still exist (they may not have been cleaned up).
	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err == nil && (sm.State == SessionStateFinished || sm.State == SessionStateError) {
		return true
	}

	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks/", recorderID, sessionID)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: false})

	// If chunks folder exists, the session is not closed
	for range objectCh {
		return false
	}

	return true
}

func (m *Minio) flushChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunk *minioChunk) error {
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Flushing chunks")

	// If we have samples left for this session, let's push those first
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

	log.Info().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Done flushing chunks")

	return nil
}

func (m *Minio) renderFromRawData(ctx context.Context, recorderID, sessionID uuid.UUID, rawData io.Reader) error {
	readers, writer, closer := makeReaders(4)
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer closer.Close()
		_, err := io.Copy(writer, rawData)
		if err != nil {
			log.Err(err).Msg("Cannot setup multiple readers")
			return err
		}
		return nil
	})

	eg.Go(func() error {
		waveformData, err := render.CreateWaveform(egCtx, readers[0], 300, 10000, 200)
		if err != nil {
			return fmt.Errorf("cannot create waveform: %w", err)
		}
		waveformObject := fmt.Sprintf("%s/sessions/%s/waveform.dat", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, waveformObject, waveformData, int64(waveformData.Len()), minio.PutObjectOptions{}); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		overviewData, err := render.CreateOverview(egCtx, readers[1], 300, 1000, 200)
		if err != nil {
			return fmt.Errorf("cannot create waveform overview: %w", err)
		}
		overviewObject := fmt.Sprintf("%s/sessions/%s/overview.png", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, overviewObject, overviewData, int64(overviewData.Len()), minio.PutObjectOptions{}); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		flacBuffer, err := render.Flac(readers[2])
		if err != nil {
			return err
		}
		flacObject := fmt.Sprintf("%s/sessions/%s/data.flac", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, flacObject, flacBuffer, int64(flacBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		buffer, err := render.CreateAudioFile(readers[3], "ogg")
		if err != nil {
			return err
		}
		object := fmt.Sprintf("%s/sessions/%s/data.ogg", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, object, buffer, int64(buffer.Len()), minio.PutObjectOptions{}); err != nil {
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		// Transition to ERROR via FSM
		if fireErr := m.fireSessionTrigger(ctx, sessionID, triggerRenderFailure, err.Error()); fireErr != nil {
			log.Err(fireErr).Msg("Cannot transition session to ERROR state")
		}
		return err
	}

	// Transition to FINISHED via FSM
	if err := m.fireSessionTrigger(ctx, sessionID, triggerRenderSuccess); err != nil {
		log.Err(err).Msg("Cannot transition session to FINISHED state")
	}

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Done rendering session")

	return nil
}

func (m *Minio) renderSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Rendering session")

	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks", recorderID, sessionID)
	rawDataObjectName := fmt.Sprintf("%s/sessions/%s/data.raw", recorderID, sessionID)

	// I need to double check this, but this dance does not seem to be necessary...
	//rawPreRenderedObjectName := fmt.Sprintf("%s/sessions/%s/pre_rendered_data.raw", RecorderID, sessionID)
	//
	//havePreRenderedData := false
	//
	//// There is an edge case. If the chunk-sink-server crashed, the current session is closed on m.Start()
	//// If the chunk-source still sends data for the same session, we just create a chunks folder again and drop
	//// the data. We can however check, if there is a chunks folder AND a data.raw. If that's the case, we can just append
	//// the data and regenerate the ogg and the flac file
	//_, err := m.client.StatObject(ctx, bucketName, rawDataObjectName, minio.StatObjectOptions{})
	//if minio.ToErrorResponse(err).Code != "NoSuchKey" {
	//	havePreRenderedData = true
	//
	//	log.Info().
	//		Stringer("recorder-id", RecorderID).
	//		Stringer("session-id", sessionID).
	//		Msg("Session already rendered, but more chunks found. Will re-render.")
	//
	//	dst := minio.CopyDestOptions{
	//		Bucket: bucketName,
	//		Object: rawPreRenderedObjectName,
	//	}
	//
	//	src := minio.CopySrcOptions{
	//		Bucket: bucketName,
	//		Object: rawDataObjectName,
	//	}
	//
	//	_, err = m.client.CopyObject(ctx, dst, src)
	//	if err != nil {
	//		log.Err(err).Msg("Could not extend pre-rendered data")
	//	}
	//}

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})
	srcs := make([]minio.CopySrcOptions, 0)

	// Just add the pre-rendered data to the list of sources first
	// that way, the new chunks should be appended
	//if havePreRenderedData {
	//	srcs = append(srcs, minio.CopySrcOptions{
	//		Bucket: bucketName,
	//		Object: rawPreRenderedObjectName,
	//	})
	//}

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
		Object: rawDataObjectName,
	}

	if len(srcs) == 0 {
		// No chunks — check if data.raw already exists (e.g. from a previous failed render)
		rawStat, err := m.client.StatObject(ctx, bucketName, rawDataObjectName, minio.StatObjectOptions{})
		if err != nil {
			log.Warn().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("No chunks found and no data.raw, nothing to render")
			return fmt.Errorf("no chunks to compose for session %s", sessionID)
		}

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Int64("size", rawStat.Size).
			Msg("No chunks found, but data.raw exists. Re-rendering from existing raw data.")

		sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
		if err != nil {
			return fmt.Errorf("cannot get session metadata: %w", err)
		}

		const bytesPerSecond float64 = 48000.0 * 2.0 * 2.0
		durationSeconds := float64(rawStat.Size) / bytesPerSecond
		sm.Duration = time.Duration(durationSeconds) * time.Second
		sm.EndTime = sm.StartTime.Add(sm.Duration)

		// Persist duration/endtime before rendering
		m.dataLock.Lock()
		if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
			m.dataLock.Unlock()
			return fmt.Errorf("cannot persist session duration: %w", err)
		}
		m.dataLock.Unlock()

		var rawData ObjectHandle
		if rawData, err = m.client.GetObject(ctx, bucketName, rawDataObjectName, minio.GetObjectOptions{}); err != nil {
			return fmt.Errorf("cannot get raw data: %w", err)
		}

		return m.renderFromRawData(ctx, recorderID, sessionID, rawData)
	}

	ui, err := m.client.ComposeObject(ctx, dst, srcs...)
	if err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Int("chunks", len(srcs)).
			Msg("Cannot compose object")
		return fmt.Errorf("cannot compose object: %w", err)
	}

	// We need to make this generic and nicer, hack for now to get teh exact end time for the session
	// raw samples for 48000hz, 2 Bytes (16bit), 2 channels
	const bytesPerSecond float64 = 48000.0 * 2.0 * 2.0
	durationSeconds := float64(ui.Size) / bytesPerSecond

	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		log.Err(err).Msg("Cannot get session metadata")
		return fmt.Errorf("cannot get session metadata: %w", err)
	}

	sm.Duration = time.Duration(durationSeconds) * time.Second
	sm.EndTime = sm.StartTime.Add(sm.Duration)

	// Persist duration/endtime before rendering
	m.dataLock.Lock()
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot persist session duration: %w", err)
	}
	m.dataLock.Unlock()

	var rawData ObjectHandle
	if rawData, err = m.client.GetObject(ctx, bucketName, dst.Object, minio.GetObjectOptions{}); err != nil {
		log.Err(err).Str("object", dst.Object).Msg("Cannot get object")

		return err
	}

	if err := m.renderFromRawData(ctx, recorderID, sessionID, rawData); err != nil {
		return err
	}

	if err := m.client.RemoveObject(ctx, bucketName, chunksPrefix, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		log.Err(err).Str("object", chunksPrefix).Msg("Cannot remove object")
	}

	return nil
}

func (m *Minio) FindRecorderIDs(ctx context.Context) ([]uuid.UUID, error) {
	recorders := make([]uuid.UUID, 0)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: "", Recursive: false})

	for object := range objectCh {
		if object.Err != nil {
			log.Err(object.Err).Msg("Cannot list objects")

			return nil, object.Err
		}

		if strings.Contains(object.Key, "metadata.json") {
			continue
		}

		idString, _ := strings.CutSuffix(object.Key, "/")
		if idString == "" {
			continue
		}

		recorderID, err := uuid.Parse(idString)
		if err != nil {
			log.Warn().
				Str("object", object.Key).
				Msg("Skipping non-recorder object while scanning")
			continue
		}

		recorders = append(recorders, recorderID)
	}

	return recorders, nil
}

func (m *Minio) GetRecorders() map[uuid.UUID]Recorder {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	return m.system.Recorders
}

func (m *Minio) GetSessions(recorderID uuid.UUID) map[uuid.UUID]Session {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	return m.system.Recorders[recorderID].Sessions
}

func (m *Minio) GetSession(recorderID, sessionID uuid.UUID) (Session, error) {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		return Session{}, fmt.Errorf("no recorder with this id")
	}

	if _, ok := m.system.Recorders[recorderID].Sessions[sessionID]; !ok {
		return Session{}, fmt.Errorf("no session with this id")
	}

	return m.system.Recorders[recorderID].Sessions[sessionID], nil
}

func (m *Minio) readSessionIDs(ctx context.Context, recorderID uuid.UUID) ([]uuid.UUID, error) {
	sessions := map[uuid.UUID]struct{}{}

	prefix := fmt.Sprintf("%s/sessions", recorderID)

	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})

	for object := range objectCh {
		if object.Err != nil {
			log.Err(object.Err).Msg("Cannot list objects")

			return nil, object.Err
		}

		if strings.Contains(object.Key, "metadata.json") {
			continue
		}

		tokens := strings.Split(object.Key, "/")
		if len(tokens) < 3 {
			continue
		}

		idString := tokens[2]
		if idString == "" {
			continue
		}

		sessionID, err := uuid.Parse(idString)
		if err != nil {
			log.Warn().
				Err(err).
				Str("object", object.Key).
				Msg("Skipping non-session object while scanning")
			continue
		}

		sessions[sessionID] = struct{}{}
	}

	ret := make([]uuid.UUID, 0)
	for sessionID := range sessions {
		ret = append(ret, sessionID)
	}

	return ret, nil
}

func (m *Minio) getSystemMetadata(ctx context.Context) (*System, error) {
	objectName := "metadata.json"

	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get object: %w", err)
	}

	buffer := new(bytes.Buffer)
	if _, err := buffer.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("cannot read object: %w", err)
	}

	metadata := &System{}
	err = json.Unmarshal(buffer.Bytes(), metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	return metadata, nil
}

func (m *Minio) putSystemMetadata(ctx context.Context, system *System) error {
	objectName := "metadata.json"

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(system)
	if err != nil {
		return fmt.Errorf("cannot marshal metadata: %w", err)
	}

	_, err = m.client.PutObject(ctx, bucketName, objectName, buffer, int64(buffer.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot put object: %w", err)
	}

	m.system = system

	return nil
}

func (m *Minio) getSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID) (*Session, error) {
	objectName := fmt.Sprintf("%s/sessions/%s/%s", recorderID, sessionID, FILENAME_METADATA)

	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get object: %w", err)
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(obj)

	if err != nil {
		return nil, fmt.Errorf("cannot read object: %w", err)
	}

	metadata := &Session{}
	err = json.Unmarshal(buffer.Bytes(), metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	return metadata, nil
}

func (m *Minio) putSessionMetadata(ctx context.Context, recorderID, sessionID uuid.UUID, session *Session) error {
	objectName := fmt.Sprintf("%s/sessions/%s/%s", recorderID, sessionID, FILENAME_METADATA)

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(session)
	if err != nil {
		return fmt.Errorf("cannot marshal metadata: %w", err)
	}

	_, err = m.client.PutObject(ctx, bucketName, objectName, buffer, int64(buffer.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot put object: %w", err)
	}

	if m.system.Recorders == nil {
		m.system.Recorders = make(map[uuid.UUID]Recorder)
	}

	recorder, ok := m.system.Recorders[recorderID]
	if !ok {
		recorder = Recorder{
			ID:       recorderID,
			Sessions: make(map[uuid.UUID]Session),
		}
	}

	if recorder.Sessions == nil {
		recorder.Sessions = make(map[uuid.UUID]Session)
	}

	recorder.Sessions[sessionID] = *session
	m.system.Recorders[recorderID] = recorder

	return nil
}

func (m *Minio) putRecorderMetadata(ctx context.Context, recorderID uuid.UUID, recorder *Recorder) error {
	objectName := fmt.Sprintf("%s/metadata.json", recorderID)

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(recorder)
	if err != nil {
		return fmt.Errorf("cannot marshal metadata: %w", err)
	}

	_, err = m.client.PutObject(ctx, bucketName, objectName, buffer, int64(buffer.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot put object: %w", err)
	}

	if m.system.Recorders == nil {
		m.system.Recorders = make(map[uuid.UUID]Recorder)
	}

	m.system.Recorders[recorderID] = *recorder

	return nil
}

func (m *Minio) getRecorderMetadata(ctx context.Context, recorderID uuid.UUID) (*Recorder, error) {
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

	metadata := &Recorder{}
	err = json.Unmarshal(buffer.Bytes(), metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	return metadata, nil
}

func (m *Minio) EnsureRecorderExists(ctx context.Context, recorderID uuid.UUID, recorderName string) {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		m.initRecorder(ctx, recorderID, recorderName)
	}
}

func (m *Minio) closeSessions(ctx context.Context, recorderID uuid.UUID) error {
	log.Debug().Stringer("recorder-id", recorderID).Msg("Closing all sessions for recorder")

	sessionIDs, err := m.readSessionIDs(ctx, recorderID)
	if err != nil {
		return fmt.Errorf("cannot read session IDs: %w", err)
	}

	for _, sessionID := range sessionIDs {
		sid := sessionID

		// Ensure machine exists
		session, ok := m.system.Recorders[recorderID].Sessions[sid]
		if !ok {
			continue
		}
		m.getOrCreateSessionMachine(recorderID, sid, session.State)

		if m.isSessionClosed(ctx, recorderID, sid) {
			continue
		}

		switch session.State {
		case SessionStateRecording:
			// Don't close immediately — set up chunk tracking so the timeout checker
			// can close it if no new chunks arrive within the timeout window.
			// If the recorder is still active, SafeChunks will resume naturally
			// because m.chunks[recorderID] already exists with the right sessionID.
			chunkCount := m.countChunks(ctx, recorderID, sid)
			m.chunks[recorderID] = &minioChunk{
				number:    chunkCount,
				sessionID: sid,
				buffer:    new(bytes.Buffer),
			}
			m.lastChunkTime[recorderID] = time.Now()
			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sid).
				Int("existing-chunks", chunkCount).
				Msg("Resuming RECORDING session on startup, waiting for chunks or timeout")

		case SessionStateProcessing:
			// Was mid-render when backend crashed — re-submit render
			m.renderQueue.submitSessionRender(context.Background(), m, recorderID, sid, nil)
		}
	}

	log.Debug().Stringer("recorder-id", recorderID).Msg("Submitted session closing jobs for recorder")

	return nil
}

// countChunks returns the number of chunk objects stored in MinIO for a session.
func (m *Minio) countChunks(ctx context.Context, recorderID, sessionID uuid.UUID) int {
	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks/", recorderID, sessionID)
	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})
	count := 0
	for obj := range objectCh {
		if obj.Err != nil {
			break
		}
		count++
	}
	return count
}

// closeIntermediateSessions finds all sessions in RECORDING or PROCESSING state for a recorder
// and transitions them to be processed. This is called when a new session starts to ensure
// no sessions are left in an intermediate state.
// closeIntermediateSessions finds all sessions in RECORDING or PROCESSING state for a recorder
// and transitions them to be processed. Called while dataLock is held (from initSession).
// State updates are done directly since we hold the lock; FSMs are synced to match.
func (m *Minio) closeIntermediateSessions(ctx context.Context, recorderID uuid.UUID) {
	recorder, ok := m.system.Recorders[recorderID]
	if !ok {
		return
	}

	// Collect sessions that need rendering (submitted outside this function,
	// but we can submit to the pool here since it's non-blocking)
	for sessionID, session := range recorder.Sessions {
		if session.State == SessionStateRecording || session.State == SessionStateProcessing {
			// Re-read metadata from storage to get the authoritative state.
			freshMeta, err := m.getSessionMetadata(ctx, recorderID, sessionID)
			if err == nil && (freshMeta.State == SessionStateFinished || freshMeta.State == SessionStateError) {
				log.Info().
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Str("stored-state", freshMeta.State.String()).
					Msg("Session already in terminal state, updating in-memory state")
				m.system.Recorders[recorderID].Sessions[sessionID] = *freshMeta
				// Sync FSM to terminal state
				m.getOrCreateSessionMachine(recorderID, sessionID, freshMeta.State)
				m.eventBus.EmitSessionStateChanged(SessionStateChangedEvent{
					RecorderID:    recorderID,
					SessionID:     sessionID,
					PreviousState: session.State,
					NewState:      freshMeta.State,
					Trigger:       "startup-recovery",
					ErrorMessage:  freshMeta.ErrorMessage,
					Session:       *freshMeta,
				})
				continue
			}

			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Str("state", session.State.String()).
				Msg("Found session in intermediate state, closing")

			// Transition to PROCESSING directly (we hold dataLock)
			if session.State == SessionStateRecording {
				previousState := session.State
				session.State = SessionStateProcessing
				if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
					log.Err(err).Msg("Cannot update session state to PROCESSING")
					continue
				}
				m.system.Recorders[recorderID].Sessions[sessionID] = session
				m.eventBus.EmitSessionStateChanged(SessionStateChangedEvent{
					RecorderID:    recorderID,
					SessionID:     sessionID,
					PreviousState: previousState,
					NewState:      SessionStateProcessing,
					Trigger:       "startup-close",
					Session:       session,
				})
			}

			// Create/sync FSM at PROCESSING state
			m.getOrCreateSessionMachine(recorderID, sessionID, SessionStateProcessing)

			// Submit render to work queue (non-blocking, safe to call while holding dataLock)
			m.renderQueue.submitSessionRender(context.Background(), m, recorderID, sessionID, nil)
		}
	}
}

// closeSessionAsync handles session closing when state was already transitioned to PROCESSING.
// Used by the work queue when a new session starts and we need to finish processing the previous session.
func (m *Minio) closeSessionAsync(ctx context.Context, recorderID, sessionID uuid.UUID, chunk *minioChunk) error {
	if chunk != nil {
		if err := m.flushChunks(ctx, recorderID, sessionID, chunk); err != nil {
			return fmt.Errorf("cannot flush session: %w", err)
		}
	}

	if m.isSessionClosed(ctx, recorderID, sessionID) {
		return nil
	}

	// Check if session was resumed (e.g., recorder reconnected after restart).
	// If it's back to RECORDING, skip the render — it's still active.
	m.dataLock.Lock()
	if recorder, ok := m.system.Recorders[recorderID]; ok {
		if session, ok := recorder.Sessions[sessionID]; ok && session.State == SessionStateRecording {
			m.dataLock.Unlock()
			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Session resumed recording, skipping render")
			return nil
		}
	}
	m.dataLock.Unlock()

	// State transition to PROCESSING was already done by the caller via FSM.
	// Proceed directly to rendering.
	if err := m.renderSession(ctx, recorderID, sessionID); err != nil {
		// renderFromRawData fires triggerRenderFailure internally, but ComposeObject
		// failures happen before that. Fire as safety net.
		m.fireSessionTrigger(ctx, sessionID, triggerRenderFailure, err.Error())
		return fmt.Errorf("cannot render session: %w", err)
	}

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")

	return nil
}

func (m *Minio) CloseRecordingSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	var chunkCopy *minioChunk

	m.dataLock.Lock()
	if chunk, ok := m.chunks[recorderID]; ok && chunk.sessionID == sessionID {
		copyChunk := *chunk
		chunkCopy = &copyChunk
		delete(m.chunks, recorderID)
		delete(m.lastChunkTime, recorderID)
	}

	// Estimate duration from known chunk data before transitioning state.
	// This lets the PROCESSING broadcast include an estimated duration for the UI.
	if chunkCopy != nil {
		const bytesPerSecond = 48000.0 * 2.0 * 2.0 // 48kHz, 16-bit, stereo
		// flushedChunks × minChunkSize + remaining buffer
		totalBytes := int64(chunkCopy.number)*int64(minChunkSize) + int64(chunkCopy.buffer.Len())
		estimatedDuration := time.Duration(float64(totalBytes) / bytesPerSecond * float64(time.Second))

		if recorder, ok := m.system.Recorders[recorderID]; ok {
			if session, ok := recorder.Sessions[sessionID]; ok {
				session.Duration = estimatedDuration
				session.EndTime = session.StartTime.Add(estimatedDuration)
				m.system.Recorders[recorderID].Sessions[sessionID] = session
				if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
					log.Err(err).Msg("Cannot persist estimated duration")
				}
			}
		}
	}
	m.dataLock.Unlock()

	// Transition to PROCESSING via FSM
	if err := m.fireSessionTrigger(ctx, sessionID, triggerCloseRecording); err != nil {
		return fmt.Errorf("cannot close recording session: %w", err)
	}

	// Submit render to work queue
	m.renderQueue.submitSessionRender(context.Background(), m, recorderID, sessionID, chunkCopy)

	return nil
}

func (m *Minio) RetryRenderSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	// FSM enforces that only ERROR -> PROCESSING is valid for this trigger
	if err := m.fireSessionTrigger(ctx, sessionID, triggerRetryRender); err != nil {
		return fmt.Errorf("cannot retry render session: %w", err)
	}

	// Submit render to work queue
	m.renderQueue.submitSessionRender(context.Background(), m, recorderID, sessionID, nil)

	return nil
}

func (m *Minio) GetPresignedURL(ctx context.Context, asset AssetOptions, signing SigningOptions) (string, error) {
	objectName := fmt.Sprintf("%s/sessions/%s/%s", asset.RecorderID.String(), asset.SessionID.String(), asset.Filename)

	values := make(url.Values)

	if signing.Download {
		signedFilename := signing.DownloadFilename
		if signedFilename == "" {
			signedFilename = string(asset.Filename)
		}
		values.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", signedFilename))
	}

	presignedURL, err := m.client.PresignedGetObject(ctx, bucketName, objectName, signing.Expires, values)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Choose endpoint based on signing options
	targetEndpoint := m.localEndpoint
	if signing.Endpoint == EndpointPublic {
		targetEndpoint = m.publicEndpoint
	}
	signedUrl := strings.Replace(presignedURL.String(), m.endpoint, targetEndpoint, 1)

	return signedUrl, nil
}

func (m *Minio) GetSegmentPresignedURL(ctx context.Context, asset SegmentAssetOptions, signing SigningOptions) (string, error) {
	objectName := fmt.Sprintf("%s/sessions/%s/segments/%s/%s",
		asset.RecorderID.String(), asset.SessionID.String(), asset.SegmentID.String(), asset.Filename)

	values := make(url.Values)

	if signing.Download {
		signedFilename := signing.DownloadFilename
		if signedFilename == "" {
			signedFilename = string(asset.Filename)
		}
		values.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", signedFilename))
	}

	presignedURL, err := m.client.PresignedGetObject(ctx, bucketName, objectName, signing.Expires, values)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Choose endpoint based on signing options
	targetEndpoint := m.localEndpoint
	if signing.Endpoint == EndpointPublic {
		targetEndpoint = m.publicEndpoint
	}
	signedUrl := strings.Replace(presignedURL.String(), m.endpoint, targetEndpoint, 1)

	return signedUrl, nil
}

func (m *Minio) GetSessionFileReader(ctx context.Context, asset AssetOptions) (io.ReadCloser, int64, error) {
	objectName := fmt.Sprintf("%s/sessions/%s/%s", asset.RecorderID.String(), asset.SessionID.String(), asset.Filename)

	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get object: %w", err)
	}

	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, fmt.Errorf("cannot stat object: %w", err)
	}

	return obj, info.Size, nil
}

func (m *Minio) GetSegmentFileReader(ctx context.Context, asset SegmentAssetOptions) (io.ReadCloser, int64, error) {
	objectName := fmt.Sprintf("%s/sessions/%s/segments/%s/%s",
		asset.RecorderID.String(), asset.SessionID.String(), asset.SegmentID.String(), asset.Filename)

	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get object: %w", err)
	}

	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, fmt.Errorf("cannot stat object: %w", err)
	}

	return obj, info.Size, nil
}

func (m *Minio) CreateSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, segment Segment) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}

	if session.Segments == nil {
		session.Segments = make(map[uuid.UUID]Segment)
	}

	segment.ID = segmentID
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Created segment")

	return nil
}

func (m *Minio) UpdateSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, segment Segment) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}

	existingSegment, ok := session.Segments[segmentID]
	if !ok {
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	segment.ID = segmentID

	// Check if start or end time changed on a rendered segment
	timeChanged := segment.StartPoint != existingSegment.StartPoint || segment.EndPoint != existingSegment.EndPoint
	wasRendered := existingSegment.State == SegmentStateFinished

	if timeChanged && wasRendered {
		// Delete rendered files since times changed
		segmentPrefix := fmt.Sprintf("%s/sessions/%s/segments/%s/", recorderID, sessionID, segmentID)

		// List and remove all segment files
		objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: segmentPrefix, Recursive: true})
		for obj := range objectCh {
			if obj.Err != nil {
				log.Warn().Err(obj.Err).Str("prefix", segmentPrefix).Msg("Error listing segment files")
				continue
			}
			if err := m.client.RemoveObject(ctx, bucketName, obj.Key, minio.RemoveObjectOptions{}); err != nil {
				log.Warn().Err(err).Str("object", obj.Key).Msg("Cannot remove segment file")
			}
		}

		// Reset state to unknown so user can re-render
		segment.State = SegmentStateUnknown
		segment.ErrorMessage = ""

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Stringer("segment-id", segmentID).
			Msg("Segment times changed, removed rendered files and reset state")
	} else {
		// Preserve state if not explicitly set
		if segment.State == SegmentStateUnknown {
			segment.State = existingSegment.State
		}
	}

	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Updated segment")

	return nil
}

func (m *Minio) DeleteSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}

	if _, ok := session.Segments[segmentID]; !ok {
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	delete(session.Segments, segmentID)

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	// Delete rendered segment files if they exist
	segmentPrefix := fmt.Sprintf("%s/sessions/%s/segments/%s", recorderID, sessionID, segmentID)
	if err := m.client.RemoveObject(ctx, bucketName, segmentPrefix, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		log.Warn().Err(err).Str("prefix", segmentPrefix).Msg("Cannot remove segment files")
	}

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Deleted segment")

	return nil
}

func (m *Minio) SetSegmentState(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, state SegmentState) error {
	m.dataLock.Lock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		m.dataLock.Unlock()
		return fmt.Errorf("no recorder with id %s", recorderID)
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		m.dataLock.Unlock()
		return fmt.Errorf("no session with id %s", sessionID)
	}

	segment, ok := session.Segments[segmentID]
	if !ok {
		m.dataLock.Unlock()
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	if err := validateSegmentTransition(segment.State, state); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot set segment state: %w", err)
	}

	previousState := segment.State
	segment.State = state
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	m.dataLock.Unlock()

	m.eventBus.EmitSegmentStateChanged(SegmentStateChangedEvent{
		RecorderID:    recorderID,
		SessionID:     sessionID,
		SegmentID:     segmentID,
		PreviousState: previousState,
		NewState:      state,
		Session:       session,
	})

	log.Debug().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Stringer("state", state).
		Msg("Set segment state")

	return nil
}

// RenderSegment enqueues a segment render job on the work queue.
// The caller should set the segment state to QUEUED before calling this.
// The work queue worker will transition QUEUED -> RENDERING -> FINISHED/ERROR.
func (m *Minio) RenderSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
	m.renderQueue.submitSegmentRender(ctx, m, recorderID, sessionID, segmentID)
	return nil
}

// renderSegmentSync performs the actual synchronous segment render.
// Called by the work queue worker via renderSegmentInternal.
func (m *Minio) renderSegmentSync(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
	// Get session and segment info
	m.dataLock.Lock()
	if _, ok := m.system.Recorders[recorderID]; !ok {
		m.dataLock.Unlock()
		return fmt.Errorf("no recorder with id %s", recorderID)
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		m.dataLock.Unlock()
		return fmt.Errorf("no session with id %s", sessionID)
	}

	segment, ok := session.Segments[segmentID]
	if !ok {
		m.dataLock.Unlock()
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	// Validate and set state to RENDERING
	if err := validateSegmentTransition(segment.State, SegmentStateRendering); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot start segment render: %w", err)
	}
	segment.State = SegmentStateRendering
	segment.ErrorMessage = ""
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	m.dataLock.Unlock()

	m.eventBus.EmitSegmentStateChanged(SegmentStateChangedEvent{
		RecorderID:    recorderID,
		SessionID:     sessionID,
		SegmentID:     segmentID,
		PreviousState: SegmentStateQueued,
		NewState:      SegmentStateRendering,
		Session:       session,
	})

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Int64("start", segment.StartPoint).
		Int64("end", segment.EndPoint).
		Msg("Rendering segment")

	// Validate segment range before starting render
	if segment.EndPoint <= segment.StartPoint {
		errMsg := fmt.Sprintf("invalid segment range: start=%d end=%d (zero or negative duration)", segment.StartPoint, segment.EndPoint)
		log.Error().
			Stringer("segment-id", segmentID).
			Int64("start", segment.StartPoint).
			Int64("end", segment.EndPoint).
			Msg("Segment has invalid range")
		m.setSegmentError(ctx, recorderID, sessionID, segmentID, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Get the raw audio file
	rawDataObjectName := fmt.Sprintf("%s/sessions/%s/data.raw", recorderID, sessionID)
	log.Info().Stringer("segment-id", segmentID).Str("object", rawDataObjectName).Msg("Fetching raw audio for segment")
	rawData, err := m.client.GetObject(ctx, bucketName, rawDataObjectName, minio.GetObjectOptions{})
	if err != nil {
		m.setSegmentError(ctx, recorderID, sessionID, segmentID, fmt.Sprintf("cannot get raw audio: %v", err))
		return fmt.Errorf("cannot get raw audio: %w", err)
	}
	log.Info().Stringer("segment-id", segmentID).Msg("Got raw audio handle, starting encoding")

	// Setup readers for parallel encoding
	readers, writer, closer := makeReaders(2)
	eg, egCtx := errgroup.WithContext(ctx)

	// Copy raw data to multiple readers
	eg.Go(func() error {
		defer closer.Close()
		log.Info().Stringer("segment-id", segmentID).Msg("Starting raw audio copy to encoders")
		n, err := io.Copy(writer, rawData)
		log.Info().Stringer("segment-id", segmentID).Int64("bytes", n).Err(err).Msg("Raw audio copy complete")
		// Ignore closed pipe errors - this is expected when sox finishes early
		// (sox only reads what it needs for the trim, then closes the pipe)
		if err != nil && (strings.Contains(err.Error(), "closed pipe") || strings.Contains(err.Error(), "broken pipe")) {
			log.Debug().Stringer("segment-id", segmentID).Msg("Pipe closed by encoder (expected)")
			return nil
		}
		return err
	})

	// Encode to OGG
	eg.Go(func() error {
		log.Info().Stringer("segment-id", segmentID).Msg("Starting OGG encoding")
		oggBuffer, err := render.ClipAndEncodeOgg(readers[0], segment.StartPoint, segment.EndPoint)
		// Close the pipe reader to unblock the copy goroutine
		// (sox only reads what it needs, leaving the rest unread)
		if rc, ok := readers[0].(io.Closer); ok {
			rc.Close()
		}
		if err != nil {
			log.Error().Stringer("segment-id", segmentID).Err(err).Msg("OGG encoding failed")
			return fmt.Errorf("cannot encode segment to OGG: %w", err)
		}
		log.Info().Stringer("segment-id", segmentID).Int("size", oggBuffer.Len()).Msg("OGG encoding complete")

		oggObject := fmt.Sprintf("%s/sessions/%s/segments/%s/%s", recorderID, sessionID, segmentID, SEGMENT_FILENAME_OGG)
		if _, err := m.client.PutObject(egCtx, bucketName, oggObject, oggBuffer, int64(oggBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			log.Error().Stringer("segment-id", segmentID).Err(err).Msg("OGG upload failed")
			return fmt.Errorf("cannot upload OGG: %w", err)
		}

		log.Info().
			Stringer("segment-id", segmentID).
			Int("size", oggBuffer.Len()).
			Msg("Segment OGG uploaded")

		return nil
	})

	// Encode to FLAC
	eg.Go(func() error {
		log.Info().Stringer("segment-id", segmentID).Msg("Starting FLAC encoding")
		flacBuffer, err := render.ClipAndEncodeFlac(readers[1], segment.StartPoint, segment.EndPoint)
		// Close the pipe reader to unblock the copy goroutine
		// (sox only reads what it needs, leaving the rest unread)
		if rc, ok := readers[1].(io.Closer); ok {
			rc.Close()
		}
		if err != nil {
			log.Error().Stringer("segment-id", segmentID).Err(err).Msg("FLAC encoding failed")
			return fmt.Errorf("cannot encode segment to FLAC: %w", err)
		}
		log.Info().Stringer("segment-id", segmentID).Int("size", flacBuffer.Len()).Msg("FLAC encoding complete")

		flacObject := fmt.Sprintf("%s/sessions/%s/segments/%s/%s", recorderID, sessionID, segmentID, SEGMENT_FILENAME_FLAC)
		if _, err := m.client.PutObject(egCtx, bucketName, flacObject, flacBuffer, int64(flacBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			log.Error().Stringer("segment-id", segmentID).Err(err).Msg("FLAC upload failed")
			return fmt.Errorf("cannot upload FLAC: %w", err)
		}

		log.Info().
			Stringer("segment-id", segmentID).
			Int("size", flacBuffer.Len()).
			Msg("Segment FLAC uploaded")

		return nil
	})

	log.Info().Stringer("segment-id", segmentID).Msg("Waiting for segment encoding to complete")
	if err := eg.Wait(); err != nil {
		log.Error().Stringer("segment-id", segmentID).Err(err).Msg("Segment encoding errgroup failed")
		m.setSegmentError(ctx, recorderID, sessionID, segmentID, err.Error())
		return err
	}
	log.Info().Stringer("segment-id", segmentID).Msg("Segment encoding completed successfully")

	// Set state to FINISHED
	m.dataLock.Lock()
	session = m.system.Recorders[recorderID].Sessions[sessionID]
	segment = session.Segments[segmentID]
	segment.State = SegmentStateFinished
	segment.ErrorMessage = ""
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	m.dataLock.Unlock()

	m.eventBus.EmitSegmentStateChanged(SegmentStateChangedEvent{
		RecorderID:    recorderID,
		SessionID:     sessionID,
		SegmentID:     segmentID,
		PreviousState: SegmentStateRendering,
		NewState:      SegmentStateFinished,
		Session:       session,
	})

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Segment rendering complete")

	return nil
}

func (m *Minio) setSegmentError(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, errorMsg string) {
	m.dataLock.Lock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		m.dataLock.Unlock()
		return
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		m.dataLock.Unlock()
		return
	}

	segment, ok := session.Segments[segmentID]
	if !ok {
		m.dataLock.Unlock()
		return
	}

	previousState := segment.State
	segment.State = SegmentStateError
	segment.ErrorMessage = errorMsg
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		log.Err(err).Msg("Cannot update segment state to ERROR")
	}

	m.dataLock.Unlock()

	m.eventBus.EmitSegmentStateChanged(SegmentStateChangedEvent{
		RecorderID:    recorderID,
		SessionID:     sessionID,
		SegmentID:     segmentID,
		PreviousState: previousState,
		NewState:      SegmentStateError,
		ErrorMessage:  errorMsg,
		Session:       session,
	})

	log.Error().
		Stringer("segment-id", segmentID).
		Str("error", errorMsg).
		Msg("Segment rendering failed")
}

// renderSegmentInternal is the synchronous segment render used by the work queue.
func (m *Minio) renderSegmentInternal(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
	return m.renderSegmentSync(ctx, recorderID, sessionID, segmentID)
}
