package storage

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// =============================================================================
// Bug #7: Delete-while-rendering race (session resurrection)
//
// DeleteSession can complete while a render job is in-flight. The render's
// onSessionTransition callback creates a minimal session entry if the session
// is not found in the in-memory map (line 458), effectively resurrecting the
// deleted session in both memory and S3 metadata.
// =============================================================================

func TestBug_DeleteWhileRenderingResurrectsSession(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session in PROCESSING state with data.raw present so renderSession
	// progresses past the StatObject check and into renderFromRawData which will
	// fail (no sox) and fire triggerRenderFailure → onSessionTransition.
	rawKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	rawData := make([]byte, 48000*2*2) // 1 second of raw PCM
	fake.PutObject(ctx, bucketName, rawKey, bytes.NewReader(rawData), int64(len(rawData)), minio.PutObjectOptions{})

	m.dataLock.Lock()
	session := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		StartTime:  time.Now(),
		State:      SessionStateProcessing,
		Segments:   make(map[uuid.UUID]Segment),
	}
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m.dataLock.Unlock()
		t.Fatalf("putSessionMetadata failed: %v", err)
	}
	m.system.Recorders[recorderID].Sessions[sessionID] = session
	m.dataLock.Unlock()
	m.getOrCreateSessionMachine(recorderID, sessionID, SessionStateProcessing)

	// Simulate the race: the render worker grabs the SM reference BEFORE the delete.
	// This is what happens in fireSessionTrigger — it looks up the SM, releases
	// machineLock, then calls sm.FireCtx().
	m.machineLock.Lock()
	sm, ok := m.sessionMachines[sessionID]
	m.machineLock.Unlock()
	if !ok || sm == nil {
		t.Fatal("State machine should exist before delete")
	}

	// Now delete the session — removes from memory, S3, and sessionMachines map.
	if err := m.DeleteSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify it's gone
	sessions := m.GetSessions(recorderID)
	if _, exists := sessions[sessionID]; exists {
		t.Fatal("Session should be deleted")
	}

	// Now fire on the stale SM reference — this is what the render worker does
	// when it grabbed the SM before the delete completed.
	// The callback onSessionTransition creates a minimal session entry if not
	// found in the map — this is the bug: it resurrects the deleted session.
	errCtx := context.WithValue(ctx, renderErrorKey{}, "render error")
	_ = sm.FireCtx(errCtx, triggerRenderFailure)

	// Check that the session was NOT resurrected
	sessions = m.GetSessions(recorderID)
	if _, exists := sessions[sessionID]; exists {
		t.Error("Session was resurrected after deletion — onSessionTransition created ghost entry")
	}

	// Also verify no metadata was written back to S3 after deletion
	metaKey := recorderID.String() + "/sessions/" + sessionID.String() + "/metadata.json"
	if fake.ObjectExists(bucketName, metaKey) {
		t.Error("Session metadata was re-created in S3 after deletion")
	}
}

// =============================================================================
// Bug #8: setSegmentError bypasses transition validation
//
// setSegmentError directly sets segment.State = SegmentStateError without
// calling validateSegmentTransition. This means it could be called from any
// state, even states where -> ERROR is not a valid transition (e.g., QUEUED).
// =============================================================================

func TestBug_SetSegmentErrorBypassesValidation(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	segmentID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a FINISHED session with a segment in QUEUED state
	m.dataLock.Lock()
	session := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		StartTime:  time.Now(),
		State:      SessionStateFinished,
		Segments: map[uuid.UUID]Segment{
			segmentID: {
				ID:         segmentID,
				Comment:    "test",
				StartPoint: 0,
				EndPoint:   100,
				State:      SegmentStateQueued,
			},
		},
	}
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m.dataLock.Unlock()
		t.Fatalf("putSessionMetadata failed: %v", err)
	}
	m.system.Recorders[recorderID].Sessions[sessionID] = session
	m.dataLock.Unlock()

	// setSegmentError from QUEUED is invalid (QUEUED can only -> RENDERING).
	// Before the fix, this silently succeeds.
	m.setSegmentError(ctx, recorderID, sessionID, segmentID, "forced error")

	// Check that the segment state was NOT changed to ERROR (QUEUED -> ERROR is invalid)
	s, _ := m.GetSession(recorderID, sessionID)
	seg := s.Segments[segmentID]
	if seg.State == SegmentStateError {
		t.Error("setSegmentError allowed invalid transition QUEUED -> ERROR (bypassed validation)")
	}
}

// =============================================================================
// Bug #9: renderSegmentSync hardcodes PreviousState in RENDERING event
//
// The SegmentStateChangedEvent emitted when transitioning to RENDERING
// hardcodes PreviousState: SegmentStateQueued instead of using the actual
// previous state captured before the transition.
// =============================================================================

func TestBug_SegmentRenderingEventHardcodesPreviousState(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	segmentID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session with raw data and a segment in QUEUED state
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Put a data.raw file so the segment render can find it
	rawKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	rawData := make([]byte, 48000*2*2) // 1 second of silence
	fake.PutObject(ctx, bucketName, rawKey, bytes.NewReader(rawData), int64(len(rawData)), minio.PutObjectOptions{})

	m.dataLock.Lock()
	sess := m.system.Recorders[recorderID].Sessions[sessionID]
	sess.State = SessionStateFinished
	sess.Segments = map[uuid.UUID]Segment{
		segmentID: {
			ID:         segmentID,
			Comment:    "test",
			StartPoint: 0,
			EndPoint:   100,
			State:      SegmentStateQueued,
		},
	}
	if err := m.putSessionMetadata(ctx, recorderID, sessionID, &sess); err != nil {
		m.dataLock.Unlock()
		t.Fatalf("putSessionMetadata failed: %v", err)
	}
	m.system.Recorders[recorderID].Sessions[sessionID] = sess
	m.dataLock.Unlock()

	// Listen for segment state change events
	var mu sync.Mutex
	var events []SegmentStateChangedEvent
	m.EventBus().AddListener(&testSessionListener{
		onSegment: func(e SegmentStateChangedEvent) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		},
	})

	// Trigger the render synchronously (will fail without sox, but the QUEUED->RENDERING event fires first)
	_ = m.renderSegmentSync(ctx, recorderID, sessionID, segmentID)

	mu.Lock()
	defer mu.Unlock()

	// Find the QUEUED -> RENDERING event
	for _, e := range events {
		if e.NewState == SegmentStateRendering {
			if e.PreviousState != SegmentStateQueued {
				t.Errorf("RENDERING event PreviousState = %s, want QUEUED", e.PreviousState)
			}
			return
		}
	}
	t.Error("No QUEUED -> RENDERING event emitted")
}

// =============================================================================
// Bug #10: closeIntermediateSessions bypasses FSM callbacks
//
// When a new session starts while an old one is still RECORDING,
// closeIntermediateSessions writes session.State = SessionStateProcessing
// directly instead of using the FSM. The event IS emitted manually, but the
// FSM's onSessionTransition callback (and its side effects) are skipped.
// =============================================================================

func TestBug_CloseIntermediateSessionsBypassesFSM(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create first session
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID1, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks for session1 failed: %v", err)
	}

	// Track events
	var mu sync.Mutex
	var sessionEvents []SessionStateChangedEvent
	m.EventBus().AddListener(&testSessionListener{
		onSession: func(e SessionStateChangedEvent) {
			mu.Lock()
			sessionEvents = append(sessionEvents, e)
			mu.Unlock()
		},
	})

	// Start second session — this should close the first one via closeIntermediateSessions
	if err := m.SafeChunks(ctx, recorderID, sessionID2, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks for session2 failed: %v", err)
	}

	// Wait briefly for events
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Verify that session1 got a PROCESSING state change event with an FSM trigger
	for _, e := range sessionEvents {
		if e.SessionID == sessionID1 && e.NewState == SessionStateProcessing {
			// After the fix, the trigger should be "CloseRecording" (the FSM trigger),
			// not "startup-close" (the manual bypass)
			if e.Trigger != string(triggerCloseRecording) {
				t.Errorf("Expected FSM trigger %q for intermediate session close, got %q",
					triggerCloseRecording, e.Trigger)
			}
			return
		}
	}
	t.Error("Session1 should have transitioned to PROCESSING when session2 started")
}
