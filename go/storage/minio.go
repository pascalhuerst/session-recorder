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
	publicEndpoint string
	accessKey      string
	secretLey      string

	client *minio.Client

	// Key is recorder ID
	chunks        map[uuid.UUID]*minioChunk
	lastChunkTime map[uuid.UUID]time.Time // Track when each recorder last received chunks
	dataLock      sync.Mutex

	sessionTimeout time.Duration
	stopTimeout    chan struct{}

	onSessionClosedCb       OnSessionClosedCb
	onSessionStateChangedCb OnSessionStateChangedCb
	onAudioChunkCb          OnAudioChunkCb
	cbLock                  sync.Mutex
}

func NewMinioStorage(endpoint, publicEndpoint, accessKey, secretKey string) (*Minio, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create minio client: %w", err)
	}

	return &Minio{
		endpoint:       endpoint,
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

func (m *Minio) RegisterOnSessionClosedCallback(cb OnSessionClosedCb) error {
	m.cbLock.Lock()
	defer m.cbLock.Unlock()

	m.onSessionClosedCb = cb

	return nil
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

	// Before creating a new session, ensure all previous sessions are closed
	m.closeIntermediateSessions(ctx, recorderID)

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

	// Notify about new recording session
	m.notifyStateChange(&session, SessionStateUnknown)
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
		m.initSession(ctx, recorderID, sessionID, timeCreated)
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

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Done flushing chunks")

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

	ui, err := m.client.ComposeObject(ctx, dst, srcs...)
	if err != nil {
		log.Err(err).Msg("Cannot compose object, too small. Will delete session.")

		return m.deleteSession(ctx, recorderID, sessionID)
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

	previousState := sm.State
	sm.Duration = time.Duration(durationSeconds) * time.Second
	sm.EndTime = sm.StartTime.Add(sm.Duration)
	// Don't set to FINISHED yet - wait for rendering to complete

	var rawData *minio.Object
	if rawData, err = m.client.GetObject(ctx, bucketName, dst.Object, minio.GetObjectOptions{}); err != nil {
		log.Err(err).Str("object", dst.Object).Msg("Cannot get object")

		return err
	}

	readers, writer, closer := makeReaders(4)
	eg, egCtx := errgroup.WithContext(ctx)

	// Setup some io for the rendering of waveform, overview, flac and ogg files
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

	// Create ogg file.
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
		// Set state to ERROR on rendering failure
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

	// Rendering succeeded - set state to FINISHED
	sm.State = SessionStateFinished
	sm.IsClosed = true
	m.dataLock.Lock()
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, sm); err != nil {
		log.Err(err).Msg("Cannot put session metadata")
	}
	m.dataLock.Unlock()

	if err := m.client.RemoveObject(ctx, bucketName, chunksPrefix, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		log.Err(err).Str("object", chunksPrefix).Msg("Cannot remove object")
	}

	// Notify state change to FINISHED
	m.notifyStateChange(sm, previousState)

	// Legacy callback for backward compatibility
	if m.onSessionClosedCb != nil {
		m.cbLock.Lock()
		m.onSessionClosedCb(sm)
		m.cbLock.Unlock()
	}

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
		recorderID, err := uuid.Parse(idString)
		if err != nil {
			log.Err(err).Str("recorder-id", object.Key).Msg("Cannot parse recorder ID")

			return nil, err
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

	m.system.Recorders[recorderID].Sessions[sessionID] = *session

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
		if err := m.closeSession(ctx, recorderID, sessionID, nil); err != nil {
			return fmt.Errorf("cannot close session: %w", err)
		}
	}

	log.Debug().Stringer("recorder-id", recorderID).Msg("Closed all sessions for recorder")

	return nil
}

// closeIntermediateSessions finds all sessions in RECORDING or PROCESSING state for a recorder
// and transitions them to be processed. This is called when a new session starts to ensure
// no sessions are left in an intermediate state.
func (m *Minio) closeIntermediateSessions(ctx context.Context, recorderID uuid.UUID) {
	recorder, ok := m.system.Recorders[recorderID]
	if !ok {
		return
	}

	for sessionID, session := range recorder.Sessions {
		if session.State == SessionStateRecording || session.State == SessionStateProcessing {
			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Str("state", session.State.String()).
				Msg("Found session in intermediate state, closing")

			// Transition to PROCESSING if still RECORDING
			if session.State == SessionStateRecording {
				previousState := session.State
				session.State = SessionStateProcessing
				if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
					log.Err(err).Msg("Cannot update session state to PROCESSING")
					continue
				}
				m.system.Recorders[recorderID].Sessions[sessionID] = session
				go m.notifyStateChange(&session, previousState)
			}

			// Kick off async rendering
			go func(recorderID, sessionID uuid.UUID) {
				if err := m.closeSessionAsync(context.Background(), recorderID, sessionID, nil); err != nil {
					log.Err(err).
						Stringer("recorder-id", recorderID).
						Stringer("session-id", sessionID).
						Msg("Cannot close intermediate session")
					return
				}
				log.Info().
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Msg("Intermediate session closed")
			}(recorderID, sessionID)
		}
	}
}

// closeSession handles full session closing including state transition.
// Used at startup when processing sessions that may still be in RECORDING state.
func (m *Minio) closeSession(ctx context.Context, recorderID, sessionID uuid.UUID, chunk *minioChunk) error {
	if m.isSessionClosed(ctx, recorderID, sessionID) {
		return nil
	}

	if chunk != nil {
		if err := m.flushChunks(ctx, recorderID, sessionID, chunk); err != nil {
			return fmt.Errorf("cannot flush session: %w", err)
		}
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

	if err := m.renderSession(ctx, recorderID, sessionID); err != nil {
		return fmt.Errorf("cannot render session: %w", err)
	}

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")

	return nil
}

// closeSessionAsync handles session closing when state was already transitioned to PROCESSING.
// Used when a new session starts and we need to finish processing the previous session.
func (m *Minio) closeSessionAsync(ctx context.Context, recorderID, sessionID uuid.UUID, chunk *minioChunk) error {
	if m.isSessionClosed(ctx, recorderID, sessionID) {
		return nil
	}

	if chunk != nil {
		if err := m.flushChunks(ctx, recorderID, sessionID, chunk); err != nil {
			return fmt.Errorf("cannot flush session: %w", err)
		}
	}

	// State transition to PROCESSING was already done synchronously in SafeChunks
	// Proceed directly to rendering
	if err := m.renderSession(ctx, recorderID, sessionID); err != nil {
		return fmt.Errorf("cannot render session: %w", err)
	}

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")

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
		values.Set("response-content-disposition", fmt.Sprintf("attachment; Filename=%s", signedFilename))
	}

	presignedURL, err := m.client.PresignedGetObject(ctx, bucketName, objectName, signing.Expires, values)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	publicUrl := strings.Replace(presignedURL.String(), m.endpoint, m.publicEndpoint, 1)

	return publicUrl, nil
}
