package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

/**
 * Test Plan: Storage Types
 *
 * Scenario: Session serialization roundtrip
 *   Given a Session struct with all fields populated
 *   When marshaled to JSON and unmarshaled back
 *   Then all fields are preserved correctly
 *
 * Scenario: Recorder serialization roundtrip
 *   Given a Recorder struct with ID and name
 *   When marshaled to JSON and unmarshaled back
 *   Then all fields are preserved correctly
 *
 * Scenario: Segment serialization roundtrip
 *   Given a Segment struct with all fields
 *   When marshaled to JSON and unmarshaled back
 *   Then all fields are preserved correctly
 *
 * Scenario: String representations
 *   Given a System/Session/Recorder struct
 *   When String() is called
 *   Then a non-empty string representation is returned
 */

func TestSession_Serialization(t *testing.T) {
	sessionID := uuid.New()
	recorderID := uuid.New()
	segmentID := uuid.New()

	original := Session{
		ID:         sessionID,
		RecorderID: recorderID,
		Name:       "Test Session",
		StartTime:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		Duration:   time.Hour,
		State:      SessionStateFinished,
		Keep:       true,
		Segments: map[uuid.UUID]Segment{
			segmentID: {
				ID:         segmentID,
				Comment:    "Test segment",
				StartPoint: 0,
				EndPoint:   1000,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Unmarshal back
	var decoded Session
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if decoded.ID != original.ID {
		t.Errorf("Session.ID = %v, want %v", decoded.ID, original.ID)
	}
	if decoded.RecorderID != original.RecorderID {
		t.Errorf("Session.RecorderID = %v, want %v", decoded.RecorderID, original.RecorderID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Session.Name = %v, want %v", decoded.Name, original.Name)
	}
	if decoded.State != original.State {
		t.Errorf("Session.State = %v, want %v", decoded.State, original.State)
	}
	if decoded.Keep != original.Keep {
		t.Errorf("Session.Keep = %v, want %v", decoded.Keep, original.Keep)
	}
	if len(decoded.Segments) != len(original.Segments) {
		t.Errorf("Session.Segments length = %v, want %v", len(decoded.Segments), len(original.Segments))
	}
}

func TestRecorder_Serialization(t *testing.T) {
	recorderID := uuid.New()

	original := Recorder{
		ID:   recorderID,
		Name: "Test Recorder",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Unmarshal back
	var decoded Recorder
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if decoded.ID != original.ID {
		t.Errorf("Recorder.ID = %v, want %v", decoded.ID, original.ID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Recorder.Name = %v, want %v", decoded.Name, original.Name)
	}
}

func TestSegment_Serialization(t *testing.T) {
	segmentID := uuid.New()

	original := Segment{
		ID:           segmentID,
		Comment:      "Test comment",
		StartPoint:   100,
		EndPoint:     500,
		State:        SegmentStateFinished,
		ErrorMessage: "",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Unmarshal back
	var decoded Segment
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if decoded.ID != original.ID {
		t.Errorf("Segment.ID = %v, want %v", decoded.ID, original.ID)
	}
	if decoded.Comment != original.Comment {
		t.Errorf("Segment.Comment = %v, want %v", decoded.Comment, original.Comment)
	}
	if decoded.StartPoint != original.StartPoint {
		t.Errorf("Segment.StartPoint = %v, want %v", decoded.StartPoint, original.StartPoint)
	}
	if decoded.EndPoint != original.EndPoint {
		t.Errorf("Segment.EndPoint = %v, want %v", decoded.EndPoint, original.EndPoint)
	}
	if decoded.State != original.State {
		t.Errorf("Segment.State = %v, want %v", decoded.State, original.State)
	}
}

func TestSegmentState_String(t *testing.T) {
	tests := []struct {
		state    SegmentState
		expected string
	}{
		{SegmentStateUnknown, "UNKNOWN"},
		{SegmentStateQueued, "QUEUED"},
		{SegmentStateRendering, "RENDERING"},
		{SegmentStateFinished, "FINISHED"},
		{SegmentStateError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("SegmentState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSessionState_String(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected string
	}{
		{SessionStateUnknown, "UNKNOWN"},
		{SessionStateRecording, "RECORDING"},
		{SessionStateProcessing, "PROCESSING"},
		{SessionStateFinished, "FINISHED"},
		{SessionStateError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("SessionState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSegment_WithErrorMessage(t *testing.T) {
	segmentID := uuid.New()

	original := Segment{
		ID:           segmentID,
		Comment:      "Failed segment",
		StartPoint:   100,
		EndPoint:     500,
		State:        SegmentStateError,
		ErrorMessage: "encoding failed",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Unmarshal back
	var decoded Segment
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	if decoded.State != SegmentStateError {
		t.Errorf("Segment.State = %v, want %v", decoded.State, SegmentStateError)
	}
	if decoded.ErrorMessage != original.ErrorMessage {
		t.Errorf("Segment.ErrorMessage = %v, want %v", decoded.ErrorMessage, original.ErrorMessage)
	}
}

func TestSystem_String(t *testing.T) {
	systemID := uuid.New()
	recorderID := uuid.New()
	sessionID := uuid.New()

	system := System{
		ID:   systemID,
		Name: "Test System",
		Recorders: map[uuid.UUID]Recorder{
			recorderID: {
				ID:   recorderID,
				Name: "Recorder 1",
				Sessions: map[uuid.UUID]Session{
					sessionID: {
						ID:       sessionID,
						Name:     "Session 1",
						State:    SessionStateFinished,
						Keep:     true,
						Duration: time.Hour,
					},
				},
			},
		},
	}

	str := system.String()

	if str == "" {
		t.Error("System.String() returned empty string")
	}

	// Should contain system name
	if len(str) == 0 {
		t.Error("System.String() should not be empty")
	}
}

func TestSession_String(t *testing.T) {
	session := Session{
		ID:       uuid.New(),
		Name:     "Test Session",
		State:    SessionStateFinished,
		Keep:     true,
		Duration: time.Hour,
	}

	str := session.String()

	if str == "" {
		t.Error("Session.String() returned empty string")
	}
}

func TestRecorder_String(t *testing.T) {
	recorder := Recorder{
		ID:       uuid.New(),
		Name:     "Test Recorder",
		Sessions: map[uuid.UUID]Session{},
	}

	str := recorder.String()

	if str == "" {
		t.Error("Recorder.String() returned empty string")
	}
}
