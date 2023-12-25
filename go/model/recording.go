package model

import "time"

type FileType string

const (
	FileTypeWav  FileType = "wav"
	FileTypeMp3  FileType = "mp3"
	FileTypeOgg  FileType = "ogg"
	FileTypeFlac FileType = "flac"
)

type Rendering struct {
	ID   string   `json:"id"`
	Type FileType `json:"type"`
}

type Recording struct {
	GenericMetadata *GenericMetadata `json:"generic_metadata"`

	Created time.Time `json:"created"`

	// key: segment id
	Renderings map[string]Rendering
}
