package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OnSessionClosedCb func(session *Session)

type Filename string

const (
	FILENAME_OGG      = Filename("data.ogg")
	FILENAME_FLAC     = Filename("data.flac")
	FILENAME_WAVEFORM = Filename("waveform.dat")
	FILENAME_METADATA = Filename("metadata.json")
)

type AssetOptions struct {
	RecorderID uuid.UUID
	SessionID  uuid.UUID
	Filename   Filename
}

type SigningOptions struct {
	Expires          time.Duration
	Download         bool
	DownloadFilename string
}

type Storage interface {
	GetRecorders() map[uuid.UUID]Recorder

	GetSessions(recorderID uuid.UUID) map[uuid.UUID]Session
	GetSession(recorderID, sessionID uuid.UUID) (Session, error)

	Start(ctx context.Context) error

	SafeChunks(ctx context.Context, recorderID, sessionID uuid.UUID, chunkID string, timeCreated time.Time, samples []int16) error
	EnsureRecorderExists(ctx context.Context, recorderID uuid.UUID, recorderName string)

	DeleteSession(ctx context.Context, recorderID, sessionID uuid.UUID) error
	SetKeepSession(ctx context.Context, recorderID, sessionID uuid.UUID, keep bool) error

	isSessionClosed(ctx context.Context, recorderID, sessionID uuid.UUID) bool
	//CloseSession(ctx context.Context, RecorderID, SessionID uuid.UUID) error
	//CloseOpenSessions(ctx context.Context, RecorderID uuid.UUID) error

	RegisterOnSessionClosedCallback(cb OnSessionClosedCb) error

	GetPresignedURL(ctx context.Context, asset AssetOptions, options SigningOptions) (string, error)
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

	IsClosed bool `json:"is_closed"`
	Keep     bool `json:"keep"`

	// key: segment id
	Segments map[uuid.UUID]Segment `json:"segments"`
}

func (s Session) String() string {
	strClosed := " open "
	if s.IsClosed {
		strClosed = "closed"
	}

	strKeep := "keep"
	if !s.Keep {
		strKeep = ""
	}

	return fmt.Sprintf("    %s [%s] (%v) %s [%s]\n", s.Name, strClosed, s.Duration, strKeep, s.ID)
}

type Segment struct {
	ID         uuid.UUID `json:"id"`
	Comment    string    `json:"comment"`
	StartPoint int64     `json:"start_point"`
	EndPoint   int64     `json:"end_point"`
}
