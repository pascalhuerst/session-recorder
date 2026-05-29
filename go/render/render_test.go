package render

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	_ "embed"
)

// skipIfRace skips tests that build a kazzmir/opus-go encoder. That library is
// ccgo-transpiled libopus and does C-style pointer arithmetic that the race
// detector's checkptr instrumentation fatally rejects (`go test -race`). Normal
// builds and the shipped binaries are unaffected.
func skipIfRace(t *testing.T) {
	if raceEnabled {
		t.Skip("skipped under -race: opus-go (transpiled libopus) trips checkptr")
	}
}

// rawTestAudio is shared raw PCM test input (s16le, 2ch, 48kHz) used by the
// clip / flac tests in this package.
//
//go:embed test_data/sweep_30_20000_s16le_2ch_48000k.raw
var rawTestAudio []byte

// genSineRaw returns interleaved s16le PCM (2ch, 48kHz) of a sine wave at the
// given frequency and duration — the same sample format used throughout the
// recorder. The amplitude is an exact integer so FLAC round-trips bit-exactly.
func genSineRaw(freqHz float64, seconds float64) []byte {
	nFrames := int(seconds * sampleRate)
	buf := bytes.NewBuffer(make([]byte, 0, nFrames*bytesPerFrame))
	const amplitude = 16000.0
	for i := range nFrames {
		v := int16(amplitude * math.Sin(2*math.Pi*freqHz*float64(i)/float64(sampleRate)))
		for range numChannels {
			binary.Write(buf, binary.LittleEndian, v)
		}
	}
	return buf.Bytes()
}
