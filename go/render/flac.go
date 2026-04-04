package render

import (
	"bytes"
	"io"
)

// FlacStream encodes raw PCM audio to FLAC, writing directly to w.
func FlacStream(raw io.Reader, w io.Writer) error {
	return CreateAudioFileStream(raw, "flac", w)
}

// Flac encodes raw PCM audio to FLAC, returning the result as a buffer.
func Flac(raw io.Reader) (*bytes.Buffer, error) {
	return CreateAudioFile(raw, "flac")
}
