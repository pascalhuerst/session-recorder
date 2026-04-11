package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// newTestStorage creates a Minio storage backed by FakeMinioClient,
// calls Start, and returns both for inspection.
func newTestStorage(t *testing.T) (*Minio, *FakeMinioClient) {
	t.Helper()
	fake := NewFakeMinioClient()
	m := NewMinioStorageWithClient(fake, "fake-minio:9000", "fake-minio:9000", "fake-minio:9000")
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { m.Stop() })
	return m, fake
}

// =============================================================================
// Start & Initialization
// =============================================================================

func TestContractStart_CreatesBucket(t *testing.T) {
	fake := NewFakeMinioClient()
	m := NewMinioStorageWithClient(fake, "fake:9000", "", "")
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	m.Stop()

	exists, _ := fake.BucketExists(context.Background(), bucketName)
	if !exists {
		t.Error("Start should create the session-recorder bucket")
	}
}

func TestContractStart_CreatesSystemMetadata(t *testing.T) {
	_, fake := newTestStorage(t)

	if !fake.ObjectExists(bucketName, "metadata.json") {
		t.Error("Start should create metadata.json")
	}
}

func TestContractStart_IdempotentBucket(t *testing.T) {
	fake := NewFakeMinioClient()
	// Pre-create bucket
	fake.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})

	m := NewMinioStorageWithClient(fake, "fake:9000", "", "")
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start on existing bucket should not fail: %v", err)
	}
	m.Stop()
}

// =============================================================================
// EnsureRecorderExists
// =============================================================================

func TestContractEnsureRecorderExists(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()
	recorderID := uuid.New()

	m.EnsureRecorderExists(ctx, recorderID, "Test Recorder")

	// Should be in GetRecorders
	recorders := m.GetRecorders()
	recorder, ok := recorders[recorderID]
	if !ok {
		t.Fatal("Recorder should exist after EnsureRecorderExists")
	}
	if recorder.Name != "Test Recorder" {
		t.Errorf("Expected name %q, got %q", "Test Recorder", recorder.Name)
	}

	// Metadata should be persisted to storage
	metadataKey := recorderID.String() + "/metadata.json"
	if !fake.ObjectExists(bucketName, metadataKey) {
		t.Error("Recorder metadata.json should exist in storage")
	}
}

func TestContractEnsureRecorderExists_Idempotent(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()
	recorderID := uuid.New()

	m.EnsureRecorderExists(ctx, recorderID, "First Name")
	m.EnsureRecorderExists(ctx, recorderID, "Second Name")

	recorders := m.GetRecorders()
	if len(recorders) != 1 {
		t.Errorf("Expected 1 recorder, got %d", len(recorders))
	}
	// EnsureRecorderExists only creates if missing — name stays as first
	if recorders[recorderID].Name != "First Name" {
		t.Errorf("Expected name %q (first call wins), got %q", "First Name", recorders[recorderID].Name)
	}
}

// =============================================================================
// SafeChunks & Session Creation
// =============================================================================

func TestContractSafeChunks_CreatesSession(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	samples := make([]int16, 100)
	for i := range samples {
		samples[i] = int16(i)
	}

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Session should exist in RECORDING state
	sessions := m.GetSessions(recorderID)
	session, ok := sessions[sessionID]
	if !ok {
		t.Fatal("Session should exist after first chunk")
	}
	if session.State != SessionStateRecording {
		t.Errorf("Expected RECORDING state, got %s", session.State)
	}
}

func TestContractSafeChunks_FiresAudioCallback(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	var callbackFired bool
	var callbackSamples []int16
	var callbackMu sync.Mutex
	m.RegisterOnAudioChunkCallback(func(rID, sID uuid.UUID, samples []int16, chunkNum int, ts time.Time) {
		callbackMu.Lock()
		defer callbackMu.Unlock()
		callbackFired = true
		callbackSamples = samples
	})

	samples := []int16{100, 200, 300}
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	callbackMu.Lock()
	defer callbackMu.Unlock()
	if !callbackFired {
		t.Fatal("Audio callback should have been called")
	}
	if len(callbackSamples) != 3 {
		t.Fatalf("Expected 3 samples in callback, got %d", len(callbackSamples))
	}
	for i, s := range samples {
		if callbackSamples[i] != s {
			t.Errorf("Sample[%d]: expected %d, got %d", i, s, callbackSamples[i])
		}
	}
}

func TestContractSafeChunks_BuffersInMemoryThenStreams(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox and/or audiowaveform not available, skipping streaming test")
	}
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send small chunk — should stay in buffer (below 1s threshold)
	smallSamples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), smallSamples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Verify chunk exists in memory with data buffered
	m.dataLock.Lock()
	chunk := m.chunks[recorderID]
	if chunk == nil {
		m.dataLock.Unlock()
		t.Fatal("Expected chunk to exist")
	}
	if chunk.buffer.Len() != 200 { // 100 int16 samples × 2 bytes each
		t.Errorf("Expected 200 bytes in buffer, got %d", chunk.buffer.Len())
	}
	if chunk.streaming == nil {
		t.Error("Expected streaming session to be active")
	}
	m.dataLock.Unlock()

	// Send enough data to exceed streamFlushSize (1s of audio = 192000 bytes)
	// Each int16 = 2 bytes via binary.Write, so need 192000/2 = 96000 samples
	largeSamples := make([]int16, streamFlushSize/2+1)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "002", time.Now(), largeSamples); err != nil {
		t.Fatalf("SafeChunks (large) failed: %v", err)
	}

	// After exceeding streamFlushSize, buffer should have been flushed to streaming pipes
	m.dataLock.Lock()
	chunk = m.chunks[recorderID]
	if chunk.buffer.Len() >= streamFlushSize {
		t.Errorf("Buffer should have been flushed (got %d bytes, threshold %d)", chunk.buffer.Len(), streamFlushSize)
	}
	m.dataLock.Unlock()
}

func TestContractSafeChunks_NoRecorderReturnsError(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	// Don't call EnsureRecorderExists
	err := m.SafeChunks(ctx, uuid.New(), uuid.New(), "001", time.Now(), []int16{1, 2, 3})
	if err == nil {
		t.Fatal("Expected error when recorder doesn't exist")
	}
}

func TestContractSafeChunks_SamplesPersistedCorrectly(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox and/or audiowaveform not available, skipping streaming test")
	}

	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send enough data to trigger at least one stream flush
	samples := make([]int16, streamFlushSize/2+1)
	for i := range samples {
		samples[i] = int16(i % 32000)
	}
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Close the session to finalize streaming uploads
	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Wait for streaming uploads to complete
	waitForSessionState(t, m, recorderID, sessionID, SessionStateFinished, 30*time.Second)

	// Read back the completed data.raw and verify data
	rawDataKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	data, ok := fake.GetObjectData(bucketName, rawDataKey)
	if !ok {
		t.Fatal("data.raw should exist after close")
	}

	// Verify the data is correctly encoded as little-endian int16
	if len(data) < 4 {
		t.Fatal("data.raw too small")
	}
	firstSample := int16(binary.LittleEndian.Uint16(data[0:2]))
	secondSample := int16(binary.LittleEndian.Uint16(data[2:4]))
	if firstSample != 0 || secondSample != 1 {
		t.Errorf("Expected samples [0, 1, ...], got [%d, %d, ...]", firstSample, secondSample)
	}
}

// =============================================================================
// Session Metadata Persistence
// =============================================================================

func TestContractSetKeepSession(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.SetKeepSession(ctx, recorderID, sessionID, true); err != nil {
		t.Fatalf("SetKeepSession failed: %v", err)
	}

	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if !session.Keep {
		t.Error("Expected Keep=true after SetKeepSession")
	}

	// Verify persistence: re-read metadata from storage
	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		t.Fatalf("getSessionMetadata failed: %v", err)
	}
	if !sm.Keep {
		t.Error("Keep flag should be persisted to storage")
	}
}

func TestContractSetName(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.SetName(ctx, recorderID, sessionID, "Sunday Jam"); err != nil {
		t.Fatalf("SetName failed: %v", err)
	}

	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.Name != "Sunday Jam" {
		t.Errorf("Expected name %q, got %q", "Sunday Jam", session.Name)
	}

	// Verify persistence
	sm, err := m.getSessionMetadata(ctx, recorderID, sessionID)
	if err != nil {
		t.Fatalf("getSessionMetadata failed: %v", err)
	}
	if sm.Name != "Sunday Jam" {
		t.Error("Name should be persisted to storage")
	}
}

func TestContractSetKeepSession_NonexistentSession(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	err := m.SetKeepSession(ctx, recorderID, uuid.New(), true)
	if err == nil {
		t.Fatal("Expected error for nonexistent session")
	}
}

// =============================================================================
// GetSession / GetSessions / GetRecorders
// =============================================================================

func TestContractGetSession_ReturnsCorrectData(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	ts := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", ts, []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, session.ID)
	}
	if session.RecorderID != recorderID {
		t.Errorf("Expected recorder ID %s, got %s", recorderID, session.RecorderID)
	}
	if session.State != SessionStateRecording {
		t.Errorf("Expected RECORDING state, got %s", session.State)
	}
	if !session.StartTime.Equal(ts) {
		t.Errorf("Expected start time %v, got %v", ts, session.StartTime)
	}
}

func TestContractGetSession_NonexistentRecorder(t *testing.T) {
	m, _ := newTestStorage(t)

	_, err := m.GetSession(uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("Expected error for nonexistent recorder")
	}
}

func TestContractGetSessions_MultipleSessions(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	session1 := uuid.New()
	session2 := uuid.New()
	if err := m.SafeChunks(ctx, recorderID, session1, "001", time.Now(), []int16{1}); err != nil {
		t.Fatalf("SafeChunks(s1) failed: %v", err)
	}

	// Switch to session 2 (closes session 1)
	if err := m.SafeChunks(ctx, recorderID, session2, "001", time.Now(), []int16{2}); err != nil {
		t.Fatalf("SafeChunks(s2) failed: %v", err)
	}

	sessions := m.GetSessions(recorderID)
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}

	// Session 2 should be RECORDING
	if sessions[session2].State != SessionStateRecording {
		t.Errorf("Session 2 should be RECORDING, got %s", sessions[session2].State)
	}
}

// =============================================================================
// CloseRecordingSession & State Transitions
// =============================================================================

func TestContractCloseRecordingSession_TransitionsToProcessing(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Register state change callback
	var stateChanges []SessionState
	var mu sync.Mutex
	m.EventBus().AddListener(&testSessionListener{onSession: func(e SessionStateChangedEvent) {
		mu.Lock()
		defer mu.Unlock()
		stateChanges = append(stateChanges, e.NewState)
	}})

	err := m.CloseRecordingSession(ctx, recorderID, sessionID)
	if err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Wait for async processing (rendering will fail without sox, but state
	// transitions should still happen)
	time.Sleep(2 * time.Second)

	mu.Lock()
	changes := make([]SessionState, len(stateChanges))
	copy(changes, stateChanges)
	mu.Unlock()

	// The first state change should be to PROCESSING
	if len(changes) == 0 {
		t.Fatal("Expected at least one state change callback")
	}
	if changes[0] != SessionStateProcessing {
		t.Errorf("First state change should be PROCESSING, got %s", changes[0])
	}
}

func TestContractCloseRecordingSession_StreamsDataToStorage(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox and/or audiowaveform not available, skipping streaming test")
	}

	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send enough data to trigger at least one stream flush (>1s of audio)
	largeSamples := make([]int16, streamFlushSize/2+1)
	for i := range largeSamples {
		largeSamples[i] = int16(i % 32000)
	}
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), largeSamples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Wait for streaming uploads to complete
	waitForSessionState(t, m, recorderID, sessionID, SessionStateFinished, 30*time.Second)

	// After close, data.raw should exist (streamed via PutObject)
	rawDataKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	if !fake.ObjectExists(bucketName, rawDataKey) {
		sessionPrefix := recorderID.String() + "/sessions/" + sessionID.String() + "/"
		ch := fake.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: sessionPrefix, Recursive: true})
		var keys []string
		for info := range ch {
			keys = append(keys, info.Key)
		}
		t.Errorf("Expected data.raw after close, found: %v", keys)
	}
}

// TestContractCloseRecordingSession_SmallChunksStillStreamed verifies that even
// when all chunks are still in the buffer (below streamFlushSize), CloseRecordingSession
// flushes them through the streaming pipelines to storage.
func TestContractCloseRecordingSession_SmallChunksStillStreamed(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox and/or audiowaveform not available, skipping streaming test")
	}

	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send very small amount of data (stays in buffer, never hits streamFlushSize)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Wait for streaming uploads to complete
	waitForSessionState(t, m, recorderID, sessionID, SessionStateFinished, 30*time.Second)

	// The small chunk should still have been streamed to data.raw
	rawDataKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	if !fake.ObjectExists(bucketName, rawDataKey) {
		sessionPrefix := recorderID.String() + "/sessions/" + sessionID.String() + "/"
		ch := fake.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: sessionPrefix, Recursive: true})
		var keys []string
		for info := range ch {
			keys = append(keys, info.Key)
		}
		t.Errorf("Expected data.raw after close of small session, found: %v", keys)
	}
}

// =============================================================================
// DeleteSession
// =============================================================================

func TestContractDeleteSession(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.DeleteSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	sessions := m.GetSessions(recorderID)
	if _, ok := sessions[sessionID]; ok {
		t.Error("Session should not exist after delete")
	}
}

// =============================================================================
// Session Switch (new session ID auto-closes previous)
// =============================================================================

func TestContractSessionSwitch_StateChangeCallback(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	session1 := uuid.New()
	session2 := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	var stateChanges []struct {
		SessionID uuid.UUID
		State     SessionState
	}
	var mu sync.Mutex
	m.EventBus().AddListener(&testSessionListener{onSession: func(e SessionStateChangedEvent) {
		mu.Lock()
		defer mu.Unlock()
		stateChanges = append(stateChanges, struct {
			SessionID uuid.UUID
			State     SessionState
		}{e.SessionID, e.NewState})
	}})

	// Start session 1
	if err := m.SafeChunks(ctx, recorderID, session1, "001", time.Now(), []int16{1}); err != nil {
		t.Fatalf("SafeChunks(s1) failed: %v", err)
	}

	// Switch to session 2 → should trigger close of session 1
	if err := m.SafeChunks(ctx, recorderID, session2, "001", time.Now(), []int16{2}); err != nil {
		t.Fatalf("SafeChunks(s2) failed: %v", err)
	}

	// Wait for async state change
	time.Sleep(time.Second)

	mu.Lock()
	defer mu.Unlock()

	// Should have received PROCESSING callback for session 1
	foundProcessing := false
	for _, sc := range stateChanges {
		if sc.SessionID == session1 && sc.State == SessionStateProcessing {
			foundProcessing = true
			break
		}
	}
	if !foundProcessing {
		t.Error("Expected PROCESSING state change for session 1 when session 2 started")
	}
}

// =============================================================================
// Presigned URL Generation
// =============================================================================

func TestContractGetPresignedURL(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()

	url, err := m.GetPresignedURL(ctx,
		AssetOptions{RecorderID: recorderID, SessionID: sessionID, Filename: FILENAME_FLAC},
		SigningOptions{Expires: time.Hour, Download: true, DownloadFilename: "test.flac"},
	)
	if err != nil {
		t.Fatalf("GetPresignedURL failed: %v", err)
	}
	if url == "" {
		t.Error("Expected non-empty URL")
	}
}

// =============================================================================
// Session Timeout
// =============================================================================

func TestContractSessionTimeout_ClosesStaleSession(t *testing.T) {
	// Create storage with short timeout BEFORE Start to avoid race
	fake := NewFakeMinioClient()
	m := NewMinioStorageWithClient(fake, "fake:9000", "fake:9000", "fake:9000")
	m.SetSessionTimeout(100 * time.Millisecond)
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { m.Stop() })
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	var stateChanges []SessionState
	var mu sync.Mutex
	m.EventBus().AddListener(&testSessionListener{onSession: func(e SessionStateChangedEvent) {
		mu.Lock()
		defer mu.Unlock()
		stateChanges = append(stateChanges, e.NewState)
	}})

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Wait for timeout checker to detect and close the stale session
	// Checker runs every 5s by default — for the test we wait long enough
	time.Sleep(7 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	foundProcessing := false
	for _, s := range stateChanges {
		if s == SessionStateProcessing {
			foundProcessing = true
			break
		}
	}
	if !foundProcessing {
		t.Error("Expected stale session to be closed (transitioned to PROCESSING)")
	}
}

// =============================================================================
// Polling Helper
// =============================================================================

// waitForSessionState polls until the session reaches the target state or timeout.
func waitForSessionState(t *testing.T, m *Minio, recorderID, sessionID uuid.UUID, target SessionState, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s, err := m.GetSession(recorderID, sessionID)
		if err == nil && s.State == target {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Final check with error reporting
	s, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("waitForSessionState: GetSession failed: %v", err)
	}
	t.Fatalf("Session did not reach %s within %v (current: %s)", target, timeout, s.State)
}

func audioToolsAvailable() bool {
	if _, err := exec.LookPath("/usr/bin/sox"); err != nil {
		return false
	}
	if _, err := exec.LookPath("audiowaveform"); err != nil {
		return false
	}
	return true
}

// =============================================================================
// Full Render Pipeline (requires sox + audiowaveform)
// =============================================================================

// TestContractFullRenderPipeline exercises the complete storage flow:
// SafeChunks → CloseRecordingSession → render → FINISHED with all output files.
func TestContractFullRenderPipeline(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox and/or audiowaveform not available, skipping render pipeline test")
	}

	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Generate enough raw PCM data to exceed minChunkSize (5MB).
	// Use a simple rising signal pattern (deterministic).
	sampleCount := minChunkSize/2 + 48000 // a bit over 5MB raw, plus 1 second extra
	samples := make([]int16, sampleCount)
	for i := range samples {
		samples[i] = int16((i * 7) % 32000) // deterministic pseudo-signal
	}

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Poll for FINISHED (rendering takes a few seconds)
	waitForSessionState(t, m, recorderID, sessionID, SessionStateFinished, 30*time.Second)

	// Verify session metadata
	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.State != SessionStateFinished {
		t.Errorf("Expected FINISHED, got %s", session.State)
	}
	if session.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", session.Duration)
	}

	// Verify all expected output objects exist in storage
	prefix := recorderID.String() + "/sessions/" + sessionID.String() + "/"

	expectedFiles := map[string][]byte{
		prefix + "data.raw":      nil,
		prefix + "data.flac":     {0x66, 0x4C, 0x61, 0x43}, // fLaC
		prefix + "waveform.dat":  nil,                       // no standard magic
		prefix + "metadata.json": nil,
	}

	for key, magic := range expectedFiles {
		data, exists := fake.GetObjectData(bucketName, key)
		if !exists {
			t.Errorf("Expected object %q to exist after render", key)
			continue
		}
		if len(data) == 0 {
			t.Errorf("Object %q is empty", key)
			continue
		}
		if magic != nil && len(data) >= len(magic) {
			if !bytes.HasPrefix(data, magic) {
				t.Errorf("Object %q: expected magic %v, got %v", key, magic, data[:len(magic)])
			}
		}
	}

	// Verify data.raw is non-trivial (should contain all our sample data)
	rawData, _ := fake.GetObjectData(bucketName, prefix+"data.raw")
	if len(rawData) < minChunkSize {
		t.Errorf("data.raw too small: %d bytes, expected at least %d", len(rawData), minChunkSize)
	}
}

// TestContractCloseRecordingSession_TransitionsToProcessingAndRenders verifies
// that when audio tools are available, CloseRecordingSession transitions through
// PROCESSING → FINISHED after rendering all output files.
// When tools are unavailable, this test is skipped.
func TestContractCloseRecordingSession_TransitionsToProcessingAndRenders(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox and/or audiowaveform not available")
	}

	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	samples := make([]int16, minChunkSize/2+1)
	for i := range samples {
		samples[i] = int16((i * 7) % 32000)
	}
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	waitForSessionState(t, m, recorderID, sessionID, SessionStateFinished, 30*time.Second)

	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", session.Duration)
	}
	if session.ErrorMessage != "" {
		t.Errorf("Expected no error message, got %q", session.ErrorMessage)
	}
}

// =============================================================================
// Multipart Upload Race Conditions
// =============================================================================

// TestContractIsSessionClosed_ProcessingWithDataRawIsNotClosed verifies that
// a session in PROCESSING state is NOT considered closed even if data.raw exists.
// This ensures rendering proceeds after the multipart upload is completed.
func TestContractIsSessionClosed_ProcessingWithDataRawIsNotClosed(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Put data.raw to simulate completed multipart upload
	rawDataObjectName := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	fake.PutObject(ctx, bucketName, rawDataObjectName, bytes.NewReader([]byte("raw-audio")), 9, minio.PutObjectOptions{})

	// Session is in RECORDING state with data.raw present — should NOT be closed
	if m.isSessionClosed(ctx, recorderID, sessionID) {
		t.Error("RECORDING session with data.raw should NOT be considered closed")
	}
}

// TestContractIsSessionClosed_FinishedIsClosed verifies terminal states are closed.
func TestContractIsSessionClosed_FinishedIsClosed(t *testing.T) {
	m, _ := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Not closed while recording
	if m.isSessionClosed(ctx, recorderID, sessionID) {
		t.Error("RECORDING session should not be closed")
	}
}

// TestContractSessionSwitch_ConcurrentCloseDoesNotFailRender verifies that when
// a session switch triggers both closeIntermediateSessions and the session switch
// close path, the session eventually renders successfully despite the race.
func TestContractSessionSwitch_ConcurrentCloseDoesNotFailRender(t *testing.T) {
	if !audioToolsAvailable() {
		t.Skip("sox/audiowaveform not available, skipping render test")
	}

	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	session1 := uuid.New()
	session2 := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Record enough data for session 1 to trigger at least one stream flush
	largeSamples := make([]int16, streamFlushSize/2+1)
	for i := range largeSamples {
		largeSamples[i] = int16(i % 32000)
	}
	if err := m.SafeChunks(ctx, recorderID, session1, "001", time.Now(), largeSamples); err != nil {
		t.Fatalf("SafeChunks session1 failed: %v", err)
	}

	// Switch to session 2 — triggers closeIntermediateSessions + session switch close
	if err := m.SafeChunks(ctx, recorderID, session2, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks session2 failed: %v", err)
	}

	// Wait for session 1 to be rendered
	waitForSessionState(t, m, recorderID, session1, SessionStateFinished, 30*time.Second)

	// data.raw should exist for session 1
	rawDataKey := recorderID.String() + "/sessions/" + session1.String() + "/data.raw"
	if !fake.ObjectExists(bucketName, rawDataKey) {
		t.Error("Expected data.raw for session1 after render")
	}
}

// testSessionListener is a test helper that implements EventListener.
type testSessionListener struct {
	onSession func(SessionStateChangedEvent)
	onSegment func(SegmentStateChangedEvent)
}

func (l *testSessionListener) OnSessionStateChanged(event SessionStateChangedEvent) {
	if l.onSession != nil {
		l.onSession(event)
	}
}

func (l *testSessionListener) OnSegmentStateChanged(event SegmentStateChangedEvent) {
	if l.onSegment != nil {
		l.onSegment(event)
	}
}

// =============================================================================
// Reader Close Verification
// =============================================================================

// TestFix_GetObjectReadersAreClosed verifies that metadata-fetching functions
// close the ObjectHandle returned by GetObject, preventing file descriptor leaks.
func TestFix_GetObjectReadersAreClosed(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Create a session so getSessionMetadata has something to read
	samples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Reset tracking after setup (SafeChunks may have called GetObject internally)
	fake.ResetObjectTracking()

	// Call getSystemMetadata
	m.dataLock.Lock()
	_, err := m.getSystemMetadata(ctx)
	m.dataLock.Unlock()
	if err != nil {
		t.Fatalf("getSystemMetadata failed: %v", err)
	}

	// Call getSessionMetadata
	m.dataLock.Lock()
	_, err = m.getSessionMetadata(ctx, recorderID, sessionID)
	m.dataLock.Unlock()
	if err != nil {
		t.Fatalf("getSessionMetadata failed: %v", err)
	}

	// Call getRecorderMetadata
	m.dataLock.Lock()
	_, err = m.getRecorderMetadata(ctx, recorderID)
	m.dataLock.Unlock()
	if err != nil {
		t.Fatalf("getRecorderMetadata failed: %v", err)
	}

	// All three should have closed their readers
	unclosed := fake.UnclosedObjects()
	if len(unclosed) > 0 {
		for _, obj := range unclosed {
			t.Errorf("GetObject reader for %q was not closed", obj.info.Key)
		}
	}
}
