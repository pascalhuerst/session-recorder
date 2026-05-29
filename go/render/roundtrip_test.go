package render

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"

	opusgo "github.com/kazzmir/opus-go"
	"github.com/mewkiz/flac"
)

/**
 * Test Plan: Encode round-trips against a known 5s 1kHz sine (s16le, 2ch, 48kHz)
 *
 * FLAC is lossless: encode -> decode must reproduce the input sample-for-sample
 * (bit-exact A/B comparison).
 *
 * Opus is lossy: encode -> decode cannot be bit-exact, so we assert the decoded
 * audio has the right length (within encoder delay/padding), carries real energy
 * (RMS in range), and that the dominant tone is still ~1kHz (Goertzel).
 */

// decodeFlacToRaw decodes a FLAC stream back to interleaved s16le PCM bytes.
func decodeFlacToRaw(t *testing.T, flacBytes []byte) []byte {
	t.Helper()
	stream, err := flac.New(bytes.NewReader(flacBytes))
	if err != nil {
		t.Fatalf("flac.New: %v", err)
	}
	out := new(bytes.Buffer)
	for {
		fr, err := stream.ParseNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ParseNext: %v", err)
		}
		n := len(fr.Subframes[0].Samples)
		for i := 0; i < n; i++ {
			for ch := 0; ch < numChannels; ch++ {
				binary.Write(out, binary.LittleEndian, int16(fr.Subframes[ch].Samples[i]))
			}
		}
	}
	return out.Bytes()
}

func TestFlacRoundTripIsLossless(t *testing.T) {
	in := genSineRaw(1000, 5) // 5s @ 48kHz, multiple of the FLAC block size

	var encoded bytes.Buffer
	if err := Flac(&encoded, bytes.NewReader(in)); err != nil {
		t.Fatalf("Flac: %v", err)
	}

	out := decodeFlacToRaw(t, encoded.Bytes())

	if len(out) != len(in) {
		t.Fatalf("decoded length %d != input length %d", len(out), len(in))
	}
	if !bytes.Equal(out, in) {
		// Pinpoint the first divergence to aid debugging.
		for i := range in {
			if in[i] != out[i] {
				t.Fatalf("FLAC round-trip not lossless: first diff at byte %d (in=%d out=%d)", i, in[i], out[i])
			}
		}
	}
}

// Production data.raw is not aligned to the 16-frame FLAC block size, so the
// encoder must emit a smaller final block and still round-trip losslessly.
func TestFlacRoundTripNonAligned(t *testing.T) {
	in := genSineRaw(1000, 1)
	tail := make([]byte, 7*bytesPerFrame) // 7 frames: not a multiple of 16
	for i := range tail {
		tail[i] = byte(i)
	}
	in = append(in, tail...)

	var encoded bytes.Buffer
	if err := Flac(&encoded, bytes.NewReader(in)); err != nil {
		t.Fatalf("Flac: %v", err)
	}

	out := decodeFlacToRaw(t, encoded.Bytes())
	if len(out) != len(in) {
		t.Fatalf("decoded length %d != input length %d", len(out), len(in))
	}
	if !bytes.Equal(out, in) {
		t.Fatal("FLAC round-trip not lossless for non-16-frame-aligned input")
	}
}

func TestOpusRoundTripPreservesTone(t *testing.T) {
	skipIfRace(t)

	const freq = 1000.0
	in := genSineRaw(freq, 5)

	var encoded bytes.Buffer
	if err := Opus(&encoded, bytes.NewReader(in)); err != nil {
		t.Fatalf("Opus: %v", err)
	}
	if !bytes.HasPrefix(encoded.Bytes(), []byte("OggS")) {
		t.Fatalf("Opus output is not an Ogg stream")
	}

	player, err := opusgo.NewPlayerFromReader(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("NewPlayerFromReader: %v", err)
	}
	decoded, err := io.ReadAll(player)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Length: lossy decode should be within ~2% of input (encoder delay/padding).
	ratio := float64(len(decoded)) / float64(len(in))
	if ratio < 0.98 || ratio > 1.02 {
		t.Fatalf("decoded length ratio %.4f out of bounds (in=%d out=%d)", ratio, len(in), len(decoded))
	}

	mono := toMonoFloat(decoded)

	rms := rms(mono)
	// Input amplitude 16000 mono-summed/2 => peak ~16000, sine RMS ~11300.
	if rms < 5000 || rms > 16000 {
		t.Fatalf("decoded RMS %.0f out of expected range (tone missing or clipped)", rms)
	}

	// The 1kHz tone must dominate an off-frequency probe (3kHz).
	on := goertzel(mono, freq, sampleRate)
	off := goertzel(mono, 3000, sampleRate)
	if on < off*10 {
		t.Fatalf("1kHz energy (%.3g) does not dominate 3kHz energy (%.3g)", on, off)
	}
}

// toMonoFloat converts interleaved s16le stereo bytes to a mono float64 signal
// (channel average).
func toMonoFloat(raw []byte) []float64 {
	nFrames := len(raw) / bytesPerFrame
	out := make([]float64, nFrames)
	for i := 0; i < nFrames; i++ {
		l := int16(binary.LittleEndian.Uint16(raw[i*bytesPerFrame:]))
		r := int16(binary.LittleEndian.Uint16(raw[i*bytesPerFrame+2:]))
		out[i] = (float64(l) + float64(r)) / 2
	}
	return out
}

func rms(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	var sum float64
	for _, v := range x {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(x)))
}

// goertzel returns the squared magnitude of the signal at targetHz.
func goertzel(x []float64, targetHz, sr float64) float64 {
	w := 2 * math.Pi * targetHz / sr
	cw := 2 * math.Cos(w)
	var s0, s1, s2 float64
	for _, v := range x {
		s0 = v + cw*s1 - s2
		s2 = s1
		s1 = s0
	}
	return s1*s1 + s2*s2 - cw*s1*s2
}
