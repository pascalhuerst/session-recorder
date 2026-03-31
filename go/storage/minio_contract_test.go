package storage

import (
	"context"
	"encoding/binary"
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

func TestContractSafeChunks_BuffersInMemoryThenFlushes(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send small chunk — should stay in memory, not flushed yet
	smallSamples := make([]int16, 100)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), smallSamples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	chunksPrefix := recorderID.String() + "/sessions/" + sessionID.String() + "/chunks/"
	ch := fake.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("Small chunk should not be flushed yet, but found %d objects", count)
	}

	// Send enough data to exceed minChunkSize (5MB)
	// Each int16 = 2 bytes via binary.Write, so need 5MB/2 = 2.5M samples
	largeSamples := make([]int16, minChunkSize/2+1)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "002", time.Now(), largeSamples); err != nil {
		t.Fatalf("SafeChunks (large) failed: %v", err)
	}

	// Now a chunk should have been flushed to storage
	ch = fake.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})
	count = 0
	for range ch {
		count++
	}
	if count == 0 {
		t.Error("Large chunk should have been flushed to storage")
	}
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
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send enough data to flush
	samples := make([]int16, minChunkSize/2+1)
	for i := range samples {
		samples[i] = int16(i % 32000)
	}
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), samples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Read back the flushed chunk and verify data
	chunkKey := recorderID.String() + "/sessions/" + sessionID.String() + "/chunks/0000000000000000"
	data, ok := fake.GetObjectData(bucketName, chunkKey)
	if !ok {
		t.Fatal("Chunk object should exist in storage")
	}

	// Verify the data is correctly encoded as little-endian int16
	if len(data) < 4 {
		t.Fatal("Chunk data too small")
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

func TestContractCloseRecordingSession_FlushesLargeChunksToStorage(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send enough data that at least one chunk is flushed to storage BEFORE close.
	// This is important because isSessionClosed() checks for chunk objects in storage —
	// if no chunks exist in storage (only in memory), closeSessionAsync skips the session.
	largeSamples := make([]int16, minChunkSize/2+1)
	for i := range largeSamples {
		largeSamples[i] = int16(i % 32000)
	}
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), largeSamples); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	// Verify chunk exists in storage before close
	chunksPrefix := recorderID.String() + "/sessions/" + sessionID.String() + "/chunks/"
	ch := fake.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: chunksPrefix, Recursive: true})
	preCloseCount := 0
	for range ch {
		preCloseCount++
	}
	if preCloseCount == 0 {
		t.Fatal("Expected chunk to be flushed to storage before close")
	}

	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Wait for async compose + render (render will fail without sox, that's OK)
	time.Sleep(3 * time.Second)

	// After close, data.raw should exist (composed from chunks)
	rawDataKey := recorderID.String() + "/sessions/" + sessionID.String() + "/data.raw"
	if !fake.ObjectExists(bucketName, rawDataKey) {
		// List what's there for debugging
		sessionPrefix := recorderID.String() + "/sessions/" + sessionID.String() + "/"
		ch := fake.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: sessionPrefix, Recursive: true})
		var keys []string
		for info := range ch {
			keys = append(keys, info.Key)
		}
		t.Errorf("Expected data.raw after close, found: %v", keys)
	}
}

// TestContractCloseRecordingSession_SmallChunksStillFlushed verifies that even
// when all chunks are still in memory (below minChunkSize), CloseRecordingSession
// flushes them to storage before checking isSessionClosed.
func TestContractCloseRecordingSession_SmallChunksStillFlushed(t *testing.T) {
	m, fake := newTestStorage(t)
	ctx := context.Background()

	recorderID := uuid.New()
	sessionID := uuid.New()
	m.EnsureRecorderExists(ctx, recorderID, "recorder")

	// Send very small amount of data (stays in memory)
	if err := m.SafeChunks(ctx, recorderID, sessionID, "001", time.Now(), []int16{1, 2, 3}); err != nil {
		t.Fatalf("SafeChunks failed: %v", err)
	}

	if err := m.CloseRecordingSession(ctx, recorderID, sessionID); err != nil {
		t.Fatalf("CloseRecordingSession failed: %v", err)
	}

	// Wait for async flush + compose
	time.Sleep(3 * time.Second)

	// The small chunk should still have been flushed (padded to minChunkSize)
	// and composed into data.raw
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
