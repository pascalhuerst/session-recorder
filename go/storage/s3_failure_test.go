package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestS3Failure_PutObject_DuringSafeChunks(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	fake.SetPutObjectError(errors.New("S3 unavailable"))

	// SafeChunks itself doesn't return the putSessionMetadata error,
	// but the session should NOT appear in GetSessions because initSession
	// bails before updating the in-memory map.
	_ = m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3})

	sessions := m.GetSessions(recorderID)
	if _, ok := sessions[sessionID]; ok {
		t.Error("Session should NOT exist when PutObject failed during init")
	}

	// Clear the error and retry with a new session ID (the old chunk state
	// may be inconsistent, so use a fresh session).
	fake.SetPutObjectError(nil)
	sessionID2 := uuid.New()
	if err := m.SafeChunks(ctx, recorderID, sessionID2, "001", time.Now(), []int16{4, 5, 6}); err != nil {
		t.Fatalf("SafeChunks should succeed after clearing error: %v", err)
	}

	sessions = m.GetSessions(recorderID)
	if _, ok := sessions[sessionID2]; !ok {
		t.Error("Session should exist after successful SafeChunks")
	}
}

func TestS3Failure_PutObject_MetadataWriteFailure(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	originalName := session.Name

	fake.SetPutObjectError(errors.New("S3 unavailable"))

	err = m.SetName(ctx, recorderID, sessionID, "new-name")
	if err == nil {
		t.Fatal("SetName should return error when PutObject fails")
	}

	fake.SetPutObjectError(nil)

	// The in-memory name should NOT have changed because putSessionMetadata
	// failed before the in-memory update in SetName.
	session, err = m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed after clearing error: %v", err)
	}
	if session.Name != originalName {
		t.Errorf("Name should be unchanged after failed SetName: got %q, want %q", session.Name, originalName)
	}
}

func TestS3Failure_GetObject_DuringSessionLoad(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Verify we can read metadata before injecting error.
	if _, err := m.getSessionMetadata(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("getSessionMetadata should succeed: %v", err)
	}

	fake.SetGetObjectError(errors.New("S3 unavailable"))

	_, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err == nil {
		t.Fatal("getSessionMetadata should fail when GetObject returns error")
	}

	fake.SetGetObjectError(nil)

	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		t.Fatalf("getSessionMetadata should succeed after clearing error: %v", err)
	}
	if sm.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, sm.ID)
	}
}

func TestS3Failure_RecoveryAfterTransientError(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// First attempt: PutObject fails, session not created in memory.
	fake.SetPutObjectError(errors.New("S3 unavailable"))
	_ = m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3})

	sessions := m.GetSessions(recorderID)
	if _, ok := sessions[sessionID]; ok {
		t.Fatal("Session should not exist after failed PutObject")
	}

	// Clear error, use a new session (the old chunk is in a bad state).
	fake.SetPutObjectError(nil)
	sessionID2 := uuid.New()

	if err := m.SafeChunks(ctx, recorderID, sessionID2, "001", time.Now(), []int16{4, 5, 6}); err != nil {
		t.Fatalf("SafeChunks should succeed: %v", err)
	}

	// Send more chunks to the same session.
	if err := m.SafeChunks(ctx, recorderID, sessionID2, "002", time.Now(), []int16{7, 8, 9}); err != nil {
		t.Fatalf("Second SafeChunks should succeed: %v", err)
	}

	// Close the session.
	if err := m.CloseRecordingSession(ctx, recorderID, sessionID2); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Session should be in PROCESSING state (render submitted).
	session, err := m.GetSession(recorderID, sessionID2)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.State != SessionStateProcessing {
		t.Errorf("Expected PROCESSING state, got %s", session.State)
	}
}
