package storage

import "github.com/google/uuid"

// SessionStateChangedEvent is emitted when a session transitions between lifecycle states.
type SessionStateChangedEvent struct {
	RecorderID    uuid.UUID
	SessionID     uuid.UUID
	PreviousState SessionState
	NewState      SessionState
	Trigger       string // e.g. "StartRecording", "CloseRecording", "RenderSuccess", "RenderFailure", "RetryRender"
	ErrorMessage  string
	Session       Session // full snapshot (value copy, safe for concurrent reads)
}

// RenderProgressEvent is emitted periodically during session rendering.
type RenderProgressEvent struct {
	RecorderID uuid.UUID
	SessionID  uuid.UUID
	Progress   float64 // 0.0 to 1.0
}

// SegmentStateChangedEvent is emitted when a segment transitions between lifecycle states.
type SegmentStateChangedEvent struct {
	RecorderID    uuid.UUID
	SessionID     uuid.UUID
	SegmentID     uuid.UUID
	PreviousState SegmentState
	NewState      SegmentState
	ErrorMessage  string
	Session       Session // full session snapshot (value copy)
}
