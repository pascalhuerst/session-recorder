package render

import (
	"bytes"
	"testing"
)

/**
 * Test Plan: FLAC Encoding
 *
 * Scenario: Encode valid raw audio to FLAC
 *   Given raw PCM audio data (s16le, 2ch, 48kHz)
 *   When Flac() is called
 *   Then a valid FLAC stream is written with the fLaC magic header
 *
 * Scenario: Handle empty input gracefully
 *   Given an empty reader as input
 *   When Flac() is called
 *   Then a valid FLAC stream with headers is written (no panic)
 *
 * Scenario: Encode small audio samples
 *   Given minimal stereo 16-bit samples
 *   When Flac() is called
 *   Then a valid FLAC stream is written with the fLaC magic header
 */

var flacMagic = []byte{0x66, 0x4C, 0x61, 0x43}

func TestFlac_ValidInput(t *testing.T) {
	var got bytes.Buffer
	if err := Flac(&got, bytes.NewReader(rawTestAudio)); err != nil {
		t.Fatalf("Flac() error = %v", err)
	}
	if got.Len() == 0 {
		t.Fatal("Flac() wrote empty output")
	}
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Errorf("Flac() output does not have FLAC magic header, got %v", got.Bytes()[:4])
	}
}

func TestFlac_EmptyInput(t *testing.T) {
	var got bytes.Buffer
	if err := Flac(&got, bytes.NewReader([]byte{})); err != nil {
		t.Fatalf("Flac() error = %v", err)
	}
	// Empty input still produces a valid FLAC header.
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Errorf("Flac() output does not have FLAC magic header, got %v", got.Bytes()[:min(4, got.Len())])
	}
}

func TestFlac_SmallInput(t *testing.T) {
	// Minimal stereo 16-bit samples (16 frames = 64 bytes).
	samples := make([]byte, 64)
	for i := range samples {
		samples[i] = byte(i)
	}

	var got bytes.Buffer
	if err := Flac(&got, bytes.NewReader(samples)); err != nil {
		t.Fatalf("Flac() error = %v", err)
	}
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Error("Flac() output does not have FLAC magic header")
	}
}
