package storage

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Issue #1: deletedSessions map grows unbounded
//
// Every DeleteSession call adds to the map, but nothing ever removes entries.
// Over a long-running server this is a memory leak.
// =============================================================================

func TestBug_DeletedSessionsMapGrowsUnbounded(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Phase 1: Create and delete 50 sessions with old timestamps
	for i := 0; i < 50; i++ {
		sessionID := uuid.New()
		samples := make([]int16, 100)
		if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
			t.Fatalf("SafeChunks failed: %v", err)
		}
		if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
			t.Fatalf("CloseRecordingSession failed: %v", err)
		}
		if err := m.DeleteSession(ctx, recorderID, sessionID); err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}
	}

	// Backdate all tombstones to 2 minutes ago so the sweep will clean them
	m.dataLock.Lock()
	old := time.Now().Add(-2 * time.Minute)
	for id := range m.deletedSessions {
		m.deletedSessions[id] = old
	}
	m.dataLock.Unlock()

	// Phase 2: One more delete triggers the amortized sweep
	sessionID := uuid.New()
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}
	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}
	if err := m.DeleteSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	m.dataLock.Lock()
	mapSize := len(m.deletedSessions)
	m.dataLock.Unlock()

	// After the fix, old tombstones should have been swept. Only the most recent
	// delete (+ any that were too fresh) should remain.
	if mapSize > 5 {
		t.Errorf("deletedSessions map has %d entries after sweep — old tombstones not cleaned up (memory leak)", mapSize)
	}
}

// =============================================================================
// Issue #2: SafeChunks resume races with in-flight render
//
// When a recorder sends chunks for a session in PROCESSING state, SafeChunks
// resets it to RECORDING and replaces the FSM. But a render worker may still
// hold a reference to the old SM. When the render completes, it fires
// triggerRenderSuccess/Failure via onSessionTransition, which overwrites the
// RECORDING state back to FINISHED/ERROR.
// =============================================================================

func TestBug_ResumeRacesWithInflightRender(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session and move to PROCESSING via CloseRecordingSession
	// (which removes the chunk entry, simulating a real close)
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Close properly — removes chunk from map and transitions to PROCESSING
	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Simulate a render worker grabbing the SM reference before the resume
	m.machineLock.Lock()
	oldSM, ok := m.sessionMachines[sessionID]
	m.machineLock.Unlock()
	if !ok {
		t.Fatal("State machine should exist")
	}

	// Now the recorder resumes sending chunks — SafeChunks sees the session
	// in PROCESSING and resets to RECORDING (no chunk in m.chunks for this recorder)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "002", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks (resume) failed: %v", err)
	}

	// Verify session is back to RECORDING
	s, _ := m.GetSession(recorderID, sessionID)
	if s.State != SessionStateRecording {
		t.Fatalf("Expected RECORDING after resume, got %s", s.State)
	}

	// Now the old render worker completes and fires on the stale SM reference.
	// This should NOT overwrite the RECORDING state.
	_ = oldSM.FireCtx(ctx, triggerRenderSuccess)

	// Check: session should still be RECORDING, not FINISHED
	s, _ = m.GetSession(recorderID, sessionID)
	if s.State != SessionStateRecording {
		t.Errorf("Session state was overwritten to %s by stale render — expected RECORDING", s.State)
	}
}

// =============================================================================
// Issue #3: initSession emits event under dataLock
//
// All other emission sites release dataLock before emitting. initSession does
// not, which means any listener that calls GetSession/GetSessions will deadlock.
// =============================================================================

func TestBug_InitSessionEmitsEventUnderDataLock(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Register a listener that tries to read session data (which needs dataLock)
	var listenerSawEvent atomic.Bool
	var listenerDeadlocked atomic.Bool
	listenerDone := make(chan struct{})

	m.EventBus().AddListener(&testSessionListener{
		onSession: func(e SessionStateChangedEvent) {
			if e.NewState == SessionStateRecording && e.Trigger == string(triggerStartRecording) {
				listenerSawEvent.Store(true)
				// Try to call GetSessions — this needs dataLock.
				// If initSession holds dataLock, this deadlocks.
				done := make(chan struct{})
				go func() {
					_ = m.GetSessions(recorderID)
					close(done)
				}()

				select {
				case <-done:
					// Good — no deadlock
				case <-time.After(2 * time.Second):
					listenerDeadlocked.Store(true)
				}
				close(listenerDone)
			}
		},
	})

	// Create a session — this calls initSession, which emits while holding dataLock
	sessionID := uuid.New()
	samples := make([]int16, 100)
	go func() {
		m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples)
	}()

	select {
	case <-listenerDone:
		if listenerDeadlocked.Load() {
			t.Error("Listener deadlocked trying to call GetSessions — initSession emits event while holding dataLock")
		}
	case <-time.After(5 * time.Second):
		if !listenerSawEvent.Load() {
			t.Skip("Listener didn't fire in time — test inconclusive")
		}
		t.Error("Test timed out — probable deadlock")
	}
}

// =============================================================================
// Issue #4: Double render submission on session switch
//
// When SafeChunks receives a new session ID, closeIntermediateSessions finds
// the old session in RECORDING and returns it in deferredCloses. Then the
// needsSessionSwitch path also fires triggerCloseRecording and submits a render
// for the same session. This causes duplicate render work.
// =============================================================================

func TestBug_DoubleRenderSubmissionOnSessionSwitch(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create first session
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID1, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks session1 failed: %v", err)
	}

	// Enable render submit counting
	m.renderQueue.enableCounting()

	// Start second session — triggers both closeIntermediateSessions and needsSessionSwitch
	if err := m.SafeChunks(ctx, recorderID, sessionID2, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks session2 failed: %v", err)
	}

	// Check how many renders were submitted for session1
	count := m.renderQueue.sessionRenderCount(sessionID1)
	if count > 1 {
		t.Errorf("Session1 render submitted %d times, want 1 (double submission wastes resources)", count)
	}
	if count == 0 {
		t.Error("Session1 render was never submitted")
	}
}
