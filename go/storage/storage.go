package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// SessionState represents the lifecycle state of a session
type SessionState int32

const (
	SessionStateUnknown    SessionState = 0
	SessionStateRecording  SessionState = 1
	SessionStateFinished   SessionState = 2
	SessionStateProcessing SessionState = 3
	SessionStateError      SessionState = 4
)

func (s SessionState) String() string {
	switch s {
	case SessionStateRecording:
		return "RECORDING"
	case SessionStateProcessing:
		return "PROCESSING"
	case SessionStateFinished:
		return "FINISHED"
	case SessionStateError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// SegmentState represents the lifecycle state of a segment
type SegmentState int32

const (
	SegmentStateUnknown   SegmentState = 0
	SegmentStateRendering SegmentState = 1
	SegmentStateFinished  SegmentState = 2
	SegmentStateError     SegmentState = 3
	SegmentStateQueued    SegmentState = 4
)

func (s SegmentState) String() string {
	switch s {
	case SegmentStateQueued:
		return "QUEUED"
	case SegmentStateRendering:
		return "RENDERING"
	case SegmentStateFinished:
		return "FINISHED"
	case SegmentStateError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// OnSessionStateChangedCb is called when a session's state changes
type OnSessionStateChangedCb func(session *Session, previousState SessionState)

// OnAudioChunkCb is called when audio samples are received for streaming
type OnAudioChunkCb func(recorderID, sessionID uuid.UUID, samples []int16, chunkNumber int, timestamp time.Time)

type Filename string

const (
	FILENAME_RAW      = Filename("data.raw")
	FILENAME_OGG      = Filename("data.ogg")
	FILENAME_FLAC     = Filename("data.flac")
	FILENAME_WAVEFORM = Filename("waveform.dat")
	FILENAME_PREVIEW  = Filename("preview.png")
	FILENAME_METADATA = Filename("metadata.json")

	SEGMENT_FILENAME_OGG  = Filename("segment.ogg")
	SEGMENT_FILENAME_FLAC = Filename("segment.flac")
)

type AssetOptions struct {
	RecorderID uuid.UUID
	SessionID  uuid.UUID
	Filename   Filename
}

type SegmentAssetOptions struct {
	RecorderID uuid.UUID
	SessionID  uuid.UUID
	SegmentID  uuid.UUID
	Filename   Filename
}

// EndpointType specifies which endpoint to use for signed URLs
type EndpointType int

const (
	// EndpointLocal uses the local endpoint (for UI/browser consumption)
	EndpointLocal EndpointType = iota
	// EndpointPublic uses the public endpoint (for external sharing via email)
	EndpointPublic
)

type SigningOptions struct {
	Expires          time.Duration
	Download         bool
	DownloadFilename string
	Endpoint         EndpointType // Which endpoint to use (default: EndpointLocal)
}

type Storage interface {
	GetRecorders() map[uuid.UUID]Recorder

	GetSessions(recorderID uuid.UUID) map[uuid.UUID]Session
	GetSession(recorderID, sessionID uuid.UUID) (Session, error)

	// SnapshotSessions returns a copy of every session across all recorders,
	// built under the backend's data lock. The returned slice is safe to iterate
	// without holding any lock and without racing concurrent chunk reception or
	// deletion (used by the retention sweeper).
	SnapshotSessions() []SessionRef

	Start(ctx context.Context) error
	// Shutdown is called on graceful exit. The backend should flush any
	// in-flight per-recorder state (e.g. the in-memory chunk buffer) and mark
	// active sessions as resumable so they can be picked up on the next Start.
	// Implementations should be safe to call concurrently with no active RPCs.
	Shutdown(ctx context.Context) error

	SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunkID string, timeCreated time.Time, samples []int16) error
	EnsureRecorderExists(ctx context.Context, recorderID uuid.UUID, recorderName string)

	DeleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
	SetKeepSession(ctx context.Context, recorderID, sessionID uuid.UUID, keep bool) error
	SetName(ctx context.Context, recorderID, sessionID uuid.UUID, name string) error

	CloseRecordingSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
	isSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool

	RegisterOnSessionStateChangedCallback(cb OnSessionStateChangedCb) error
	RegisterOnAudioChunkCallback(cb OnAudioChunkCb) error

	GetPresignedURL(ctx context.Context, asset AssetOptions, options SigningOptions) (string, error)

	// GetSessionFileReader returns a reader for a session file
	GetSessionFileReader(ctx context.Context, asset AssetOptions) (io.ReadCloser, int64, error)

	// GetSegmentFileReader returns a reader for a segment file
	GetSegmentFileReader(ctx context.Context, asset SegmentAssetOptions) (io.ReadCloser, int64, error)

	// EnsurePreview renders preview.png from waveform.dat when the preview is
	// missing, returning true if it created one. It is a no-op (false, nil) when
	// the preview already exists or there is no waveform.dat to render from. Used
	// by the housekeeping backfill for sessions rendered before previews existed.
	EnsurePreview(ctx context.Context, recorderID, sessionID uuid.UUID) (bool, error)

	// Segment operations
	CreateSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, segment Segment) error
	UpdateSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, segment Segment) error
	DeleteSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error
	SetSegmentState(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID, state SegmentState) error
	RenderSegment(ctx context.Context, recorderID, sessionID, segmentID uuid.UUID) error
	GetSegmentPresignedURL(ctx context.Context, asset SegmentAssetOptions, options SigningOptions) (string, error)
}

type System struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`

	Recorders map[uuid.UUID]Recorder `json:"-"`
}

func (s *System) String() string {
	ret := fmt.Sprintf("%s [%s]\n", s.Name, s.ID)
	ret += "  Recorders:\n"

	for _, r := range s.Recorders {
		ret += r.String()
	}

	return ret
}

type Recorder struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`

	// key: session id
	Sessions map[uuid.UUID]Session `json:"-"`
}

func (r Recorder) String() string {
	ret := fmt.Sprintf("  %s [%s]\n", r.Name, r.ID)
	ret += "    Sessions:\n"

	for _, s := range r.Sessions {
		ret += s.String()
	}

	return ret
}

// SessionRef pairs a session snapshot with the IDs needed to address it,
// independent of whether the embedded Session has its own ID fields populated.
type SessionRef struct {
	RecorderID uuid.UUID
	SessionID  uuid.UUID
	Session    Session
}

// copySessionForSnapshot returns a copy of session with its Segments map cloned,
// so the result can be read (including ranging Segments) without holding the
// data lock and without racing concurrent segment create/delete.
func copySessionForSnapshot(session Session) Session {
	if session.Segments != nil {
		segments := make(map[uuid.UUID]Segment, len(session.Segments))
		for id, seg := range session.Segments {
			segments[id] = seg
		}
		session.Segments = segments
	}
	return session
}

type Session struct {
	ID         uuid.UUID `json:"id"`
	RecorderID uuid.UUID `json:"recorder_id"`
	Name       string    `json:"name"`

	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	State        SessionState `json:"state"`
	ErrorMessage string       `json:"error_message,omitempty"`
	Keep         bool         `json:"keep"`

	// PartialChunkNumber, if non-nil, names a chunks/<n> object that holds an
	// in-flight buffer flushed on graceful shutdown. The session is resumable:
	// on the next Start, the named chunk is loaded back into memory and
	// deleted from disk. Always nil during normal operation.
	PartialChunkNumber *int `json:"partial_chunk_number,omitempty"`

	// key: segment id
	Segments map[uuid.UUID]Segment `json:"segments"`
}

func (s Session) String() string {
	strKeep := ""
	if s.Keep {
		strKeep = " keep"
	}

	return fmt.Sprintf("    %s [%s] (%v)%s [%s]\n", s.Name, s.State.String(), s.Duration, strKeep, s.ID)
}

type Segment struct {
	ID           uuid.UUID    `json:"id"`
	Comment      string       `json:"comment"`
	StartPoint   int64        `json:"start_point"`
	EndPoint     int64        `json:"end_point"`
	State        SegmentState `json:"state"`
	ErrorMessage string       `json:"error_message,omitempty"`
}
