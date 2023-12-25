package model

type GenericMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type System struct {
	Recorders map[string]Recorder `json:"recorders"`
}
