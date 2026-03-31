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
