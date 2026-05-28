package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/render"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ErrNotSupportedByFsBackend is returned by Fs methods that have no meaning
// for a local-filesystem backend (e.g. presigned URLs).
var ErrNotSupportedByFsBackend = errors.New("not supported by filesystem storage backend")

// fsRecording tracks the open data.raw file for a recorder's active session.
type fsRecording struct {
	sessionID uuid.UUID
	file      *os.File
}

// Fs is a Storage backed by the local filesystem. The on-disk layout mirrors
// the MinIO bucket layout: <root>/<recorder-uuid>/{metadata.json, sessions/<session-uuid>/...}
// Sample data is appended directly to <session>/data.raw as it arrives, so
// there is no in-memory chunk buffer and no minimum part size.
type Fs struct {
	root   string
	system *System

	// Key is recorder ID.
	recordings    map[uuid.UUID]*fsRecording
	lastChunkTime map[uuid.UUID]time.Time
	dataLock      sync.Mutex

	sessionTimeout time.Duration
	stopTimeout    chan struct{}

	onSessionStateChangedCb OnSessionStateChangedCb
	onAudioChunkCb          OnAudioChunkCb
	cbLock                  sync.Mutex
}

// NewFsStorage creates an Fs storage backend rooted at the given directory.
// The directory is created if it does not exist.
func NewFsStorage(root string) (*Fs, error) {
	if root == "" {
		return nil, errors.New("fs storage root path cannot be empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create root directory %s: %w", root, err)
	}
	return &Fs{
		root:           root,
		recordings:     make(map[uuid.UUID]*fsRecording),
		lastChunkTime:  make(map[uuid.UUID]time.Time),
		sessionTimeout: DefaultSessionTimeout,
		stopTimeout:    make(chan struct{}),
	}, nil
}

func (f *Fs) SetSessionTimeout(timeout time.Duration) {
	f.sessionTimeout = timeout
}

// -----------------------------------------------------------------------------
// Path helpers
// -----------------------------------------------------------------------------

func (f *Fs) systemMetadataPath() string {
	return filepath.Join(f.root, string(FILENAME_METADATA))
}

func (f *Fs) recorderDir(id uuid.UUID) string {
	return filepath.Join(f.root, id.String())
}

func (f *Fs) recorderMetadataPath(id uuid.UUID) string {
	return filepath.Join(f.recorderDir(id), string(FILENAME_METADATA))
}

func (f *Fs) sessionDir(recorderID, sessionID uuid.UUID) string {
	return filepath.Join(f.recorderDir(recorderID), "sessions", sessionID.String())
}

func (f *Fs) sessionMetadataPath(recorderID, sessionID uuid.UUID) string {
	return filepath.Join(f.sessionDir(recorderID, sessionID), string(FILENAME_METADATA))
}

func (f *Fs) sessionFilePath(recorderID, sessionID uuid.UUID, filename Filename) string {
	return filepath.Join(f.sessionDir(recorderID, sessionID), string(filename))
}

func (f *Fs) segmentDir(recorderID, sessionID, segmentID uuid.UUID) string {
	return filepath.Join(f.sessionDir(recorderID, sessionID), "segments", segmentID.String())
}

func (f *Fs) segmentFilePath(recorderID, sessionID, segmentID uuid.UUID, filename Filename) string {
	return filepath.Join(f.segmentDir(recorderID, sessionID, segmentID), string(filename))
}

// -----------------------------------------------------------------------------
// Lifecycle
// -----------------------------------------------------------------------------

func (f *Fs) Start(ctx context.Context) error {
	var err error
	f.system, err = f.getSystemMetadata()
	if err != nil {
		log.Warn().Msg("Cannot get system metadata, creating...")
		f.system = &System{
			ID:        uuid.New(),
			Name:      "Session Recorder",
			Recorders: make(map[uuid.UUID]Recorder),
		}
		if err := f.putSystemMetadata(f.system); err != nil {
			log.Err(err).Msg("Cannot put system metadata")
			return err
		}
	}
	if f.system.Recorders == nil {
		f.system.Recorders = make(map[uuid.UUID]Recorder)
	}

	recorderIDs, err := f.findRecorderIDs()
	if err != nil {
		log.Err(err).Msg("Cannot read recorder IDs")
		return nil
	}

	for _, recorderID := range recorderIDs {
		recorderMetadata, err := f.getRecorderMetadata(recorderID)
		if err != nil {
			log.Warn().Err(err).Stringer("recorder-id", recorderID).Msg("Cannot get metadata, ignoring recorder")
			continue
		}
		if recorderMetadata.Sessions == nil {
			recorderMetadata.Sessions = make(map[uuid.UUID]Session)
		}
		f.system.Recorders[recorderID] = *recorderMetadata

		sessionIDs, err := f.readSessionIDs(recorderID)
		if err != nil {
			log.Err(err).Stringer("recorder-id", recorderID).Msg("Cannot read session IDs")
			continue
		}
		for _, sessionID := range sessionIDs {
			sessionMetadata, err := f.getSessionMetadata(recorderID, sessionID)
			if err != nil {
				log.Warn().Err(err).Stringer("session-id", sessionID).Msg("Cannot get metadata, ignoring session")
				continue
			}
			f.system.Recorders[recorderID].Sessions[sessionID] = *sessionMetadata
		}

		if err := f.closeSessions(ctx, recorderID); err != nil {
			log.Err(err).Msg("Cannot close sessions")
			continue
		}
	}

	go f.runSessionTimeoutChecker(ctx)
	return nil
}

func (f *Fs) Stop() {
	close(f.stopTimeout)
}

// Shutdown flushes active recordings on graceful exit. For the fs backend
// every sample is already on disk in <session>/data.raw the moment SafeChunks
// returns — there is no chunked staging — so the only thing to do is close
// the open file handles and leave the session in RECORDING state for resume.
func (f *Fs) Shutdown(ctx context.Context) error {
	f.dataLock.Lock()
	count := len(f.recordings)
	for recorderID, rec := range f.recordings {
		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", rec.sessionID).
			Msg("Shutdown: closing data.raw for active recording")
		_ = rec.file.Close()
	}
	f.recordings = make(map[uuid.UUID]*fsRecording)
	f.lastChunkTime = make(map[uuid.UUID]time.Time)
	f.dataLock.Unlock()

	close(f.stopTimeout)

	log.Info().Int("flushed", count).Msg("Shutdown complete")
	return nil
}

func (f *Fs) runSessionTimeoutChecker(ctx context.Context) {
	ticker := time.NewTicker(sessionTimeoutCheckInterval)
	defer ticker.Stop()

	log.Info().
		Dur("timeout", f.sessionTimeout).
		Dur("check-interval", sessionTimeoutCheckInterval).
		Msg("Session timeout checker started")

	for {
		select {
		case <-f.stopTimeout:
			log.Info().Msg("Session timeout checker stopped")
			return
		case <-ctx.Done():
			log.Info().Msg("Session timeout checker stopped (context cancelled)")
			return
		case <-ticker.C:
			f.checkAndCloseStaleSession(ctx)
		}
	}
}

func (f *Fs) checkAndCloseStaleSession(ctx context.Context) {
	f.dataLock.Lock()

	now := time.Now()
	type stale struct {
		recorderID uuid.UUID
		sessionID  uuid.UUID
	}
	var staleSessions []stale

	for recorderID, recording := range f.recordings {
		lastTime, ok := f.lastChunkTime[recorderID]
		if !ok {
			continue
		}
		if now.Sub(lastTime) > f.sessionTimeout {
			log.Warn().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", recording.sessionID).
				Dur("since-last-chunk", now.Sub(lastTime)).
				Msg("Session timed out, closing")

			staleSessions = append(staleSessions, stale{recorderID, recording.sessionID})

			// Synchronously transition to PROCESSING and close the file handle.
			sm, err := f.getSessionMetadata(recorderID, recording.sessionID)
			if err == nil && sm.State == SessionStateRecording {
				previousState := sm.State
				sm.State = SessionStateProcessing
				if err := f.putSessionMetadata(recorderID, recording.sessionID, sm); err != nil {
					log.Err(err).Msg("Cannot update session state to PROCESSING")
				}
				sessionCopy := *sm
				go f.notifyStateChange(&sessionCopy, previousState)
			}

			_ = recording.file.Close()
			delete(f.recordings, recorderID)
			delete(f.lastChunkTime, recorderID)
		}
	}

	f.dataLock.Unlock()

	for _, s := range staleSessions {
		go func(recorderID, sessionID uuid.UUID) {
			log.Info().
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Closing timed-out session")

			if err := f.closeSessionAsync(context.Background(), recorderID, sessionID); err != nil {
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
		}(s.recorderID, s.sessionID)
	}
}

// -----------------------------------------------------------------------------
// Callbacks
// -----------------------------------------------------------------------------

func (f *Fs) RegisterOnSessionStateChangedCallback(cb OnSessionStateChangedCb) error {
	f.cbLock.Lock()
	defer f.cbLock.Unlock()
	f.onSessionStateChangedCb = cb
	return nil
}

func (f *Fs) RegisterOnAudioChunkCallback(cb OnAudioChunkCb) error {
	f.cbLock.Lock()
	defer f.cbLock.Unlock()
	f.onAudioChunkCb = cb
	return nil
}

func (f *Fs) notifyStateChange(session *Session, previousState SessionState) {
	if f.onSessionStateChangedCb != nil {
		f.cbLock.Lock()
		f.onSessionStateChangedCb(session, previousState)
		f.cbLock.Unlock()
	}
}

// -----------------------------------------------------------------------------
// Session lifecycle
// -----------------------------------------------------------------------------

// initSession creates a new session and opens its data.raw file for appending.
// Caller must hold f.dataLock.
//
// Note: stale-session cleanup is handled at startup by Start()'s closeSessions
// and during rotation by the explicit close-and-render path in SafeChunks.
// Calling closeIntermediateSessions from here would race with that explicit
// close on the just-transitioned previous session.
func (f *Fs) initSession(ctx context.Context, recorderID, sessionID uuid.UUID, timeCreated time.Time) error {
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Creating new session")

	dir := f.sessionDir(recorderID, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create session directory: %w", err)
	}

	rawPath := f.sessionFilePath(recorderID, sessionID, FILENAME_RAW)
	// O_TRUNC: this is a brand new session — the file should not exist, but
	// if it somehow does (e.g. crashed mid-render), start fresh.
	file, err := os.OpenFile(rawPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open data.raw: %w", err)
	}
	f.recordings[recorderID] = &fsRecording{sessionID: sessionID, file: file}

	session := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		StartTime:  timeCreated,
		State:      SessionStateRecording,
		Segments:   make(map[uuid.UUID]Segment),
	}
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		log.Err(err).
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Cannot put session metadata")
		return err
	}

	f.notifyStateChange(&session, SessionStateUnknown)
	return nil
}

// resumeSession reopens a session's data.raw for appending (after a graceful
// shutdown). Caller must hold f.dataLock.
func (f *Fs) resumeSession(recorderID, sessionID uuid.UUID) error {
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Resuming session")

	rawPath := f.sessionFilePath(recorderID, sessionID, FILENAME_RAW)
	file, err := os.OpenFile(rawPath, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open data.raw for append: %w", err)
	}
	f.recordings[recorderID] = &fsRecording{sessionID: sessionID, file: file}
	f.lastChunkTime[recorderID] = time.Now()
	return nil
}

// initRecorder creates a new recorder. Caller must hold f.dataLock.
func (f *Fs) initRecorder(recorderID uuid.UUID, recorderName string) {
	log.Info().Stringer("recorder-id", recorderID).Msg("Creating new recorder")

	recorder := &Recorder{
		ID:       recorderID,
		Name:     recorderName,
		Sessions: make(map[uuid.UUID]Session),
	}
	if err := f.putRecorderMetadata(recorderID, recorder); err != nil {
		log.Err(err).Stringer("recorder-id", recorderID).Msg("Cannot put recorder metadata")
	}
}

func (f *Fs) EnsureRecorderExists(ctx context.Context, recorderID uuid.UUID, recorderName string) {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	if _, ok := f.system.Recorders[recorderID]; !ok {
		f.initRecorder(recorderID, recorderName)
	}
}

func (f *Fs) SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, _ string, timeCreated time.Time, samples []int16) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	f.lastChunkTime[recorderID] = time.Now()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		log.Warn().Stringer("recorder-id", recorderID).Msg("No recorder with this id")
		return fmt.Errorf("no recorder with this id")
	}

	rec, ok := f.recordings[recorderID]
	if !ok {
		// First chunk after process start: see if there is a resumable
		// session on disk for this (recorder, session) pair.
		if sm, err := f.getSessionMetadata(recorderID, sessionID); err == nil &&
			sm.State == SessionStateRecording {
			if err := f.resumeSession(recorderID, sessionID); err != nil {
				log.Err(err).
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sessionID).
					Msg("Cannot resume session, starting fresh")
				if err := f.initSession(ctx, recorderID, sessionID, timeCreated); err != nil {
					return err
				}
			}
		} else {
			if err := f.initSession(ctx, recorderID, sessionID, timeCreated); err != nil {
				return err
			}
		}
		rec = f.recordings[recorderID]
		// The recorder has committed to this session id. Any other session
		// for this recorder still marked RECORDING (typically preserved across
		// a backend restart that the recorder did NOT pick back up) will never
		// see another sample — render and close it.
		f.closeOrphanRecordingSessions(ctx, recorderID, sessionID)
	}

	// New session arrived while a previous one is still open: rotate.
	if rec.sessionID != sessionID {
		oldSessionID := rec.sessionID

		sm, err := f.getSessionMetadata(recorderID, oldSessionID)
		if err == nil && sm.State == SessionStateRecording {
			previousState := sm.State
			sm.State = SessionStateProcessing
			if err := f.putSessionMetadata(recorderID, oldSessionID, sm); err != nil {
				log.Err(err).Msg("Cannot update session state to PROCESSING")
			}
			sessionCopy := *sm
			go f.notifyStateChange(&sessionCopy, previousState)
		}

		_ = rec.file.Close()
		delete(f.recordings, recorderID)

		go func(recorderID, sessionID uuid.UUID) {
			log.Info().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Closing session")
			if err := f.closeSessionAsync(context.Background(), recorderID, sessionID); err != nil {
				log.Err(err).Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Cannot close session")
				return
			}
			log.Info().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")
		}(recorderID, oldSessionID)

		if err := f.initSession(ctx, recorderID, sessionID, timeCreated); err != nil {
			return err
		}
		rec = f.recordings[recorderID]
		// Same reasoning as the first-chunk branch: catch any other RECORDING
		// sessions that aren't the just-rotated one (already PROCESSING).
		f.closeOrphanRecordingSessions(ctx, recorderID, sessionID)
	}

	if err := binary.Write(rec.file, binary.LittleEndian, samples); err != nil {
		return fmt.Errorf("cannot append samples: %w", err)
	}

	if f.onAudioChunkCb != nil {
		// Cheap: chunkNumber isn't meaningful for fs since we stream straight
		// to a single file. Pass 0 to keep the callback contract.
		f.onAudioChunkCb(recorderID, sessionID, samples, 0, timeCreated)
	}

	return nil
}

// isSessionClosed reports whether the session has no in-progress data.raw
// (i.e. no captured audio waiting to be rendered).
func (f *Fs) isSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool {
	info, err := os.Stat(f.sessionFilePath(recorderID, sessionID, FILENAME_RAW))
	if err != nil {
		return true
	}
	return info.Size() == 0
}

func (f *Fs) renderSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Rendering session")

	rawPath := f.sessionFilePath(recorderID, sessionID, FILENAME_RAW)
	rawInfo, err := os.Stat(rawPath)
	if err != nil {
		return fmt.Errorf("cannot stat data.raw: %w", err)
	}
	if rawInfo.Size() == 0 {
		log.Warn().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Empty data.raw, deleting session")
		return f.deleteSession(ctx, recorderID, sessionID)
	}

	// raw samples: 48000 Hz, 2 channels, int16 LE
	const bytesPerSecond float64 = 48000.0 * 2.0 * 2.0
	durationSeconds := float64(rawInfo.Size()) / bytesPerSecond

	sm, err := f.getSessionMetadata(recorderID, sessionID)
	if err != nil {
		return fmt.Errorf("cannot get session metadata: %w", err)
	}
	previousState := sm.State
	sm.Duration = time.Duration(durationSeconds) * time.Second
	sm.EndTime = sm.StartTime.Add(sm.Duration)

	rawData, err := os.Open(rawPath)
	if err != nil {
		return fmt.Errorf("cannot open data.raw: %w", err)
	}
	defer rawData.Close()

	// Three parallel consumers of data.raw: waveform.dat, data.flac, data.ogg.
	// The reader count must equal the number of consumers or the MultiWriter
	// blocks on an undrained pipe.
	readers, writer, closer := makeReaders(3)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer closer.Close()
		_, err := io.Copy(writer, rawData)
		if err != nil && !isPipeClosed(err) {
			log.Err(err).Msg("Cannot setup multiple readers")
			return err
		}
		return nil
	})

	waveformReader := readers[0]
	flacReader := readers[1]
	oggReader := readers[2]

	eg.Go(func() error {
		defer closeReader(waveformReader)
		waveformData, err := render.CreateWaveform(waveformReader)
		if err != nil {
			return fmt.Errorf("cannot create waveform: %w", err)
		}
		return f.writeFile(f.sessionFilePath(recorderID, sessionID, FILENAME_WAVEFORM), waveformData.Bytes())
	})

	eg.Go(func() error {
		defer closeReader(flacReader)
		return f.writeFileStream(f.sessionFilePath(recorderID, sessionID, FILENAME_FLAC), func(w io.Writer) error {
			return render.Flac(w, flacReader)
		})
	})

	eg.Go(func() error {
		defer closeReader(oggReader)
		return f.writeFileStream(f.sessionFilePath(recorderID, sessionID, FILENAME_OGG), func(w io.Writer) error {
			return render.Opus(w, oggReader)
		})
	})

	if err := eg.Wait(); err != nil {
		sm.State = SessionStateError
		sm.ErrorMessage = err.Error()
		f.dataLock.Lock()
		if putErr := f.putSessionMetadata(recorderID, sessionID, sm); putErr != nil {
			log.Err(putErr).Msg("Cannot update session state to ERROR")
		}
		f.dataLock.Unlock()
		f.notifyStateChange(sm, previousState)
		return err
	}

	sm.State = SessionStateFinished
	f.dataLock.Lock()
	if err := f.putSessionMetadata(recorderID, sessionID, sm); err != nil {
		log.Err(err).Msg("Cannot put session metadata")
	}
	f.dataLock.Unlock()

	f.notifyStateChange(sm, previousState)

	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Done rendering session")
	return nil
}

// writeFile writes data atomically (temp file + rename).
func (f *Fs) writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create parent directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("cannot rename %s: %w", path, err)
	}
	return nil
}

// writeFileStream writes a file atomically (temp file + rename) by streaming:
// write is called with the destination file as an io.Writer, so the content is
// never fully buffered in memory — safe for multi-hour encodes on a Pi.
func (f *Fs) writeFileStream(path string, write func(io.Writer) error) (err error) {
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return fmt.Errorf("cannot create parent directory: %w", mkErr)
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", tmp, err)
	}
	defer func() {
		if err != nil {
			file.Close()
			os.Remove(tmp)
		}
	}()

	bw := bufio.NewWriterSize(file, 1<<20)
	if err = write(bw); err != nil {
		return err
	}
	if err = bw.Flush(); err != nil {
		return fmt.Errorf("cannot flush %s: %w", tmp, err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("cannot close %s: %w", tmp, err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("cannot rename %s: %w", path, err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Close paths
// -----------------------------------------------------------------------------

// closeOrphanRecordingSessions sweeps for sessions still in RECORDING state
// for this recorder that no longer match the live session id. The recorder
// has committed to a new session, so any prior RECORDING entry (typically
// preserved across a backend restart that the recorder did not pick up)
// will never see another sample. Caller must hold f.dataLock.
func (f *Fs) closeOrphanRecordingSessions(ctx context.Context, recorderID, exceptSessionID uuid.UUID) {
	recorder, ok := f.system.Recorders[recorderID]
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
		if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
			log.Err(err).Msg("Cannot update orphan session state to PROCESSING")
			continue
		}
		sessionCopy := session
		go f.notifyStateChange(&sessionCopy, previousState)

		sid := sessionID
		go func() {
			if err := f.closeSessionAsync(context.Background(), recorderID, sid); err != nil {
				log.Err(err).
					Stringer("recorder-id", recorderID).
					Stringer("session-id", sid).
					Msg("Cannot close orphan session")
			}
		}()
	}
}

// closeSessions is the startup cleanup pass. It closes PROCESSING sessions
// (those interrupted mid-render) but LEAVES RECORDING sessions alone so the
// recorder can resume them on its next SafeChunks call. If the recorder
// never reconnects, the RECORDING session simply stays on disk; the user
// can delete it via the UI.
func (f *Fs) closeSessions(ctx context.Context, recorderID uuid.UUID) error {
	log.Debug().Stringer("recorder-id", recorderID).Msg("Startup cleanup for recorder")

	sessionIDs, err := f.readSessionIDs(recorderID)
	if err != nil {
		return fmt.Errorf("cannot read session IDs: %w", err)
	}

	for _, sessionID := range sessionIDs {
		sm, err := f.getSessionMetadata(recorderID, sessionID)
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
				Msg("Leaving RECORDING session on disk for resume")
			continue
		case SessionStateFinished, SessionStateError:
			// Terminal states — already rendered (or failed). Don't re-process.
			continue
		}

		if err := f.closeSession(ctx, recorderID, sessionID); err != nil {
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

func (f *Fs) closeSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	if f.isSessionClosed(ctx, recorderID, sessionID) {
		return nil
	}

	sm, err := f.getSessionMetadata(recorderID, sessionID)
	if err == nil && sm.State == SessionStateRecording {
		previousState := sm.State
		sm.State = SessionStateProcessing
		if err := f.putSessionMetadata(recorderID, sessionID, sm); err != nil {
			log.Err(err).Msg("Cannot update session state to PROCESSING")
		}
		f.notifyStateChange(sm, previousState)
	}

	if err := f.renderSession(ctx, recorderID, sessionID); err != nil {
		return fmt.Errorf("cannot render session: %w", err)
	}
	return nil
}

func (f *Fs) closeSessionAsync(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	if f.isSessionClosed(ctx, recorderID, sessionID) {
		return nil
	}
	if err := f.renderSession(ctx, recorderID, sessionID); err != nil {
		return fmt.Errorf("cannot render session: %w", err)
	}
	log.Debug().Stringer("recorder-id", recorderID).Stringer("session-id", sessionID).Msg("Session closed")
	return nil
}

func (f *Fs) CloseRecordingSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	f.dataLock.Lock()
	if rec, ok := f.recordings[recorderID]; ok && rec.sessionID == sessionID {
		_ = rec.file.Close()
		delete(f.recordings, recorderID)
		delete(f.lastChunkTime, recorderID)
	}
	f.dataLock.Unlock()

	sm, err := f.getSessionMetadata(recorderID, sessionID)
	if err != nil {
		return fmt.Errorf("cannot get session metadata: %w", err)
	}
	if sm.State == SessionStateRecording {
		previousState := sm.State
		sm.State = SessionStateProcessing
		if err := f.putSessionMetadata(recorderID, sessionID, sm); err != nil {
			return fmt.Errorf("cannot update session state to PROCESSING: %w", err)
		}
		sessionCopy := *sm
		go f.notifyStateChange(&sessionCopy, previousState)
	}

	go func(recorderID, sessionID uuid.UUID) {
		if err := f.closeSessionAsync(context.Background(), recorderID, sessionID); err != nil {
			log.Err(err).
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Cannot close session asynchronously")
		}
	}(recorderID, sessionID)

	return nil
}

// -----------------------------------------------------------------------------
// Session/segment CRUD
// -----------------------------------------------------------------------------

func (f *Fs) DeleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	return f.deleteSession(ctx, recorderID, sessionID)
}

func (f *Fs) deleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error {
	log.Warn().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Msg("Deleting session")

	if err := os.RemoveAll(f.sessionDir(recorderID, sessionID)); err != nil {
		log.Err(err).Str("dir", f.sessionDir(recorderID, sessionID)).Msg("Cannot remove session directory")
		return err
	}

	if _, ok := f.system.Recorders[recorderID]; ok {
		delete(f.system.Recorders[recorderID].Sessions, sessionID)
	}
	return nil
}

func (f *Fs) SetKeepSession(ctx context.Context, recorderID, sessionID uuid.UUID, keep bool) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with this id")
	}
	if _, ok := f.system.Recorders[recorderID].Sessions[sessionID]; !ok {
		return fmt.Errorf("no session with this id")
	}

	session := f.system.Recorders[recorderID].Sessions[sessionID]
	session.Keep = keep
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		return err
	}
	f.system.Recorders[recorderID].Sessions[sessionID] = session
	return nil
}

func (f *Fs) SetName(ctx context.Context, recorderID, sessionID uuid.UUID, name string) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with this id")
	}
	if _, ok := f.system.Recorders[recorderID].Sessions[sessionID]; !ok {
		return fmt.Errorf("no session with this id")
	}

	session := f.system.Recorders[recorderID].Sessions[sessionID]
	session.Name = name
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		return err
	}
	f.system.Recorders[recorderID].Sessions[sessionID] = session
	return nil
}

// -----------------------------------------------------------------------------
// Accessors
// -----------------------------------------------------------------------------

func (f *Fs) GetRecorders() map[uuid.UUID]Recorder {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	return f.system.Recorders
}

func (f *Fs) GetSessions(recorderID uuid.UUID) map[uuid.UUID]Session {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	return f.system.Recorders[recorderID].Sessions
}

func (f *Fs) SnapshotSessions() []SessionRef {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	var out []SessionRef
	for recorderID, recorder := range f.system.Recorders {
		for sessionID, session := range recorder.Sessions {
			out = append(out, SessionRef{RecorderID: recorderID, SessionID: sessionID, Session: copySessionForSnapshot(session)})
		}
	}
	return out
}

func (f *Fs) GetSession(recorderID, sessionID uuid.UUID) (Session, error) {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	if _, ok := f.system.Recorders[recorderID]; !ok {
		return Session{}, fmt.Errorf("no recorder with this id")
	}
	if _, ok := f.system.Recorders[recorderID].Sessions[sessionID]; !ok {
		return Session{}, fmt.Errorf("no session with this id")
	}
	return f.system.Recorders[recorderID].Sessions[sessionID], nil
}

// -----------------------------------------------------------------------------
// Sharing — stubbed for the fs backend
// -----------------------------------------------------------------------------

func (f *Fs) GetPresignedURL(ctx context.Context, asset AssetOptions, signing SigningOptions) (string, error) {
	return "", ErrNotSupportedByFsBackend
}

func (f *Fs) GetSegmentPresignedURL(ctx context.Context, asset SegmentAssetOptions, signing SigningOptions) (string, error) {
	return "", ErrNotSupportedByFsBackend
}

func (f *Fs) GetSessionFileReader(ctx context.Context, asset AssetOptions) (io.ReadCloser, int64, error) {
	path := f.sessionFilePath(asset.RecorderID, asset.SessionID, asset.Filename)
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot open file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("cannot stat file: %w", err)
	}
	return file, info.Size(), nil
}

func (f *Fs) GetSegmentFileReader(ctx context.Context, asset SegmentAssetOptions) (io.ReadCloser, int64, error) {
	path := f.segmentFilePath(asset.RecorderID, asset.SessionID, asset.SegmentID, asset.Filename)
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot open file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("cannot stat file: %w", err)
	}
	return file, info.Size(), nil
}

// -----------------------------------------------------------------------------
// Segments
// -----------------------------------------------------------------------------

func (f *Fs) CreateSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, segment Segment) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}
	session, ok := f.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}
	if session.Segments == nil {
		session.Segments = make(map[uuid.UUID]Segment)
	}
	segment.ID = segmentID
	session.Segments[segmentID] = segment

	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Created segment")
	return nil
}

func (f *Fs) UpdateSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, segment Segment) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}
	session, ok := f.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}
	existingSegment, ok := session.Segments[segmentID]
	if !ok {
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	segment.ID = segmentID
	timeChanged := segment.StartPoint != existingSegment.StartPoint || segment.EndPoint != existingSegment.EndPoint
	wasRendered := existingSegment.State == SegmentStateFinished

	if timeChanged && wasRendered {
		if err := os.RemoveAll(f.segmentDir(recorderID, sessionID, segmentID)); err != nil {
			log.Warn().Err(err).Msg("Cannot remove segment files")
		}
		segment.State = SegmentStateUnknown
		segment.ErrorMessage = ""
		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Stringer("segment-id", segmentID).
			Msg("Segment times changed, removed rendered files and reset state")
	} else if segment.State == SegmentStateUnknown {
		segment.State = existingSegment.State
	}

	session.Segments[segmentID] = segment
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}

	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Updated segment")
	return nil
}

func (f *Fs) DeleteSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}
	session, ok := f.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}
	if _, ok := session.Segments[segmentID]; !ok {
		return fmt.Errorf("no segment with id %s", segmentID)
	}

	delete(session.Segments, segmentID)
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	if err := os.RemoveAll(f.segmentDir(recorderID, sessionID, segmentID)); err != nil {
		log.Warn().Err(err).Msg("Cannot remove segment directory")
	}
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Deleted segment")
	return nil
}

func (f *Fs) SetSegmentState(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, state SegmentState) error {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return fmt.Errorf("no recorder with id %s", recorderID)
	}
	session, ok := f.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		return fmt.Errorf("no session with id %s", sessionID)
	}
	segment, ok := session.Segments[segmentID]
	if !ok {
		return fmt.Errorf("no segment with id %s", segmentID)
	}
	segment.State = state
	session.Segments[segmentID] = segment

	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	return nil
}

func (f *Fs) RenderSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error {
	f.dataLock.Lock()
	if _, ok := f.system.Recorders[recorderID]; !ok {
		f.dataLock.Unlock()
		return fmt.Errorf("no recorder with id %s", recorderID)
	}
	session, ok := f.system.Recorders[recorderID].Sessions[sessionID]
	if !ok {
		f.dataLock.Unlock()
		return fmt.Errorf("no session with id %s", sessionID)
	}
	segment, ok := session.Segments[segmentID]
	if !ok {
		f.dataLock.Unlock()
		return fmt.Errorf("no segment with id %s", segmentID)
	}
	segment.State = SegmentStateRendering
	segment.ErrorMessage = ""
	session.Segments[segmentID] = segment
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		f.dataLock.Unlock()
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	f.dataLock.Unlock()

	f.notifyStateChange(&session, session.State)

	if segment.EndPoint <= segment.StartPoint {
		errMsg := fmt.Sprintf("invalid segment range: start=%d end=%d", segment.StartPoint, segment.EndPoint)
		f.setSegmentError(ctx, recorderID, sessionID, segmentID, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	rawPath := f.sessionFilePath(recorderID, sessionID, FILENAME_RAW)
	rawData, err := os.Open(rawPath)
	if err != nil {
		f.setSegmentError(ctx, recorderID, sessionID, segmentID, fmt.Sprintf("cannot open raw audio: %v", err))
		return fmt.Errorf("cannot open raw audio: %w", err)
	}
	defer rawData.Close()

	// Seek to the segment start and bound the window so we never read the whole
	// file to extract a short clip. Raw is s16le/2ch => 4 bytes per frame;
	// StartPoint/EndPoint are frame indices.
	const rawBytesPerFrame = 2 * 2
	startByte := segment.StartPoint * rawBytesPerFrame
	want := (segment.EndPoint - segment.StartPoint) * rawBytesPerFrame
	if _, err := rawData.Seek(startByte, io.SeekStart); err != nil {
		f.setSegmentError(ctx, recorderID, sessionID, segmentID, fmt.Sprintf("cannot seek raw audio: %v", err))
		return fmt.Errorf("cannot seek raw audio: %w", err)
	}
	segmentRaw := io.LimitReader(rawData, want)

	readers, writer, closer := makeReaders(2)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer closer.Close()
		_, err := io.Copy(writer, segmentRaw)
		if err != nil && !isPipeClosed(err) {
			return err
		}
		return nil
	})

	oggReader := readers[0]
	flacReader := readers[1]

	eg.Go(func() error {
		defer closeReader(oggReader)
		return f.writeFileStream(f.segmentFilePath(recorderID, sessionID, segmentID, SEGMENT_FILENAME_OGG), func(w io.Writer) error {
			return render.Opus(w, oggReader)
		})
	})

	eg.Go(func() error {
		defer closeReader(flacReader)
		return f.writeFileStream(f.segmentFilePath(recorderID, sessionID, segmentID, SEGMENT_FILENAME_FLAC), func(w io.Writer) error {
			return render.Flac(w, flacReader)
		})
	})

	if err := eg.Wait(); err != nil {
		f.setSegmentError(ctx, recorderID, sessionID, segmentID, err.Error())
		return err
	}

	f.dataLock.Lock()
	session = f.system.Recorders[recorderID].Sessions[sessionID]
	segment = session.Segments[segmentID]
	segment.State = SegmentStateFinished
	segment.ErrorMessage = ""
	session.Segments[segmentID] = segment
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		f.dataLock.Unlock()
		return fmt.Errorf("cannot update session metadata: %w", err)
	}
	f.dataLock.Unlock()

	f.notifyStateChange(&session, session.State)
	log.Info().
		Stringer("recorder-id", recorderID).
		Stringer("session-id", sessionID).
		Stringer("segment-id", segmentID).
		Msg("Segment rendering complete")
	return nil
}

func (f *Fs) setSegmentError(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, errorMsg string) {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()

	if _, ok := f.system.Recorders[recorderID]; !ok {
		return
	}
	session, ok := f.system.Recorders[recorderID].Sessions[sessionID]
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
	if err := f.putSessionMetadata(recorderID, sessionID, &session); err != nil {
		log.Err(err).Msg("Cannot persist segment error")
	}
}

// -----------------------------------------------------------------------------
// Metadata IO
// -----------------------------------------------------------------------------

func (f *Fs) findRecorderIDs() ([]uuid.UUID, error) {
	entries, err := os.ReadDir(f.root)
	if err != nil {
		return nil, fmt.Errorf("cannot read root dir: %w", err)
	}
	var ids []uuid.UUID
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, err := uuid.Parse(entry.Name())
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (f *Fs) readSessionIDs(recorderID uuid.UUID) ([]uuid.UUID, error) {
	dir := filepath.Join(f.recorderDir(recorderID), "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read sessions dir: %w", err)
	}
	var ids []uuid.UUID
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, err := uuid.Parse(entry.Name())
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (f *Fs) getSystemMetadata() (*System, error) {
	data, err := os.ReadFile(f.systemMetadataPath())
	if err != nil {
		return nil, fmt.Errorf("cannot read system metadata: %w", err)
	}
	var sys System
	if err := json.Unmarshal(data, &sys); err != nil {
		return nil, fmt.Errorf("cannot unmarshal system metadata: %w", err)
	}
	return &sys, nil
}

func (f *Fs) putSystemMetadata(system *System) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(system); err != nil {
		return fmt.Errorf("cannot marshal system metadata: %w", err)
	}
	if err := f.writeFile(f.systemMetadataPath(), buf.Bytes()); err != nil {
		return err
	}
	f.system = system
	return nil
}

func (f *Fs) getRecorderMetadata(recorderID uuid.UUID) (*Recorder, error) {
	data, err := os.ReadFile(f.recorderMetadataPath(recorderID))
	if err != nil {
		return nil, fmt.Errorf("cannot read recorder metadata: %w", err)
	}
	var rec Recorder
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("cannot unmarshal recorder metadata: %w", err)
	}
	return &rec, nil
}

func (f *Fs) putRecorderMetadata(recorderID uuid.UUID, recorder *Recorder) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(recorder); err != nil {
		return fmt.Errorf("cannot marshal recorder metadata: %w", err)
	}
	if err := f.writeFile(f.recorderMetadataPath(recorderID), buf.Bytes()); err != nil {
		return err
	}
	if f.system.Recorders == nil {
		f.system.Recorders = make(map[uuid.UUID]Recorder)
	}
	f.system.Recorders[recorderID] = *recorder
	return nil
}

func (f *Fs) getSessionMetadata(recorderID, sessionID uuid.UUID) (*Session, error) {
	data, err := os.ReadFile(f.sessionMetadataPath(recorderID, sessionID))
	if err != nil {
		return nil, fmt.Errorf("cannot read session metadata: %w", err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("cannot unmarshal session metadata: %w", err)
	}
	return &session, nil
}

func (f *Fs) putSessionMetadata(recorderID, sessionID uuid.UUID, session *Session) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(session); err != nil {
		return fmt.Errorf("cannot marshal session metadata: %w", err)
	}
	if err := f.writeFile(f.sessionMetadataPath(recorderID, sessionID), buf.Bytes()); err != nil {
		return err
	}

	if f.system.Recorders == nil {
		f.system.Recorders = make(map[uuid.UUID]Recorder)
	}
	recorder, ok := f.system.Recorders[recorderID]
	if !ok {
		recorder = Recorder{ID: recorderID, Sessions: make(map[uuid.UUID]Session)}
	}
	if recorder.Sessions == nil {
		recorder.Sessions = make(map[uuid.UUID]Session)
	}
	recorder.Sessions[sessionID] = *session
	f.system.Recorders[recorderID] = recorder
	return nil
}
