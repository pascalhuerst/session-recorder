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

// OnSessionClosedCb is called when a session finishes rendering (legacy callback)
type OnSessionClosedCb func(session *Session)

// OnSessionStateChangedCb is called when a session's state changes
type OnSessionStateChangedCb func(session *Session, previousState SessionState)

// OnAudioChunkCb is called when audio samples are received for streaming
type OnAudioChunkCb func(recorderID, sessionID uuid.UUID, samples []int16, chunkNumber int, timestamp time.Time)

type Filename string

const (
	FILENAME_OGG      = Filename("data.ogg")
	FILENAME_FLAC     = Filename("data.flac")
	FILENAME_WAVEFORM = Filename("waveform.dat")
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

	Start(ctx context.Context) error
	Stop()

	SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunkID string, timeCreated time.Time, samples []int16) error
	EnsureRecorderExists(ctx context.Context, recorderID uuid.UUID, recorderName string)

	DeleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
	SetKeepSession(ctx context.Context, recorderID, sessionID uuid.UUID, keep bool) error
	SetName(ctx context.Context, recorderID, sessionID uuid.UUID, name string) error

	CloseRecordingSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
	RetryRenderSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
	//CloseSession(ctx context.Context, RecorderID, SessionID uuid.UUID) error
	//CloseOpenSessions(ctx context.Context, RecorderID uuid.UUID) error

	RegisterOnSessionClosedCallback(cb OnSessionClosedCb) error
	RegisterOnSessionStateChangedCallback(cb OnSessionStateChangedCb) error
	RegisterOnAudioChunkCallback(cb OnAudioChunkCb) error

	GetPresignedURL(ctx context.Context, asset AssetOptions, options SigningOptions) (string, error)

	// GetSessionFileReader returns a reader for a session file
	GetSessionFileReader(ctx context.Context, asset AssetOptions) (io.ReadCloser, int64, error)

	// GetSegmentFileReader returns a reader for a segment file
	GetSegmentFileReader(ctx context.Context, asset SegmentAssetOptions) (io.ReadCloser, int64, error)

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

type Session struct {
	ID         uuid.UUID `json:"id"`
	RecorderID uuid.UUID `json:"recorder_id"`
	Name       string    `json:"name"`

	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// State replaces IsClosed - tracks full lifecycle state
	State        SessionState `json:"state"`
	ErrorMessage string       `json:"error_message,omitempty"`
	Keep         bool         `json:"keep"`

	// IsClosed is deprecated, kept for backward compatibility with existing metadata
	IsClosed bool `json:"is_closed,omitempty"`

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
