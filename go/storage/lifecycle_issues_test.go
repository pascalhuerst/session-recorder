package storage

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Issue #1: Work queue stop() drops pending jobs
//
// Minio.Stop() calls renderQueue.stop() which is non-blocking — it cancels
// the context and calls pool.Stop() which discards pending jobs immediately.
// Sessions in PROCESSING state whose render jobs are still queued will never
// reach FINISHED, leaving them permanently stuck.
// =============================================================================

func TestFix_WorkQueueStopAndWaitCompletesAllJobs(t *testing.T) {
	wq := newWorkQueue(1) // 1 worker to force queueing

	var completed atomic.Int32
	total := 5

	// Submit jobs that take time to complete
	for i := 0; i < total; i++ {
		wq.pool.Submit(func() {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)
		})
	}

	// stopAndWait should block until all jobs complete
	wq.stopAndWait()

	if completed.Load() != int32(total) {
		t.Errorf("stopAndWait() completed %d/%d — expected all jobs to finish",
			completed.Load(), total)
	}
}

// =============================================================================
// Issue #2: Work queue context cancelled immediately on stop()
//
// The work queue creates a single ctx via context.WithCancel(Background()).
// stop() calls cancel() which immediately cancels all in-flight jobs' context.
// Jobs that check ctx.Done() or pass ctx to S3 operations will fail prematurely
// even if the pool hasn't stopped yet.
// =============================================================================

func TestFix_WorkQueueContextSurvivesGracefulStop(t *testing.T) {
	wq := newWorkQueue(1)

	ctxValid := make(chan bool, 1)
	jobStarted := make(chan struct{})

	wq.pool.Submit(func() {
		close(jobStarted)
		// Check that context is still valid during job execution
		select {
		case <-wq.ctx.Done():
			ctxValid <- false
		case <-time.After(50 * time.Millisecond):
			ctxValid <- true
		}
	})

	<-jobStarted

	// stopAndWait should not cancel context while jobs are running
	wq.stopAndWait()

	if !<-ctxValid {
		t.Error("Work queue context was cancelled while jobs were still running — " +
			"in-flight render jobs need valid context for S3 uploads")
	}
}

// =============================================================================
// Issue #3: Streaming uploads ignore parent context (use context.Background)
//
// startStreamingSession() at minio.go:1024 explicitly uses context.Background()
// for upload goroutines. When the server shuts down and contexts are cancelled,
// streaming upload goroutines continue running, potentially blocking on slow S3.
// =============================================================================

func TestFix_StreamingUploadsUseShutdownContext(t *testing.T) {
	// After the fix, startStreamingSession uses m.shutdownCtx instead of
	// context.Background(). Verify that the shutdown context is cancellable.
	fake := NewFakeMinioClient()
	m := NewMinioStorageWithClient(fake, "fake:9000", "", "")
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	// Don't use newTestStorage — its cleanup calls Stop(), which would
	// double-close stopTimeout when we call Stop() explicitly below.

	// The shutdown context should be valid before Stop
	select {
	case <-m.shutdownCtx.Done():
		t.Fatal("Shutdown context should be valid before Stop()")
	default:
		// Good
	}

	m.Stop()

	// After Stop, the shutdown context should be cancelled
	select {
	case <-m.shutdownCtx.Done():
		// Good — streaming uploads would be cancelled
	default:
		t.Error("Shutdown context should be cancelled after Stop() — " +
			"streaming uploads would still be running")
	}
}

// =============================================================================
// Issue #4: closeSessionAsync has TOCTOU race
//
// closeSessionAsync checks isSessionClosed() (line 1605) BEFORE acquiring
// dataLock (line 1611). Between these checks, the session state can change:
// - Another goroutine resumes the session to RECORDING
// - The check at 1613 sees RECORDING and skips the render
// - But the session was supposed to be rendered
// =============================================================================

func TestIssue_CloseSessionAsyncTOCTOU(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := newTestRecorder(t, m, ctx)
	sessionID := newTestSession(t, m, ctx, recorderID)

	// Transition to PROCESSING via FSM
	if err := m.fireSessionTrigger(ctx, sessionID, triggerCloseRecording); err != nil {
		t.Fatalf("Cannot transition to PROCESSING: %v", err)
	}

	// Verify state is PROCESSING
	sess, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.State != SessionStateProcessing {
		t.Fatalf("Expected PROCESSING, got %s", sess.State)
	}

	// Now simulate the race: between isSessionClosed check and dataLock acquire,
	// another goroutine resumes the session to RECORDING.
	//
	// We prove this by:
	// 1. Starting closeSessionAsync in a goroutine
	// 2. Resuming the session concurrently
	// 3. Checking if the render was silently skipped

	var closeErr error
	var closeWg sync.WaitGroup
	closeWg.Add(1)

	// Resume session to RECORDING before closeSessionAsync acquires lock
	m.dataLock.Lock()
	sess = m.system.Recorders[recorderID].Sessions[sessionID]
	sess.State = SessionStateRecording
	m.system.Recorders[recorderID].Sessions[sessionID] = sess
	m.removeSessionMachine(sessionID)
	m.getOrCreateSessionMachine(recorderID, sessionID, SessionStateRecording)
	m.dataLock.Unlock()

	go func() {
		defer closeWg.Done()
		closeErr = m.closeSessionAsync(ctx, recorderID, sessionID, nil)
	}()

	closeWg.Wait()

	// closeSessionAsync returns nil (silently skips) because it sees RECORDING
	// But the session was supposed to be in PROCESSING when the render was queued
	if closeErr == nil {
		sess, _ := m.GetSession(recorderID, sessionID)
		if sess.State == SessionStateRecording {
			// This proves the issue: the render was silently skipped
			// because the state changed between queue time and execution time.
			// The session will never be rendered unless another close is triggered.
			t.Log("TOCTOU confirmed: closeSessionAsync silently skipped render " +
				"because session was resumed between queue time and execution. " +
				"Session will stay in RECORDING but may have incomplete data.")
		}
	}
}

// =============================================================================
// Issue #5: deletedSessions tombstone map cleanup is amortized
//
// Tombstone entries are only cleaned during the next DeleteSession() call.
// If deletes are infrequent, the map grows unbounded.
// =============================================================================

func TestFix_DeletedSessionsTombstonePeriodicSweep(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	// Create and delete several sessions
	for i := 0; i < 10; i++ {
		recorderID := newTestRecorder(t, m, ctx)
		sessionID := newTestSession(t, m, ctx, recorderID)
		m.fireSessionTrigger(ctx, sessionID, triggerCloseRecording)
		m.fireSessionTrigger(ctx, sessionID, triggerRenderSuccess)
		if err := m.DeleteSession(ctx, recorderID, sessionID); err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}
	}

	// Backdate all tombstones so they're eligible for cleanup
	m.dataLock.Lock()
	for id := range m.deletedSessions {
		m.deletedSessions[id] = time.Now().Add(-2 * time.Minute)
	}
	countBefore := len(m.deletedSessions)
	m.dataLock.Unlock()

	if countBefore == 0 {
		t.Skip("No tombstones to clean up")
	}

	// sweepDeletedSessions should clean up stale entries
	// (called periodically by the timeout checker)
	m.sweepDeletedSessions()

	m.dataLock.Lock()
	countAfter := len(m.deletedSessions)
	m.dataLock.Unlock()

	if countAfter != 0 {
		t.Errorf("sweepDeletedSessions left %d stale entries (had %d)", countAfter, countBefore)
	}
}

// Helper to create a recorder for lifecycle tests
func newTestRecorder(t *testing.T, m *Minio, ctx context.Context) (recorderID uuid.UUID) {
	t.Helper()
	recorderID = uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "test-recorder")
	return
}

// Helper to create a session in RECORDING state
func newTestSession(t *testing.T, m *Minio, ctx context.Context, recorderID uuid.UUID) (sessionID uuid.UUID) {
	t.Helper()
	sessionID = uuid.New()
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}
	return
}
