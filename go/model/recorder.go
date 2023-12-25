package model

type RecorderMetadata struct {
	GenericMetadata *GenericMetadata `json:"generic_metadata"`
}

type Recorder struct {
	Metadata *RecorderMetadata `json:"metadata"`

	// key: session id
	Sessions map[string]Session `json:"sessions"`
}
