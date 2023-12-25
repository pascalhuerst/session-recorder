package model

import "time"

type Segment struct {
	ID         string `json:"id"`
	Comment    string `json:"comment"`
	StartPoint int64  `json:"start_point"`
	EndPoint   int64  `json:"end_point"`

	Recordings map[string]Recording `json:"recordings"`
}

type SessionMetadata struct {
	GenericMetadata GenericMetadata `json:"generic_metadata"`

	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

type Session struct {
	Metadata *SessionMetadata `json:"metadata"`

	IsClosed bool `json:"is_closed"`

	// key: segment id
	Segments map[string]Segment `json:"segments"`
}
