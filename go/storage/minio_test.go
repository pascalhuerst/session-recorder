package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

/**
 * Test Plan: MinIO Storage
 *
 * Scenario: Handle invalid endpoint gracefully
 *   Given an invalid endpoint format
 *   When NewMinioStorage is called
 *   Then either an error is returned or a struct is created that fails on connect
 *
 * Scenario: Handle empty credentials
 *   Given empty access key and secret key
 *   When NewMinioStorage is called
 *   Then the storage struct is created (validation deferred to connect)
 *
 * Scenario: Set session timeout
 *   Given a valid Minio storage instance
 *   When SetSessionTimeout is called with a duration
 *   Then the session timeout is updated
 *
 * Scenario: Session state transitions
 *   Given session state enum values
 *   When comparing states
 *   Then the correct state values are returned
 */

func TestNewMinioStorage_InvalidEndpoint(t *testing.T) {
	// Test with invalid endpoint - should still create the struct
	// but fail when trying to connect
	storage, err := NewMinioStorage("invalid:endpoint:format", "local", "public", "access", "secret")
	if err != nil {
		// Some invalid endpoints may fail at creation time
		t.Logf("NewMinioStorage() error = %v (expected for invalid endpoint)", err)
		return
	}

	if storage == nil {
		t.Error("NewMinioStorage() returned nil storage")
	}
}

func TestNewMinioStorage_EmptyCredentials(t *testing.T) {
	// Test with empty credentials - should still create the struct
	storage, err := NewMinioStorage("localhost:9000", "localhost:9000", "localhost:9000", "", "")
	if err != nil {
		t.Logf("NewMinioStorage() error = %v", err)
		return
	}

	if storage == nil {
		t.Error("NewMinioStorage() returned nil storage")
	}
}

func TestMinio_GetRecorders_Empty(t *testing.T) {
	// Create a Minio struct directly for unit testing
	m := &Minio{
		system: &System{
			ID:        uuid.New(),
			Name:      "Test System",
			Recorders: make(map[uuid.UUID]Recorder),
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	recorders := m.GetRecorders()
	if recorders == nil {
		t.Error("GetRecorders() returned nil")
	}
	if len(recorders) != 0 {
		t.Errorf("GetRecorders() returned %d recorders, want 0", len(recorders))
	}
}

func TestMinio_GetRecorders_WithData(t *testing.T) {
	recorderID := uuid.New()

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:       recorderID,
					Name:     "Test Recorder",
					Sessions: make(map[uuid.UUID]Session),
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	recorders := m.GetRecorders()
	if len(recorders) != 1 {
		t.Errorf("GetRecorders() returned %d recorders, want 1", len(recorders))
	}

	if _, ok := recorders[recorderID]; !ok {
		t.Error("GetRecorders() does not contain expected recorder")
	}
}

func TestMinio_GetSessions_Empty(t *testing.T) {
	recorderID := uuid.New()

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:       recorderID,
					Name:     "Test Recorder",
					Sessions: make(map[uuid.UUID]Session),
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	sessions := m.GetSessions(recorderID)
	if sessions == nil {
		t.Error("GetSessions() returned nil")
	}
	if len(sessions) != 0 {
		t.Errorf("GetSessions() returned %d sessions, want 0", len(sessions))
	}
}

func TestMinio_GetSessions_WithData(t *testing.T) {
	recorderID := uuid.New()
	sessionID := uuid.New()

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:   recorderID,
					Name: "Test Recorder",
					Sessions: map[uuid.UUID]Session{
						sessionID: {
							ID:         sessionID,
							RecorderID: recorderID,
							Name:       "Test Session",
							StartTime:  time.Now(),
							State:      SessionStateFinished,
						},
					},
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	sessions := m.GetSessions(recorderID)
	if len(sessions) != 1 {
		t.Errorf("GetSessions() returned %d sessions, want 1", len(sessions))
	}

	if _, ok := sessions[sessionID]; !ok {
		t.Error("GetSessions() does not contain expected session")
	}
}

func TestMinio_GetSession_Success(t *testing.T) {
	recorderID := uuid.New()
	sessionID := uuid.New()

	expectedSession := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		Name:       "Test Session",
		StartTime:  time.Now(),
		State:      SessionStateFinished,
		Keep:       true,
	}

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:   recorderID,
					Name: "Test Recorder",
					Sessions: map[uuid.UUID]Session{
						sessionID: expectedSession,
					},
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	session, err := m.GetSession(recorderID, sessionID)
	if err != nil {
		t.Errorf("GetSession() error = %v", err)
		return
	}

	if session.ID != expectedSession.ID {
		t.Errorf("GetSession() ID = %v, want %v", session.ID, expectedSession.ID)
	}
	if session.Name != expectedSession.Name {
		t.Errorf("GetSession() Name = %v, want %v", session.Name, expectedSession.Name)
	}
}

func TestMinio_GetSession_RecorderNotFound(t *testing.T) {
	m := &Minio{
		system: &System{
			ID:        uuid.New(),
			Name:      "Test System",
			Recorders: make(map[uuid.UUID]Recorder),
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	_, err := m.GetSession(uuid.New(), uuid.New())
	if err == nil {
		t.Error("GetSession() error = nil, want error for non-existent recorder")
	}
}

func TestMinio_GetSession_SessionNotFound(t *testing.T) {
	recorderID := uuid.New()

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:       recorderID,
					Name:     "Test Recorder",
					Sessions: make(map[uuid.UUID]Session),
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	_, err := m.GetSession(recorderID, uuid.New())
	if err == nil {
		t.Error("GetSession() error = nil, want error for non-existent session")
	}
}


func TestFilenameConstants(t *testing.T) {
	tests := []struct {
		name     string
		filename Filename
		want     string
	}{
		{"OGG", FILENAME_OGG, "data.ogg"},
		{"FLAC", FILENAME_FLAC, "data.flac"},
		{"WAVEFORM", FILENAME_WAVEFORM, "waveform.dat"},
		{"METADATA", FILENAME_METADATA, "metadata.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.filename) != tt.want {
				t.Errorf("Filename %s = %v, want %v", tt.name, tt.filename, tt.want)
			}
		})
	}
}

func TestAssetOptions(t *testing.T) {
	recorderID := uuid.New()
	sessionID := uuid.New()

	options := AssetOptions{
		RecorderID: recorderID,
		SessionID:  sessionID,
		Filename:   FILENAME_OGG,
	}

	if options.RecorderID != recorderID {
		t.Errorf("AssetOptions.RecorderID = %v, want %v", options.RecorderID, recorderID)
	}
	if options.SessionID != sessionID {
		t.Errorf("AssetOptions.SessionID = %v, want %v", options.SessionID, sessionID)
	}
	if options.Filename != FILENAME_OGG {
		t.Errorf("AssetOptions.Filename = %v, want %v", options.Filename, FILENAME_OGG)
	}
}

func TestSigningOptions(t *testing.T) {
	options := SigningOptions{
		Expires:          time.Hour,
		Download:         true,
		DownloadFilename: "test.ogg",
	}

	if options.Expires != time.Hour {
		t.Errorf("SigningOptions.Expires = %v, want %v", options.Expires, time.Hour)
	}
	if !options.Download {
		t.Error("SigningOptions.Download = false, want true")
	}
	if options.DownloadFilename != "test.ogg" {
		t.Errorf("SigningOptions.DownloadFilename = %v, want %v", options.DownloadFilename, "test.ogg")
	}
}

func TestMinio_CloseIntermediateSessions_RecorderNotFound(t *testing.T) {
	m := &Minio{
		system: &System{
			ID:        uuid.New(),
			Name:      "Test System",
			Recorders: make(map[uuid.UUID]Recorder),
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	// Should not panic when recorder doesn't exist
	m.closeIntermediateSessions(t.Context(), uuid.New())
}

func TestMinio_CloseIntermediateSessions_NoIntermediateSessions(t *testing.T) {
	recorderID := uuid.New()
	sessionID := uuid.New()

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:   recorderID,
					Name: "Test Recorder",
					Sessions: map[uuid.UUID]Session{
						sessionID: {
							ID:         sessionID,
							RecorderID: recorderID,
							State:      SessionStateFinished,
						},
					},
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	// Should not modify finished sessions
	m.closeIntermediateSessions(t.Context(), recorderID)

	session := m.system.Recorders[recorderID].Sessions[sessionID]
	if session.State != SessionStateFinished {
		t.Errorf("Session state = %v, want %v", session.State, SessionStateFinished)
	}
}

func TestMinio_CloseIntermediateSessions_IgnoresErrorState(t *testing.T) {
	recorderID := uuid.New()
	sessionID := uuid.New()

	m := &Minio{
		system: &System{
			ID:   uuid.New(),
			Name: "Test System",
			Recorders: map[uuid.UUID]Recorder{
				recorderID: {
					ID:   recorderID,
					Name: "Test Recorder",
					Sessions: map[uuid.UUID]Session{
						sessionID: {
							ID:           sessionID,
							RecorderID:   recorderID,
							State:        SessionStateError,
							ErrorMessage: "test error",
						},
					},
				},
			},
		},
		chunks: make(map[uuid.UUID]*minioChunk),
	}

	// Should not modify error sessions
	m.closeIntermediateSessions(t.Context(), recorderID)

	session := m.system.Recorders[recorderID].Sessions[sessionID]
	if session.State != SessionStateError {
		t.Errorf("Session state = %v, want %v", session.State, SessionStateError)
	}
}
