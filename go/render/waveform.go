package render

import (
	"bytes"
	"io"

	"github.com/pascalhuerst/session-recorder/audiowaveform"
)

// CreateWaveform generates an audiowaveform ".dat" (binary, 8-bit, mono) from
// raw interleaved S16LE / 48 kHz / 2-channel audio. It is a pure-Go
// reimplementation of the BBC `audiowaveform` tool for the single
// configuration the recorder uses, so there is no external binary dependency.
// The output is byte-for-byte identical to the C++ tool (see the golden test
// in the audiowaveform package).
func CreateWaveform(raw io.Reader) (*bytes.Buffer, error) {
	return audiowaveform.GenerateDat(raw, audiowaveform.DefaultParams())
}
