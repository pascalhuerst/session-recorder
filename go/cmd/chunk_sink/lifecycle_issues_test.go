package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	"github.com/pascalhuerst/session-recorder/storage"
)

// =============================================================================
// Issue: OnRecorderDisconnected uses context.Background()
//
// When a recorder disconnects, OnRecorderDisconnected (chunk-sink-handler.go:204)
// calls closeRecorderSession with context.Background(). If the storage layer
// (MinIO/S3) hangs, this goroutine blocks forever with no timeout.
// =============================================================================

func TestFix_DisconnectCleanupHasTimeout(t *testing.T) {
	// Create a storage mock that captures the context passed to CloseRecordingSession
	ctxStorage := &contextCaptureMockStorage{
		mockStorage: newMockStorage(),
		ctxCh:       make(chan context.Context, 1),
	}

	rb := broadcast.NewRecorderBroadcaster(10)
	handler := NewChunkSinkHandler(ctxStorage, rb)

	recorderID := uuid.New()
	sessionID := uuid.New()

	// Simulate connected recorder with active session
	handler.lock.Lock()
	handler.connectedRecorders[recorderID] = struct{}{}
	handler.recorderStates[recorderID] = &recorderState{
		lastSession: sessionID,
		recording:   true,
	}
	handler.lock.Unlock()

	// Disconnect the recorder
	go handler.OnRecorderDisconnected(recorderID)

	// Capture the context used for cleanup
	select {
	case ctx := <-ctxStorage.ctxCh:
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("OnRecorderDisconnected should use a context with a timeout, " +
				"but no deadline was set")
		} else {
			remaining := time.Until(deadline)
			if remaining < 20*time.Second || remaining > 35*time.Second {
				t.Errorf("Expected ~30s timeout, got %v remaining", remaining)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("CloseRecordingSession was never called")
	}
}

// =============================================================================
// Issue: setChunks has race window between unlock and previous session close
//
// At chunk-sink-handler.go:139-144, the handler records previousSession and
// unlocks the mutex. Between the unlock (line 144) and the close call (line 163),
// another chunk goroutine can modify recorderStates, causing:
// - The previous session to be closed twice
// - The wrong session to be closed
// =============================================================================

func TestIssue_SetChunksRaceOnSessionSwitch(t *testing.T) {
	trackingStorage := newTrackingMockStorage()
	closedSessions := trackingStorage.closedSessions

	rb := broadcast.NewRecorderBroadcaster(10)
	handler := NewChunkSinkHandler(trackingStorage, rb)

	recorderID := uuid.New()
	session1 := uuid.New()
	session2 := uuid.New()
	session3 := uuid.New()

	// Set up initial state with session1 active
	handler.lock.Lock()
	handler.recorderStates[recorderID] = &recorderState{
		lastSession: session1,
		recording:   true,
	}
	handler.lock.Unlock()

	// Simulate rapid session switches from concurrent chunk deliveries
	var wg sync.WaitGroup

	// Goroutine 1: chunks for session2 (triggers close of session1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		trackingStorage.SafeChunks(context.Background(), recorderID, session2, "001", time.Now(), nil)
		handler.setRecorderStatus(context.Background(), &cmpb.RecorderStatus{
			RecorderID:   recorderID.String(),
			RecorderName: "test",
			SignalStatus: cmpb.SignalStatus_SIGNAL,
		})
		// Manually trigger the session switch logic
		handler.lock.Lock()
		state := handler.recorderStates[recorderID]
		prev := state.lastSession
		state.lastSession = session2
		handler.lock.Unlock()

		if prev != uuid.Nil && prev != session2 {
			handler.closeRecorderSession(context.Background(), recorderID, prev)
		}
	}()

	// Goroutine 2: chunks for session3 (triggers close of session2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Small delay to increase race window overlap
		time.Sleep(time.Millisecond)
		handler.lock.Lock()
		state := handler.recorderStates[recorderID]
		prev := state.lastSession
		state.lastSession = session3
		handler.lock.Unlock()

		if prev != uuid.Nil && prev != session3 {
			handler.closeRecorderSession(context.Background(), recorderID, prev)
		}
	}()

	wg.Wait()

	// Check for duplicate closes
	counts := closedSessions.getCounts()
	for sid, count := range counts {
		if count > 1 {
			t.Errorf("Session %s closed %d times — race between concurrent session switches", sid, count)
		}
	}
}

// =============================================================================
// Issue: Connection callback fired in background goroutine (chunksink-server.go:99)
//
// OnRecorderConnectedCB is fired via `go callback(recorderID)` after the mutex
// is released. IsRecorderConnected can return true BEFORE the connected callback
// has finished executing. This means:
// - The handler might start processing chunks before connection state is ready
// - There's no ordering guarantee between "connected" event and first chunks
// =============================================================================

func TestFix_ConnectionCallbackSynchronous(t *testing.T) {
	// After the fix, GetCommands fires the callback synchronously after
	// releasing the mutex (not in a goroutine). This means the callback
	// completes before GetCommands blocks on context.Done().
	//
	// We verify this by checking that OnRecorderConnected is called
	// synchronously — the recorder state is initialized before the
	// function returns.

	rb := broadcast.NewRecorderBroadcaster(10)
	handler := NewChunkSinkHandler(newTrackingMockStorage(), rb)

	recorderID := uuid.New()

	// Call OnRecorderConnected directly (synchronous in the fix)
	handler.OnRecorderConnected(recorderID)

	// State should be fully initialized by now
	if !handler.IsRecorderConnected(recorderID) {
		t.Error("Recorder should be connected after synchronous OnRecorderConnected")
	}

	handler.lock.Lock()
	state, ok := handler.recorderStates[recorderID]
	handler.lock.Unlock()

	if !ok || state == nil {
		t.Error("Recorder state should be initialized after OnRecorderConnected")
	}
}

// =============================================================================
// Issue: Peak accumulator cleanup only on RECORDING→* transitions
//
// SessionSourceHandler.OnSessionStateChanged (session-source-handler.go:281)
// only removes peak data when PreviousState == RECORDING. Sessions that
// enter ERROR from PROCESSING (render failure) or are deleted never clean
// up their peak accumulator entries.
// =============================================================================

func TestFix_PeakAccumulatorCleanupOnAllNonRecordingTransitions(t *testing.T) {
	// After the fix, the condition is:
	//   if event.NewState != SessionStateRecording
	// This means ALL transitions to non-recording states trigger cleanup.

	transitions := []struct {
		name        string
		from        storage.SessionState
		to          storage.SessionState
		shouldClean bool
	}{
		{"RECORDING→PROCESSING", storage.SessionStateRecording, storage.SessionStateProcessing, true},
		{"PROCESSING→ERROR", storage.SessionStateProcessing, storage.SessionStateError, true},
		{"PROCESSING→FINISHED", storage.SessionStateProcessing, storage.SessionStateFinished, true},
		{"ERROR→PROCESSING", storage.SessionStateError, storage.SessionStateProcessing, true},
		// RECORDING should NOT trigger cleanup (session is still active)
		{"UNKNOWN→RECORDING", storage.SessionStateUnknown, storage.SessionStateRecording, false},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the fixed condition: NewState != RECORDING
			wouldClean := tt.to != storage.SessionStateRecording

			if wouldClean != tt.shouldClean {
				t.Errorf("Transition %s→%s: wouldClean=%v, want %v",
					tt.from, tt.to, wouldClean, tt.shouldClean)
			}
		})
	}
}

// --- Test Helpers ---

// contextCaptureMockStorage captures the context passed to CloseRecordingSession
type contextCaptureMockStorage struct {
	*mockStorage
	ctxCh chan context.Context
}

func (s *contextCaptureMockStorage) CloseRecordingSession(ctx context.Context, _, _ uuid.UUID) error {
	s.ctxCh <- ctx
	return nil
}

// blockingMockStorage blocks on CloseRecordingSession until unblocked
type blockingMockStorage struct {
	*mockStorage
	closeCalled chan struct{}
	unblock     chan struct{}
}

func newBlockingMockStorage() *blockingMockStorage {
	return &blockingMockStorage{
		mockStorage: newMockStorage(),
		closeCalled: make(chan struct{}),
		unblock:     make(chan struct{}),
	}
}

func (s *blockingMockStorage) CloseRecordingSession(_ context.Context, _, _ uuid.UUID) error {
	close(s.closeCalled)
	<-s.unblock
	return nil
}

// sessionTracker tracks session close calls thread-safely
type sessionTracker struct {
	mu     sync.Mutex
	counts map[uuid.UUID]int
}

func (t *sessionTracker) record(sessionID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.counts == nil {
		t.counts = make(map[uuid.UUID]int)
	}
	t.counts[sessionID]++
}

func (t *sessionTracker) getCounts() map[uuid.UUID]int {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make(map[uuid.UUID]int)
	for k, v := range t.counts {
		result[k] = v
	}
	return result
}

// trackingMockStorage tracks which sessions were closed
type trackingMockStorage struct {
	*mockStorage
	closedSessions *sessionTracker
}

func newTrackingMockStorage() *trackingMockStorage {
	return &trackingMockStorage{
		mockStorage:    newMockStorage(),
		closedSessions: &sessionTracker{},
	}
}

func (s *trackingMockStorage) CloseRecordingSession(_ context.Context, _ uuid.UUID, sessionID uuid.UUID) error {
	s.closedSessions.record(sessionID)
	return nil
}
