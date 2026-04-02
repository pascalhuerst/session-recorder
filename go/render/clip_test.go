package render

import (
	"bytes"
	"testing"
	"time"
)

/**
 * Test Plan: Audio Clipping
 *
 * Scenario: Convert sample position to duration
 *   Given a sample position at 48000 (1 second worth at 48kHz)
 *   When SamplePositionToDuration is called
 *   Then a duration of 1 second is returned
 *
 * Scenario: Clip and encode raw audio to OGG format
 *   Given raw PCM audio data and valid start/end positions
 *   When ClipAndEncodeOgg is called
 *   Then a valid OGG buffer is returned with OggS magic header
 *
 * Scenario: Clip and encode raw audio to FLAC format
 *   Given raw PCM audio data and valid start/end positions
 *   When ClipAndEncodeFlac is called
 *   Then a valid FLAC buffer is returned with fLaC magic header
 *
 * Scenario: Handle invalid segment range
 *   Given an end position less than or equal to start position
 *   When clip function is called
 *   Then an error is returned
 */

func TestSamplePositionToDuration(t *testing.T) {
	tests := []struct {
		name         string
		samplePos    int64
		wantDuration time.Duration
	}{
		{
			name:         "zero position",
			samplePos:    0,
			wantDuration: 0,
		},
		{
			name:         "one second (48000 samples)",
			samplePos:    48000,
			wantDuration: time.Second,
		},
		{
			name:         "half second (24000 samples)",
			samplePos:    24000,
			wantDuration: 500 * time.Millisecond,
		},
		{
			name:         "10 seconds (480000 samples)",
			samplePos:    480000,
			wantDuration: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SamplePositionToDuration(tt.samplePos)
			if got != tt.wantDuration {
				t.Errorf("SamplePositionToDuration() = %v, want %v", got, tt.wantDuration)
			}
		})
	}
}

func TestClipAndEncodeOgg(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	// Use embedded test data - clip from 0.5s to 1s
	// rawTestAudio is s16le, 2ch, 48kHz
	raw := bytes.NewReader(rawTestAudio)

	// Clip from 0.5s (24000 samples) to 1s (48000 samples)
	startPos := int64(24000)
	endPos := int64(48000)

	got, err := ClipAndEncodeOgg(raw, startPos, endPos)
	if err != nil {
		t.Errorf("ClipAndEncodeOgg() error = %v", err)
		return
	}

	if got == nil {
		t.Error("ClipAndEncodeOgg() returned nil buffer")
		return
	}

	if got.Len() == 0 {
		t.Error("ClipAndEncodeOgg() returned empty buffer")
		return
	}

	// OGG files start with "OggS"
	oggMagic := []byte{0x4F, 0x67, 0x67, 0x53}
	if !bytes.HasPrefix(got.Bytes(), oggMagic) {
		t.Errorf("ClipAndEncodeOgg() output does not have OGG magic header, got %v", got.Bytes()[:min(4, got.Len())])
	}

	// A 0.5s clip should be much smaller than the full 30s OGG encoding
	if got.Len() < 50 {
		t.Errorf("Clipped OGG suspiciously small: %d bytes", got.Len())
	}

	// Also encode the full file to compare sizes
	fullRaw := bytes.NewReader(rawTestAudio)
	fullOgg, err := CreateAudioFile(fullRaw, "ogg")
	if err != nil {
		t.Fatalf("CreateAudioFile() for full OGG failed: %v", err)
	}
	// A 0.5s clip of 30s audio should be roughly 60x smaller, with generous margin
	if got.Len() >= fullOgg.Len() {
		t.Errorf("Clipped OGG (%d bytes) should be smaller than full OGG (%d bytes)", got.Len(), fullOgg.Len())
	}
}

func TestClipAndEncodeFlac(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	// Use embedded test data - clip from 0.5s to 1s
	raw := bytes.NewReader(rawTestAudio)

	// Clip from 0.5s (24000 samples) to 1s (48000 samples)
	startPos := int64(24000)
	endPos := int64(48000)

	got, err := ClipAndEncodeFlac(raw, startPos, endPos)
	if err != nil {
		t.Errorf("ClipAndEncodeFlac() error = %v", err)
		return
	}

	if got == nil {
		t.Error("ClipAndEncodeFlac() returned nil buffer")
		return
	}

	if got.Len() == 0 {
		t.Error("ClipAndEncodeFlac() returned empty buffer")
		return
	}

	// FLAC files start with "fLaC"
	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Errorf("ClipAndEncodeFlac() output does not have FLAC magic header, got %v", got.Bytes()[:min(4, got.Len())])
	}

	// A 0.5s FLAC clip should be much smaller than the 30s raw input
	if got.Len() < 50 {
		t.Errorf("Clipped FLAC suspiciously small: %d bytes", got.Len())
	}
	// 0.5s of raw = 48000 samples/s * 0.5s * 2ch * 2bytes = 96000 bytes
	// FLAC should be at most that size (lossless)
	maxExpected := 96000 * 2 // generous margin
	if got.Len() > maxExpected {
		t.Errorf("Clipped FLAC (%d bytes) larger than expected max (%d bytes) for 0.5s clip", got.Len(), maxExpected)
	}
}

func TestClipAndEncode_InvalidRange(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	raw := bytes.NewReader(rawTestAudio)

	// End before start should fail
	_, err := ClipAndEncodeOgg(raw, 48000, 24000)
	if err == nil {
		t.Error("ClipAndEncodeOgg() expected error for invalid range, got nil")
	}
}

func TestClipAndEncode_ZeroDuration(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	raw := bytes.NewReader(rawTestAudio)

	// Same start and end should fail
	_, err := ClipAndEncodeOgg(raw, 48000, 48000)
	if err == nil {
		t.Error("ClipAndEncodeOgg() expected error for zero duration, got nil")
	}
}
