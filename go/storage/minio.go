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

type minioChunk struct {
	number    int
	sessionID uuid.UUID
	buffer    *bytes.Buffer
	// pushedToMinio is true once at least one buffer has been uploaded as a
	// chunks/<n> object. Until then, no audio for this session exists on minio
	// — the in-memory buffer is the only copy.
	pushedToMinio bool
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

	client *minio.Client

	// Key is recorder ID
	chunks        map[uuid.UUID]*minioChunk
	lastChunkTime map[uuid.UUID]time.Time // Track when each recorder last received chunks
	dataLock      sync.Mutex

	sessionTimeout time.Duration
	stopTimeout    chan struct{}

	onSessionStateChangedCb OnSessionStateChangedCb
	onAudioChunkCb          OnAudioChunkCb
	cbLock                  sync.Mutex
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
		endpoint:       endpoint,
		localEndpoint:  localEndpoint,
		publicEndpoint: publicEndpoint,
		accessKey:      accessKey,
		secretLey:      secretKey,
		client:         c,
		chunks:         make(map[uuid.UUID]*minioChunk),
		lastChunkTime:  make(map[uuid.UUID]time.Time),
		sessionTimeout: DefaultSessionTimeout,
		stopTimeout:    make(chan struct{}),
	}, nil
}

// SetSessionTimeout configures the session timeout duration.
// Sessions that don't receive chunks for this duration are automatically closed.
func (m *Minio) SetSessionTimeout(timeout time.Duration) {
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

			m.system.Recorders[recorderID].Sessions[sessionID] = *sessionMetadata
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

// Stop stops the session timeout checker goroutine
func (m *Minio) Stop() {
	close(m.stopTimeout)
}

// Shutdown flushes every active recorder's in-flight buffer to a partial
// chunks/<n> object and marks the session metadata with the chunk number so
// the next Start can resume from where we left off.
//
// Padding is intentionally omitted: the partial is a small chunks/<n>
// which is always the highest-numbered chunk on disk. On resume we load it
// back into memory and delete it before any new chunks/<n+1> is uploaded,
// so it never ends up in the middle of a ComposeObject part-list.
func (m *Minio) Shutdown(ctx context.Context) error {
	m.dataLock.Lock()

	type pending struct {
		recorderID uuid.UUID
		sessionID  uuid.UUID
		chunk      minioChunk
	}
	var toFlush []pending
	for recorderID, chunk := range m.chunks {
		if chunk == nil || chunk.buffer == nil || chunk.buffer.Len() == 0 {
			continue
		}
		toFlush = append(toFlush, pending{
			recorderID: recorderID,
			sessionID:  chunk.sessionID,
			chunk:      *chunk,
		})
	}
	// Clear the maps; we own these buffers now and won't accept further writes.
	m.chunks = make(map[uuid.UUID]*minioChunk)
	m.lastChunkTime = make(map[uuid.UUID]time.Time)
	m.dataLock.Unlock()

	close(m.stopTimeout)

	for _, p := range toFlush {
		log.Info().
			Stringer("recorder-id", p.recorderID).
			Stringer("session-id", p.sessionID).
			Int("bytes", p.chunk.buffer.Len()).
			Int("chunk-number", p.chunk.number).
			Msg("Shutdown: flushing partial chunk")

		objectName := fmt.Sprintf("%s/sessions/%s/chunks/%s",
			p.recorderID, p.sessionID, fmt.Sprintf("%016d", p.chunk.number))

		if _, err := m.client.PutObject(
			ctx, bucketName, objectName,
			p.chunk.buffer, int64(p.chunk.buffer.Len()),
			minio.PutObjectOptions{},
		); err != nil {
			log.Err(err).
				Stringer("recorder-id", p.recorderID).
				Stringer("session-id", p.sessionID).
				Msg("Shutdown: cannot flush partial chunk")
			continue
		}

		// Persist the marker so the next Start knows this chunks/<n> is partial.
		sm, err := m.getSessionMetadata(ctx, p.recorderID, p.sessionID)
		if err != nil {
			log.Err(err).
				Stringer("recorder-id", p.recorderID).
				Stringer("session-id", p.sessionID).
				Msg("Shutdown: cannot read session metadata for partial-chunk marker")
			continue
		}
		n := p.chunk.number
		sm.PartialChunkNumber = &n
		m.dataLock.Lock()
		err = m.putSessionMetadata(ctx, p.recorderID, p.sessionID, sm)
		m.dataLock.Unlock()
		if err != nil {
			log.Err(err).
				Stringer("recorder-id", p.recorderID).
				Stringer("session-id", p.sessionID).
				Msg("Shutdown: cannot persist partial-chunk marker")
		}
	}

	log.Info().Int("flushed", len(toFlush)).Msg("Shutdown complete")
	return nil
}

// runSessionTimeoutChecker periodically checks for stale sessions and closes them
func (m *Minio) runSessionTimeoutChecker(ctx context.Context) {
	ticker := time.NewTicker(sessionTimeoutCheckInterval)
	defer ticker.Stop()

	log.Info().
		Dur("timeout", m.sessionTimeout).
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

			// Synchronously transition to PROCESSING
			sm, err := m.getSessionMetadata(ctx, recorderID, chunk.sessionID)
			if err == nil && sm.State == SessionStateRecording {
				previousState := sm.State
				sm.State = SessionStateProcessing
				if err := m.putSessionMetadata(ctx, recorderID, chunk.sessionID, sm); err != nil {
					log.Err(err).Msg("Cannot update session state to PROCESSING")
				}
				// Create a copy for the callback to avoid races with concurrent modifications
				sessionCopy := *sm
				// Notify outside lock to avoid deadlock
				go m.notifyStateChange(&sessionCopy, previousState)
			}

			// Remove from tracking maps
			delete(m.chunks, recorderID)
			delete(m.lastChunkTime, recorderID)
		}
	}

	m.dataLock.Unlock()

	// Process stale sessions asynchronously (outside the lock)
	for _, stale := range staleRecorders {
		go func(recorderID, sessionID uuid.UUID, chunk minioChunk) {
			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Closing timed-out session")

			if err := m.closeSessionAsync(context.Background(), recorderID, sessionID, &chunk); err != nil {
				log.Err(err).
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Msg("Cannot close timed-out session")
				return
			}

			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Timed-out session closed")
		}(stale.recorderID, stale.sessionID, stale.chunk)
	}
}

func (m *Minio) RegisterOnSessionStateChangedCallback(cb OnSessionStateChangedCb) error {
	m.cbLock.Lock()
	defer m.cbLock.Unlock()

	m.onSessionStateChangedCb = cb

	return nil
}

func (m *Minio) RegisterOnAudioChunkCallback(cb OnAudioChunkCb) error {
	m.cbLock.Lock()
	defer m.cbLock.Unlock()

	m.onAudioChunkCb = cb

	return nil
}

// notifyStateChange calls the state changed callback if registered
func (m *Minio) notifyStateChange(session *Session, previousState SessionState) {
	if m.onSessionStateChangedCb != nil {
		m.cbLock.Lock()
		m.onSessionStateChangedCb(session, previousState)
		m.cbLock.Unlock()
	}
}

func (m *Minio) initSession(ctx context.Context, recorderID, sessionID uuid.UUID, timeCreated time.Time) {
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Creating new session")

	// Note: stale-session cleanup is handled at startup by Start()'s
	// closeSessions, and during rotation by the explicit closeSessionAsync
	// spawned in SafeChunks. Calling closeIntermediateSessions from here would
	// race with that explicit close on the just-transitioned previous session.

	m.chunks[recorderID] = &minioChunk{
		number:    0,
		sessionID: sessionID,
		buffer:    new(bytes.Buffer),
	}

	session := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		StartTime:  timeCreated,
		EndTime:    time.Time{},
		Duration:   0,
		State:      SessionStateRecording,
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

	// Notify about new recording session
	m.notifyStateChange(&session, SessionStateUnknown)
}

// resumeSession rehydrates the in-memory chunk state for a session that was
// left in RECORDING by a previous backend process. If the metadata names a
// PartialChunkNumber, the corresponding chunks/<n> object is loaded back into
// the in-memory buffer and removed from disk so subsequent uploads can replace
// it with a full-size chunk. If there's no partial marker, chunk.number is
// advanced past whatever full chunks already exist on disk.
func (m *Minio) resumeSession(ctx context.Context, recorderID, sessionID uuid.UUID, sm *Session) error {
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Resuming session")

	buffer := new(bytes.Buffer)
	var nextNumber int

	if sm.PartialChunkNumber != nil {
		partialNumber := *sm.PartialChunkNumber
		partialKey := fmt.Sprintf("%s/sessions/%s/chunks/%s",
			recorderID, sessionID, fmt.Sprintf("%016d", partialNumber))

		obj, err := m.client.GetObject(ctx, bucketName, partialKey, minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("cannot read partial chunk %s: %w", partialKey, err)
		}
		if _, err := buffer.ReadFrom(obj); err != nil {
			obj.Close()
			return fmt.Errorf("cannot drain partial chunk %s: %w", partialKey, err)
		}
		obj.Close()

		if err := m.client.RemoveObject(ctx, bucketName, partialKey, minio.RemoveObjectOptions{}); err != nil {
			log.Warn().Err(err).Str("object", partialKey).Msg("Cannot remove loaded partial chunk")
		}

		nextNumber = partialNumber

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Int("chunk-number", partialNumber).
			Int("bytes", buffer.Len()).
			Msg("Loaded partial chunk back into memory")

		// Clear the marker on disk.
		sm.PartialChunkNumber = nil
	} else {
		nextNumber = m.countExistingChunks(ctx, recorderID, sessionID)
	}

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
		return fmt.Errorf("cannot persist resumed session metadata: %w", err)
	}

	m.chunks[recorderID] = &minioChunk{
		number:        nextNumber,
		sessionID:     sessionID,
		buffer:        buffer,
		pushedToMinio: nextNumber > 0,
	}
	m.lastChunkTime[recorderID] = time.Now()

	return nil
}

// countExistingChunks returns the count (i.e. the next free chunk number) of
// chunks/<n> objects on disk for a session.
func (m *Minio) countExistingChunks(ctx context.Context, recorderID, sessionID uuid.UUID) int {
	prefix := fmt.Sprintf("%s/sessions/%s/chunks/", recorderID, sessionID)
	objectCh := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: false})
	n := 0
	for o := range objectCh {
		if o.Err != nil {
			continue
		}
		n++
	}
	return n
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
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	// Track when we last received chunks from this recorder
	m.lastChunkTime[recorderID] = time.Now()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		log.Warn().Stringer("recorder-id", recorderID).Msg("No recorder with this id")

		return fmt.Errorf("no recorder with this id")
	}

	if _, ok := m.chunks[recorderID]; !ok {
		// First chunk after process start: see if there is a resumable
		// session on disk for this (recorder, session) pair.
		if sm, err := m.getSessionMetadata(ctx, recorderID, sessionID); err == nil &&
			sm.State == SessionStateRecording {
			if err := m.resumeSession(ctx, recorderID, sessionID, sm); err != nil {
				log.Err(err).
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Msg("Cannot resume session, starting fresh")
				m.initSession(ctx, recorderID, sessionID, timeCreated)
			}
		} else {
			m.initSession(ctx, recorderID, sessionID, timeCreated)
		}
		// The recorder has committed to this session id. Any other session
		// for this recorder still marked RECORDING (typically a session
		// preserved across a backend restart that the recorder did NOT pick
		// back up) will never see another sample — render and close it.
		m.closeOrphanRecordingSessions(ctx, recorderID, sessionID)
	}

	chunk := m.chunks[recorderID]

	// If we have a new sessionID, we need to close the old one
	// This creates a copy of the last chunk, initSession below resets the chunks
	if chunk.sessionID != sessionID {
		oldSessionID := chunk.sessionID

		// Synchronously transition old session to PROCESSING before starting new session
		// This ensures only one session per recorder can be in RECORDING state at a time
		sm, err := m.getSessionMetadata(ctx, recorderID, oldSessionID)
		if err == nil && sm.State == SessionStateRecording {
			previousState := sm.State
			sm.State = SessionStateProcessing
			if err := m.putSessionMetadata(ctx, recorderID, oldSessionID, sm); err != nil {
				log.Err(err).Msg("Cannot update session state to PROCESSING")
			}
			// Create a copy for the callback to avoid races with concurrent modifications
			sessionCopy := *sm
			// Notify outside lock to avoid deadlock (we're already holding dataLock)
			go m.notifyStateChange(&sessionCopy, previousState)
		}

		// Now process the old session asynchronously (flush + render)
		go func(recorderID uuid.UUID, lastChunk minioChunk) {
			sessionID := lastChunk.sessionID

			log.Info().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Closing session")

			if err := m.closeSessionAsync(context.Background(), recorderID, sessionID, &lastChunk); err != nil {
				log.Err(err).Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Cannot close session")

				return
			}

			log.Info().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")
		}(recorderID, *chunk)

		m.initSession(ctx, recorderID, sessionID, timeCreated)
		chunk = m.chunks[recorderID]
		// Same reasoning as the first-chunk branch: catch any other RECORDING
		// sessions that aren't the just-rotated one (already PROCESSING).
		m.closeOrphanRecordingSessions(ctx, recorderID, sessionID)
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
			return fmt.Errorf("cannot put object: %w", err)
		}
		chunk.pushedToMinio = true
	}

	return nil
}

func (m *Minio) isSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool {
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

// renderSession is the "compose-from-uploaded-chunks" rendering path. It
// concatenates every chunks/<n> object into data.raw, then runs the shared
// derived-file rendering. Use this when at least one chunk has been pushed to
// minio (chunk.pushedToMinio == true, or any chunks/<n> objects exist).
func (m *Minio) renderSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Rendering session (compose path)")

	if err := m.composeChunksIntoRaw(ctx, recorderID, sessionID); err != nil {
		return err
	}
	return m.renderDerivedFiles(ctx, recorderID, sessionID, true)
}

// renderInMemorySession is the "buffer-only" rendering path. The recorder went
// silent (or got cut) before the in-memory buffer ever reached minChunkSize,
// so nothing was uploaded as a chunks/<n>. The bytes are right here in
// memory — feed them straight into the renderers and upload data.raw in
// parallel. No compose, no round-trip through minio, no zero-padding.
func (m *Minio) renderInMemorySession(ctx context.Context, recorderID, sessionID uuid.UUID, buffer *bytes.Buffer) error {
	rawBytes := buffer.Bytes()
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Int("bytes", len(rawBytes)).
		Msg("Rendering session from in-memory buffer (nothing pushed to minio yet)")

	// raw samples: 48000 Hz, 2 channels, int16 LE
	const bytesPerSecond float64 = 48000.0 * 2.0 * 2.0
	durationSeconds := float64(len(rawBytes)) / bytesPerSecond

	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		return fmt.Errorf("cannot get session metadata: %w", err)
	}
	previousState := sm.State
	sm.Duration = time.Duration(durationSeconds) * time.Second
	sm.EndTime = sm.StartTime.Add(sm.Duration)

	rawDataObjectName := fmt.Sprintf("%s/sessions/%s/data.raw", recorderID, sessionID)
	waveformObject := fmt.Sprintf("%s/sessions/%s/waveform.dat", recorderID, sessionID)
	overviewObject := fmt.Sprintf("%s/sessions/%s/overview.png", recorderID, sessionID)
	flacObject := fmt.Sprintf("%s/sessions/%s/data.flac", recorderID, sessionID)
	oggObject := fmt.Sprintf("%s/sessions/%s/data.ogg", recorderID, sessionID)

	// Each renderer reads from its own bytes.Reader over the same underlying
	// slice — cheap (cursor only) and independent. data.raw is uploaded as a
	// fifth parallel task so it's available for later segment rendering.
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		_, err := m.client.PutObject(egCtx, bucketName, rawDataObjectName, bytes.NewReader(rawBytes), int64(len(rawBytes)), minio.PutObjectOptions{})
		if err != nil {
			log.Err(err).Str("object", rawDataObjectName).Msg("Cannot upload data.raw")
			return fmt.Errorf("cannot upload data.raw: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		waveformData, werr := render.CreateWaveform(egCtx, bytes.NewReader(rawBytes), 300, 10000, 200)
		if werr != nil {
			return fmt.Errorf("cannot create waveform: %w", werr)
		}
		if _, err := m.client.PutObject(egCtx, bucketName, waveformObject, waveformData, int64(waveformData.Len()), minio.PutObjectOptions{}); err != nil {
			return fmt.Errorf("cannot upload waveform: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		overviewData, oerr := render.CreateOverview(egCtx, bytes.NewReader(rawBytes), 300, 1000, 200)
		if oerr != nil {
			return fmt.Errorf("cannot create overview: %w", oerr)
		}
		if _, err := m.client.PutObject(egCtx, bucketName, overviewObject, overviewData, int64(overviewData.Len()), minio.PutObjectOptions{}); err != nil {
			return fmt.Errorf("cannot upload overview: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		flacBuffer, ferr := render.Flac(bytes.NewReader(rawBytes))
		if ferr != nil {
			return fmt.Errorf("cannot create flac: %w", ferr)
		}
		if _, err := m.client.PutObject(egCtx, bucketName, flacObject, flacBuffer, int64(flacBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			return fmt.Errorf("cannot upload flac: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		oggBuffer, oerr := render.CreateAudioFile(bytes.NewReader(rawBytes), "ogg")
		if oerr != nil {
			return fmt.Errorf("cannot create ogg: %w", oerr)
		}
		if _, err := m.client.PutObject(egCtx, bucketName, oggObject, oggBuffer, int64(oggBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			return fmt.Errorf("cannot upload ogg: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		sm.State = SessionStateError
		sm.ErrorMessage = err.Error()
		m.dataLock.Lock()
		if putErr := m.putSessionMetadata(ctx, recorderID, sessionID, sm); putErr != nil {
			log.Err(putErr).Msg("Cannot update session state to ERROR")
		}
		m.dataLock.Unlock()
		m.notifyStateChange(sm, previousState)
		return err
	}

	sm.State = SessionStateFinished
	m.dataLock.Lock()
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
		log.Err(err).Msg("Cannot put session metadata")
	}
	m.dataLock.Unlock()

	m.notifyStateChange(sm, previousState)
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Done rendering in-memory session")
	return nil
}

// composeChunksIntoRaw concatenates every chunks/<n> object for this session
// into a single data.raw, via S3 server-side composition.
func (m *Minio) composeChunksIntoRaw(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks", recorderID, sessionID)
	rawDataObjectName := fmt.Sprintf("%s/sessions/%s/data.raw", recorderID, sessionID)

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
		Object: rawDataObjectName,
	}

	if _, err := m.client.ComposeObject(ctx, dst, srcs...); err != nil {
		log.Err(err).Msg("Cannot compose object, too small. Will delete session.")
		return m.deleteSession(ctx, recorderID, sessionID)
	}

	return nil
}

// renderDerivedFiles assumes <session>/data.raw exists, then renders waveform,
// overview, flac and ogg files. If removeChunks is true, the chunks/ folder is
// purged on success (used by the compose path; the in-memory path has nothing
// to clean up there).
func (m *Minio) renderDerivedFiles(ctx context.Context, recorderID, sessionID uuid.UUID, removeChunks bool) error {
	chunksPrefix := fmt.Sprintf("%s/sessions/%s/chunks", recorderID, sessionID)
	rawDataObjectName := fmt.Sprintf("%s/sessions/%s/data.raw", recorderID, sessionID)

	rawData, err := m.client.GetObject(ctx, bucketName, rawDataObjectName, minio.GetObjectOptions{})
	if err != nil {
		log.Err(err).Str("object", rawDataObjectName).Msg("Cannot get object")
		return err
	}

	rawInfo, err := rawData.Stat()
	if err != nil {
		rawData.Close()
		return fmt.Errorf("cannot stat data.raw: %w", err)
	}

	// raw samples: 48000 Hz, 2 channels, int16 LE
	const bytesPerSecond float64 = 48000.0 * 2.0 * 2.0
	durationSeconds := float64(rawInfo.Size) / bytesPerSecond

	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		log.Err(err).Msg("Cannot get session metadata")
		return fmt.Errorf("cannot get session metadata: %w", err)
	}

	previousState := sm.State
	sm.Duration = time.Duration(durationSeconds) * time.Second
	sm.EndTime = sm.StartTime.Add(sm.Duration)
	// Don't set to FINISHED yet - wait for rendering to complete

	readers, writer, closer := makeReaders(4)
	eg, egCtx := errgroup.WithContext(ctx)

	// Pipe data.raw to the four render workers.
	eg.Go(func() error {
		defer closer.Close()
		_, err := io.Copy(writer, rawData)
		if err != nil {
			log.Err(err).Msg("Cannot setup multiple readers")
			return err
		}
		return nil
	})

	// Create waveform dat file.
	eg.Go(func() error {
		waveformData, err := render.CreateWaveform(egCtx, readers[0], 300, 10000, 200)
		if err != nil {
			log.Err(err).Msg("Cannot create waveform")
			return fmt.Errorf("cannot create waveform: %w", err)
		}
		waveformObject := fmt.Sprintf("%s/sessions/%s/waveform.dat", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, waveformObject, waveformData, int64(waveformData.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", waveformObject).Msg("Cannot put object")
			return err
		}
		return nil
	})

	// Create waveform png overview file.
	eg.Go(func() error {
		overviewData, err := render.CreateOverview(egCtx, readers[1], 300, 1000, 200)
		if err != nil {
			log.Err(err).Msg("Cannot create waveform overview")
			return fmt.Errorf("cannot create waveform overview: %w", err)
		}
		overviewObject := fmt.Sprintf("%s/sessions/%s/overview.png", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, overviewObject, overviewData, int64(overviewData.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", overviewObject).Msg("Cannot put object")
			return err
		}
		return nil
	})

	// Create flac file.
	eg.Go(func() error {
		flacBuffer, ferr := render.Flac(readers[2])
		if ferr != nil {
			log.Err(ferr).Msg("Cannot convert to flac")
			return ferr
		}
		flacObject := fmt.Sprintf("%s/sessions/%s/data.flac", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, flacObject, flacBuffer, int64(flacBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", flacObject).Msg("Cannot put object")
			return err
		}
		return nil
	})

	// Create ogg file.
	eg.Go(func() error {
		oggBuffer, oerr := render.CreateAudioFile(readers[3], "ogg")
		if oerr != nil {
			log.Err(oerr).Msg("Cannot convert to ogg")
			return oerr
		}
		object := fmt.Sprintf("%s/sessions/%s/data.ogg", recorderID, sessionID)
		if _, err := m.client.PutObject(ctx, bucketName, object, oggBuffer, int64(oggBuffer.Len()), minio.PutObjectOptions{}); err != nil {
			log.Err(err).Str("object", object).Msg("Cannot put object")
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		sm.State = SessionStateError
		sm.ErrorMessage = err.Error()
		m.dataLock.Lock()
		if putErr := m.putSessionMetadata(ctx, recorderID, sessionID, sm); putErr != nil {
			log.Err(putErr).Msg("Cannot update session state to ERROR")
		}
		m.dataLock.Unlock()
		m.notifyStateChange(sm, previousState)
		return err
	}

	sm.State = SessionStateFinished
	m.dataLock.Lock()
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
		log.Err(err).Msg("Cannot put session metadata")
	}
	m.dataLock.Unlock()

	if removeChunks {
		if err := m.client.RemoveObject(ctx, bucketName, chunksPrefix, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
			log.Err(err).Str("object", chunksPrefix).Msg("Cannot remove object")
		}
	}

	m.notifyStateChange(sm, previousState)
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Done rendering session")
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

// closeSessions is the startup cleanup pass. It closes PROCESSING sessions
// (those interrupted mid-render) but LEAVES RECORDING sessions alone so the
// recorder can resume them on its next SafeChunks call. If the recorder
// never reconnects, the RECORDING session simply stays on disk; the user
// can delete it via the UI / session_source_client.
func (m *Minio) closeSessions(ctx context.Context, recorderID uuid.UUID) error {
	log.Debug().Stringer("recorder-id", recorderID).Msg("Startup cleanup for recorder")

	sessionIDs, err := m.readSessionIDs(ctx, recorderID)
	if err != nil {
		return fmt.Errorf("cannot read session IDs: %w", err)
	}

	for _, sessionID := range sessionIDs {
		sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
		if err != nil {
			log.Warn().Err(err).
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Cannot read session metadata, skipping")
			continue
		}

		switch sm.State {
		case SessionStateRecording:
			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Bool("has-partial-chunk", sm.PartialChunkNumber != nil).
				Msg("Leaving RECORDING session on disk for resume")
			continue
		case SessionStateFinished, SessionStateError:
			// Terminal states — already rendered (or failed and recorded as such).
			// closeSession would see no chunks/<n> on disk (they were composed
			// away during render) and incorrectly treat the session as empty.
			continue
		}

		// PROCESSING / UNKNOWN: interrupted mid-render, finish the job.
		if err := m.closeSession(ctx, recorderID, sessionID, nil); err != nil {
			log.Err(err).
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Cannot close session on startup")
			continue
		}
	}

	log.Debug().Stringer("recorder-id", recorderID).Msg("Startup cleanup done")
	return nil
}

// closeOrphanRecordingSessions sweeps for sessions that are still in
// RECORDING state for this recorder but no longer match the live session id.
// The recorder has committed to a new session, so any prior RECORDING entry
// (typically one preserved across a backend restart that the recorder did
// not pick back up) will never see another sample. Caller must hold
// m.dataLock.
func (m *Minio) closeOrphanRecordingSessions(ctx context.Context, recorderID, exceptSessionID uuid.UUID) {
	recorder, ok := m.system.Recorders[recorderID]
	if !ok {
		return
	}
	for sessionID, session := range recorder.Sessions {
		if sessionID == exceptSessionID || session.State != SessionStateRecording {
			continue
		}

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Stringer("new-session-id", exceptSessionID).
			Msg("Orphan RECORDING session detected, closing")

		previousState := session.State
		session.State = SessionStateProcessing
		if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
			log.Err(err).Msg("Cannot update orphan session state to PROCESSING")
			continue
		}
		sessionCopy := session
		go m.notifyStateChange(&sessionCopy, previousState)

		sid := sessionID
		go func() {
			if err := m.closeSessionAsync(context.Background(), recorderID, sid, nil); err != nil {
				log.Err(err).
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sid).
					Msg("Cannot close orphan session")
			}
		}()
	}
}

// closeSession handles full session closing including state transition.
// Used at startup when processing sessions that may still be in RECORDING state.
// The startup path never has an in-memory chunk (the process just restarted),
// so the only thing to check is whether there are any chunks/<n> on disk.
func (m *Minio) closeSession(ctx context.Context, recorderID, sessionID uuid.UUID, chunk *minioChunk) error {
	hasOnDisk := !m.isSessionClosed(ctx, recorderID, sessionID)
	hasInMemory := chunk != nil && chunk.buffer != nil && chunk.buffer.Len() > 0

	if !hasOnDisk && !hasInMemory {
		// No raw chunks anywhere — either a genuinely empty session (orphan
		// metadata from a crash before any audio landed) or a session that has
		// already been rendered (chunks/<n> composed away into data.raw +
		// data.flac/data.ogg). Only the former should be deleted; never wipe
		// rendered output.
		sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
		if err == nil && (sm.State == SessionStateFinished || sm.State == SessionStateError) {
			log.Debug().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Str("state", sm.State.String()).
				Msg("Skipping empty-session cleanup (already rendered)")
			return nil
		}
		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Closing empty session (no chunks on disk, no in-memory buffer)")
		m.dataLock.Lock()
		defer m.dataLock.Unlock()
		return m.deleteSession(ctx, recorderID, sessionID)
	}

	// Update state to PROCESSING before rendering
	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err == nil && sm.State == SessionStateRecording {
		previousState := sm.State
		sm.State = SessionStateProcessing
		m.dataLock.Lock()
		if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
			log.Err(err).Msg("Cannot update session state to PROCESSING")
		}
		m.dataLock.Unlock()
		m.notifyStateChange(sm, previousState)
	}

	if err := m.dispatchRender(ctx, recorderID, sessionID, chunk, hasOnDisk, hasInMemory); err != nil {
		return fmt.Errorf("cannot render session: %w", err)
	}

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")

	return nil
}

// closeSessionAsync handles session closing when state was already transitioned to PROCESSING.
// Used when a new session starts and we need to finish processing the previous session.
func (m *Minio) closeSessionAsync(ctx context.Context, recorderID, sessionID uuid.UUID, chunk *minioChunk) error {
	hasOnDisk := !m.isSessionClosed(ctx, recorderID, sessionID)
	hasInMemory := chunk != nil && chunk.buffer != nil && chunk.buffer.Len() > 0

	if !hasOnDisk && !hasInMemory {
		sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
		if err == nil && (sm.State == SessionStateFinished || sm.State == SessionStateError) {
			log.Debug().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Str("state", sm.State.String()).
				Msg("Skipping empty-session cleanup (already rendered)")
			return nil
		}
		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Closing empty session (no chunks on disk, no in-memory buffer)")
		m.dataLock.Lock()
		defer m.dataLock.Unlock()
		return m.deleteSession(ctx, recorderID, sessionID)
	}

	// State transition to PROCESSING was already done synchronously in SafeChunks
	// Proceed directly to rendering
	if err := m.dispatchRender(ctx, recorderID, sessionID, chunk, hasOnDisk, hasInMemory); err != nil {
		return fmt.Errorf("cannot render session: %w", err)
	}

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")

	return nil
}

// dispatchRender picks the right rendering path based on what data exists:
//
//   - on-disk chunks AND a non-empty in-memory buffer: flush the remaining
//     buffer to a final chunks/<n>, then compose all chunks/<n> into data.raw
//     and render derived files. The buffer is appended (padded to minChunkSize
//     so multipart copy accepts it).
//   - on-disk chunks only: skip flush, just compose and render.
//   - in-memory only (recorder went silent before the buffer ever reached
//     minChunkSize): upload the buffer directly as data.raw, no padding,
//     no compose. This is the case that used to silently drop data.
//
// Caller has already verified at least one of hasOnDisk / hasInMemory is true.
func (m *Minio) dispatchRender(
	ctx context.Context,
	recorderID, sessionID uuid.UUID,
	chunk *minioChunk,
	hasOnDisk, hasInMemory bool,
) error {
	switch {
	case hasOnDisk && hasInMemory:
		if err := m.flushChunks(ctx, recorderID, sessionID, chunk); err != nil {
			return fmt.Errorf("cannot flush session: %w", err)
		}
		return m.renderSession(ctx, recorderID, sessionID)

	case hasOnDisk:
		return m.renderSession(ctx, recorderID, sessionID)

	case hasInMemory:
		return m.renderInMemorySession(ctx, recorderID, sessionID, chunk.buffer)

	default:
		return nil
	}
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
	m.dataLock.Unlock()

	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		return fmt.Errorf("cannot get session metadata: %w", err)
	}

	if sm.State == SessionStateRecording {
		previousState := sm.State
		sm.State = SessionStateProcessing
		if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
			return fmt.Errorf("cannot update session state to PROCESSING: %w", err)
		}
		sessionCopy := *sm
		go m.notifyStateChange(&sessionCopy, previousState)
	}

	go func(recorderID, sessionID uuid.UUID, chunk *minioChunk) {
		var chunkArg *minioChunk
		if chunk != nil {
			c := *chunk
			chunkArg = &c
		}
		if err := m.closeSessionAsync(context.Background(), recorderID, sessionID, chunkArg); err != nil {
			log.Err(err).
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Cannot close session asynchronously")
		}
	}(recorderID, sessionID, chunkCopy)

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
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}

	segment, ok := session.Segments[segmentID]
	if !ok {
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	segment.State = state
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	log.Debug().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Stringer("state", state).
		Msg("Set segment state")

	return nil
}

func (m *Minio) RenderSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
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

	// Set state to RENDERING
	segment.State = SegmentStateRendering
	segment.ErrorMessage = ""
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m.dataLock.Unlock()
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	m.dataLock.Unlock()

	// Notify about state change
	m.notifyStateChange(&session, session.State)

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

	// Notify about state change
	m.notifyStateChange(&session, session.State)

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Segment rendering complete")

	return nil
}

func (m *Minio) setSegmentError(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, errorMsg string) {
	m.dataLock.Lock()
	defer m.dataLock.Unlock()

	if _, ok := m.system.Recorders[recorderID]; !ok {
		return
	}

	session, ok := m.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return
	}

	segment, ok := session.Segments[segmentID]
	if !ok {
		return
	}

	segment.State = SegmentStateError
	segment.ErrorMessage = errorMsg
	session.Segments[segmentID] = segment

	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		log.Err(err).Msg("Cannot update segment state to ERROR")
	}

	// Notify about state change
	go m.notifyStateChange(&session, session.State)

	log.Error().
		Stringer("segment-id", segmentID).
		Str("error", errorMsg).
		Msg("Segment rendering failed")
}
