package storage

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

/**
 * Test Plan: a closed session stays closed
 *
 * Scenario: Which chunks may start, resume or touch a session
 *   Given the result of reading a session's stored metadata
 *   When admitChunkForSession() decides
 *   Then only a missing session is started, only a RECORDING one resumed,
 *     a stored session in any other state is refused with ErrSessionClosed,
 *     and an unreadable one is refused without guessing
 *
 * Scenario: A chunk arriving after the close is dropped (filesystem backend)
 *   Given a session with audio that has been closed
 *   When another chunk for the same session arrives
 *   Then SafeChunks returns ErrSessionClosed
 *   And data.raw still holds exactly the audio from before the close
 *   And the session keeps its start time and reaches FINISHED
 *
 * Scenario: Two closes of the same session at once
 *   Given a session with audio
 *   When CloseRecordingSession is called twice concurrently
 *   Then the session reaches FINISHED once and data.raw is intact
 */

func TestAdmitChunkForSession(t *testing.T) {
	readError := errors.New("connection reset")
	cases := map[string]struct {
		sm          *Session
		err         error
		notFound    bool
		wantStart   bool
		wantResume  bool
		wantRejects error // nil: admitted; ErrSessionClosed; or readError (wrapped)
	}{
		"nothing stored yet":        {err: os.ErrNotExist, notFound: true, wantStart: true},
		"stored and RECORDING":      {sm: &Session{State: SessionStateRecording}, wantResume: true},
		"stored and PROCESSING":     {sm: &Session{State: SessionStateProcessing}, wantRejects: ErrSessionClosed},
		"stored and FINISHED":       {sm: &Session{State: SessionStateFinished}, wantRejects: ErrSessionClosed},
		"stored and ERROR":          {sm: &Session{State: SessionStateError}, wantRejects: ErrSessionClosed},
		"metadata cannot be read":   {err: readError, wantRejects: readError},
		"unknown state when stored": {sm: &Session{State: SessionStateUnknown}, wantRejects: ErrSessionClosed},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			start, resume, reject := admitChunkForSession(tc.sm, tc.err, tc.notFound)
			if start != tc.wantStart || resume != tc.wantResume {
				t.Errorf("start=%v resume=%v, want start=%v resume=%v", start, resume, tc.wantStart, tc.wantResume)
			}
			if tc.wantRejects == nil && reject != nil {
				t.Errorf("rejected with %v, want admitted", reject)
			}
			if tc.wantRejects != nil && !errors.Is(reject, tc.wantRejects) {
				t.Errorf("reject = %v, want %v", reject, tc.wantRejects)
			}
			if reject != nil && (start || resume) {
				t.Error("a rejected chunk must neither start nor resume a session")
			}
		})
	}
}

// newTestFs returns a started filesystem storage with one recorder. Closing a
// session there renders it, Opus included, and kazzmir/opus-go (transpiled
// libopus) trips checkptr under -race - same skip as in the render package.
func newTestFs(t *testing.T) (*Fs, uuid.UUID) {
	t.Helper()
	if raceEnabled {
		t.Skip("skipped under -race: rendering a session builds an opus-go encoder, which trips checkptr")
	}
	fs, err := NewFsStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFsStorage: %v", err)
	}
	if err := fs.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(fs.Stop)
	recorderID := uuid.New()
	fs.EnsureRecorderExists(context.Background(), recorderID, "test recorder")
	return fs, recorderID
}

// tone returns n stereo frames of a non-silent, non-constant signal.
func tone(n int) []int16 {
	samples := make([]int16, n*2)
	for i := range samples {
		samples[i] = int16((i*37)%2000 - 1000)
	}
	return samples
}

func waitForState(t *testing.T, fs *Fs, recorderID, sessionID uuid.UUID, want SessionState) Session {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		session, err := fs.getSessionMetadata(recorderID, sessionID)
		if err == nil && session.State == want {
			return *session
		}
		if time.Now().After(deadline) {
			t.Fatalf("session did not reach %v (last: %+v, err %v)", want, session, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func rawSize(t *testing.T, fs *Fs, recorderID, sessionID uuid.UUID) int64 {
	t.Helper()
	info, err := os.Stat(fs.sessionFilePath(recorderID, sessionID, FILENAME_RAW))
	if err != nil {
		t.Fatalf("stat data.raw: %v", err)
	}
	return info.Size()
}

func TestFsChunkAfterCloseIsDropped(t *testing.T) {
	ctx := context.Background()
	fs, recorderID := newTestFs(t)
	sessionID := uuid.New()
	started := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	const frames = 48000
	for i := range 3 {
		if err := fs.SafeChunks(ctx, recorderID, sessionID, "", started.Add(time.Duration(i)*time.Second), tone(frames)); err != nil {
			t.Fatalf("SafeChunks %d: %v", i, err)
		}
	}
	wantBytes := int64(3 * frames * 2 * 2)

	if err := fs.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession: %v", err)
	}

	// The recorder's last chunk, overtaken by the status that closed the session.
	late := started.Add(time.Minute)
	err := fs.SafeChunks(ctx, recorderID, sessionID, "", late, tone(frames))
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("SafeChunks after close = %v, want ErrSessionClosed", err)
	}

	session := waitForState(t, fs, recorderID, sessionID, SessionStateFinished)
	if got := rawSize(t, fs, recorderID, sessionID); got != wantBytes {
		t.Errorf("data.raw is %d bytes after the late chunk, want the %d from before the close", got, wantBytes)
	}
	if !session.StartTime.Equal(started) {
		t.Errorf("start time = %v, want the first chunk's %v (the late chunk must not restart the session)", session.StartTime, started)
	}

	// And once rendered it stays refused.
	if err := fs.SafeChunks(ctx, recorderID, sessionID, "", late, tone(frames)); !errors.Is(err, ErrSessionClosed) {
		t.Errorf("SafeChunks on a finished session = %v, want ErrSessionClosed", err)
	}
	if got := rawSize(t, fs, recorderID, sessionID); got != wantBytes {
		t.Errorf("data.raw changed to %d bytes on a finished session", got)
	}
}

func TestFsLateChunkDoesNotEndTheCurrentSession(t *testing.T) {
	ctx := context.Background()
	fs, recorderID := newTestFs(t)
	oldSession, currentSession := uuid.New(), uuid.New()
	now := time.Now()

	if err := fs.SafeChunks(ctx, recorderID, oldSession, "", now, tone(4800)); err != nil {
		t.Fatalf("SafeChunks old: %v", err)
	}
	if err := fs.CloseRecordingSession(ctx, recorderID, oldSession); err != nil {
		t.Fatalf("CloseRecordingSession: %v", err)
	}
	waitForState(t, fs, recorderID, oldSession, SessionStateFinished)

	if err := fs.SafeChunks(ctx, recorderID, currentSession, "", now, tone(4800)); err != nil {
		t.Fatalf("SafeChunks current: %v", err)
	}
	// A straggler for the finished session while the next one is recording.
	if err := fs.SafeChunks(ctx, recorderID, oldSession, "", now, tone(4800)); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("straggler = %v, want ErrSessionClosed", err)
	}
	if err := fs.SafeChunks(ctx, recorderID, currentSession, "", now, tone(4800)); err != nil {
		t.Fatalf("current session no longer accepts chunks after the straggler: %v", err)
	}
	current, err := fs.getSessionMetadata(recorderID, currentSession)
	if err != nil || current.State != SessionStateRecording {
		t.Errorf("current session state = %+v (err %v), want RECORDING", current, err)
	}
	if got, want := rawSize(t, fs, recorderID, currentSession), int64(2*4800*2*2); got != want {
		t.Errorf("current session's data.raw is %d bytes, want %d", got, want)
	}
}

func TestFsConcurrentClosesRenderOnce(t *testing.T) {
	ctx := context.Background()
	fs, recorderID := newTestFs(t)
	sessionID := uuid.New()
	const frames = 48000
	if err := fs.SafeChunks(ctx, recorderID, sessionID, "", time.Now(), tone(frames)); err != nil {
		t.Fatalf("SafeChunks: %v", err)
	}

	var finished int
	var mu sync.Mutex
	fs.RegisterOnSessionStateChangedCallback(func(session *Session, _ SessionState) {
		if session.ID == sessionID && session.State == SessionStateFinished {
			mu.Lock()
			finished++
			mu.Unlock()
		}
	})

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fs.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
				t.Errorf("CloseRecordingSession: %v", err)
			}
		}()
	}
	wg.Wait()

	waitForState(t, fs, recorderID, sessionID, SessionStateFinished)
	time.Sleep(200 * time.Millisecond) // room for a second render to show up
	mu.Lock()
	defer mu.Unlock()
	if finished != 1 {
		t.Errorf("session reached FINISHED %d times, want once", finished)
	}
	if got, want := rawSize(t, fs, recorderID, sessionID), int64(frames*2*2); got != want {
		t.Errorf("data.raw is %d bytes, want %d", got, want)
	}
}
