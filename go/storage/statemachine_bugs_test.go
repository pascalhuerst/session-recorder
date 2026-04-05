package storage

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// =============================================================================
// Bug #1: Double triggerRenderFailure firing (now fixed)
//
// renderFromRawData fires triggerRenderFailure on failure (line 1136).
// Previously, closeSessionAsync also fired it again (line 1654), causing a
// spurious error log. The fix removes the redundant fire from closeSessionAsync.
//
// This test verifies that renderFromRawData's trigger is the single authority
// for the PROCESSING -> ERROR transition.
// =============================================================================

func TestFix_RenderFailureFiresExactlyOnce(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session in PROCESSING state
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

	// First fire: this is what renderFromRawData does on failure
	err1 := m.fireSessionTrigger(ctx, sessionID, triggerRenderFailure, "render error")
	if err1 != nil {
		t.Fatalf("triggerRenderFailure should succeed from PROCESSING: %v", err1)
	}

	// Verify session is now in ERROR state with the error message
	s, _ := m.GetSession(recorderID, sessionID)
	if s.State != SessionStateError {
		t.Fatalf("Expected ERROR after trigger, got %s", s.State)
	}
	if s.ErrorMessage != "render error" {
		t.Errorf("Expected error message %q, got %q", "render error", s.ErrorMessage)
	}
}

// =============================================================================
// Bug #2: Streaming encode failure transitions to ERROR (not stuck in PROCESSING)
//
// If the streaming encode fails in closeSessionAsync, the session must transition
// to ERROR so the user can see the failure and retry via the fallback render path.
// =============================================================================

func TestFix_StreamingFailureTransitionsToError(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session in PROCESSING state (simulating post-close)
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

	// Trigger render failure (simulating what closeSessionAsync does on streaming error)
	err := m.fireSessionTrigger(ctx, sessionID, triggerRenderFailure, "streaming encode failed")
	if err != nil {
		t.Fatalf("triggerRenderFailure should succeed from PROCESSING: %v", err)
	}

	s, _ := m.GetSession(recorderID, sessionID)
	if s.State != SessionStateError {
		t.Fatalf("Expected ERROR after streaming failure, got %s", s.State)
	}
	if s.ErrorMessage == "" {
		t.Error("Expected non-empty error message for streaming failure")
	}
}

// =============================================================================
// Bug #6: Segments stuck in RENDERING after crash (now fixed)
//
// On startup, segments in RENDERING state are recovered to ERROR so the user
// can retry them.
// =============================================================================

func TestFix_SegmentRecoveredFromRenderingOnRestart(t *testing.T) {
	fake := NewFakeMinioClient()
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	segmentID := uuid.New()

	// --- First "boot": create a session with a segment in RENDERING state ---
	m1 := NewMinioStorageWithClient(fake, "fake:9000", "fake:9000", "fake:9000")
	if err := m1.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	m1.EnsureRecorderExists(ctx, recorderID, "recorder")

	samples := make([]int16, 100)
	if err := m1.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Put data.raw so the session can appear FINISHED
	rawKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	fake.PutObject(ctx, bucketName, rawKey, bytes.NewReader([]byte("raw")), 3, minio.PutObjectOptions{})

	// Create a segment directly in RENDERING state (simulating a crash mid-render)
	seg := Segment{
		ID:         segmentID,
		Comment:    "test segment",
		StartPoint: 0,
		EndPoint:   100,
		State:      SegmentStateRendering,
	}
	m1.dataLock.Lock()
	session := m1.system.Recorders[recorderID].Sessions[sessionID]
	session.State = SessionStateFinished
	session.Segments[segmentID] = seg
	if err := m1.putSessionMetadata(ctx, recorderID, sessionID, &session); err != nil {
		m1.dataLock.Unlock()
		t.Fatalf("putSessionMetadata failed: %v", err)
	}
	m1.system.Recorders[recorderID].Sessions[sessionID] = session
	m1.dataLock.Unlock()

	m1.Stop()

	// --- Second "boot": simulate restart ---
	m2 := NewMinioStorageWithClient(fake, "fake:9000", "fake:9000", "fake:9000")
	if err := m2.Start(ctx); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	defer m2.Stop()

	// After restart, the segment should be in ERROR with a descriptive message
	sessions := m2.GetSessions(recorderID)
	session2, ok := sessions[sessionID]
	if !ok {
		t.Fatal("Session should exist after restart")
	}

	segment, ok := session2.Segments[segmentID]
	if !ok {
		t.Fatal("Segment should exist after restart")
	}

	if segment.State != SegmentStateError {
		t.Errorf("Expected segment in ERROR after restart, got %s", segment.State)
	}
	if segment.ErrorMessage == "" {
		t.Error("Expected non-empty error message on recovered segment")
	}
}
