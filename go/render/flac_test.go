package render

import (
	"bytes"
	"testing"
)

/**
 * Test Plan: FLAC Encoding (via sox)
 *
 * Scenario: Encode valid raw audio to FLAC
 *   Given raw PCM audio data (s16le, 2ch, 48kHz)
 *   When Flac() is called
 *   Then a valid FLAC buffer is returned with fLaC magic header
 *
 * Scenario: Encode small audio samples
 *   Given minimal stereo 16-bit samples
 *   When Flac() is called
 *   Then a valid FLAC buffer is returned with fLaC magic header
 *
 * Scenario: Deterministic output
 *   Given the same raw PCM input
 *   When Flac() is called twice
 *   Then both outputs are bit-identical
 */

func TestFlac_ValidInput(t *testing.T) {
	raw := bytes.NewReader(rawTestAudio)

	got, err := Flac(raw)
	if err != nil {
		t.Errorf("Flac() error = %v", err)
		return
	}

	if got == nil {
		t.Error("Flac() returned nil buffer")
		return
	}

	if got.Len() == 0 {
		t.Error("Flac() returned empty buffer")
		return
	}

	// Verify FLAC magic bytes (fLaC)
	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Errorf("Flac() output does not have FLAC magic header, got %v", got.Bytes()[:4])
	}
}

func TestFlac_EmptyInput(t *testing.T) {
	raw := bytes.NewReader([]byte{})

	got, err := Flac(raw)
	if err != nil {
		t.Errorf("Flac() error = %v", err)
		return
	}

	if got == nil {
		t.Error("Flac() returned nil buffer")
		return
	}

	// Empty input should produce valid FLAC with just headers
	if got.Len() > 0 {
		flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
		if !bytes.HasPrefix(got.Bytes(), flacMagic) {
			t.Errorf("Flac() output does not have FLAC magic header, got %v", got.Bytes()[:min(4, got.Len())])
		}
	}
}

// TestFlac_DeterministicOutput verifies the FLAC encoder produces
// identical output for the same input across two runs.
func TestFlac_DeterministicOutput(t *testing.T) {
	first, err := Flac(bytes.NewReader(rawTestAudio))
	if err != nil {
		t.Fatalf("Flac() first run error = %v", err)
	}

	second, err := Flac(bytes.NewReader(rawTestAudio))
	if err != nil {
		t.Fatalf("Flac() second run error = %v", err)
	}

	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("Flac output is not deterministic: first %d bytes, second %d bytes", first.Len(), second.Len())
	}
}

func TestFlac_SmallInput(t *testing.T) {
	// Create minimal stereo 16-bit samples (32 samples = 16 per channel)
	samples := make([]byte, 64) // 32 samples * 2 bytes
	for i := range samples {
		samples[i] = byte(i)
	}
	raw := bytes.NewReader(samples)

	got, err := Flac(raw)
	if err != nil {
		t.Errorf("Flac() error = %v", err)
		return
	}

	if got == nil {
		t.Error("Flac() returned nil buffer")
		return
	}

	// Verify FLAC magic bytes
	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Errorf("Flac() output does not have FLAC magic header")
	}
}
