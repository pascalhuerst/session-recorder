package render

import (
	"bytes"
	"io"
	"testing"
	"time"
)

/**
 * Test Plan: Segment extraction
 *
 * Scenario: Convert sample position to duration
 *   Given a sample position at 48000 (1 second worth at 48kHz)
 *   When SamplePositionToDuration is called
 *   Then a duration of 1 second is returned
 *
 * Scenario: SegmentReader yields exactly the requested frame window
 *   Given raw PCM and a [start, end) frame range
 *   When SegmentReader is read fully
 *   Then the bytes equal raw[start*frame : end*frame]
 *
 * Scenario: SegmentReader streams without buffering the whole window
 *   Given a window larger than any single read
 *   Then it can be consumed incrementally and bounds the length
 *
 * Scenario: SegmentReader feeds the streaming encoders
 *   Given a window of raw audio
 *   When piped into Opus / Flac
 *   Then valid OGG / FLAC streams are produced
 *
 * Scenario: Invalid range is rejected
 *   Given end <= start
 *   When SegmentReader is called
 *   Then an error is returned
 */

func TestSamplePositionToDuration(t *testing.T) {
	tests := []struct {
		name         string
		samplePos    int64
		wantDuration time.Duration
	}{
		{"zero position", 0, 0},
		{"one second (48000 samples)", 48000, time.Second},
		{"half second (24000 samples)", 24000, 500 * time.Millisecond},
		{"10 seconds (480000 samples)", 480000, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SamplePositionToDuration(tt.samplePos); got != tt.wantDuration {
				t.Errorf("SamplePositionToDuration() = %v, want %v", got, tt.wantDuration)
			}
		})
	}
}

func TestSegmentReader_ExactWindow(t *testing.T) {
	raw := genSineRaw(1000, 1) // 1s
	startPos := int64(24000)   // 0.5s
	endPos := int64(36000)     // 0.75s

	r, err := SegmentReader(bytes.NewReader(raw), startPos, endPos)
	if err != nil {
		t.Fatalf("SegmentReader() error = %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}

	want := raw[startPos*bytesPerFrame : endPos*bytesPerFrame]
	if !bytes.Equal(got, want) {
		t.Errorf("segment content does not match raw[start:end] window (got %d bytes, want %d)", len(got), len(want))
	}
}

func TestSegmentReader_ShortStreamClamps(t *testing.T) {
	raw := genSineRaw(1000, 1) // 1s == 48000 frames

	// Ask for a window that runs past the end of the stream.
	r, err := SegmentReader(bytes.NewReader(raw), 24000, 96000)
	if err != nil {
		t.Fatalf("SegmentReader() error = %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}

	// Only frames [24000, 48000) exist.
	wantLen := int((48000 - 24000) * bytesPerFrame)
	if len(got) != wantLen {
		t.Fatalf("clamped segment len = %d, want %d", len(got), wantLen)
	}
}

func TestSegmentReader_EncodesOgg(t *testing.T) {
	skipIfRace(t)

	r, err := SegmentReader(bytes.NewReader(rawTestAudio), 24000, 48000)
	if err != nil {
		t.Fatalf("SegmentReader() error = %v", err)
	}
	var out bytes.Buffer
	if err := Opus(&out, r); err != nil {
		t.Fatalf("Opus() error = %v", err)
	}
	if !bytes.HasPrefix(out.Bytes(), []byte("OggS")) {
		t.Errorf("segment OGG missing OggS magic, got %v", out.Bytes()[:min(4, out.Len())])
	}
}

func TestSegmentReader_EncodesFlac(t *testing.T) {
	r, err := SegmentReader(bytes.NewReader(rawTestAudio), 24000, 48000)
	if err != nil {
		t.Fatalf("SegmentReader() error = %v", err)
	}
	var out bytes.Buffer
	if err := Flac(&out, r); err != nil {
		t.Fatalf("Flac() error = %v", err)
	}
	if !bytes.HasPrefix(out.Bytes(), flacMagic) {
		t.Errorf("segment FLAC missing fLaC magic, got %v", out.Bytes()[:min(4, out.Len())])
	}
}

func TestSegmentReader_InvalidRange(t *testing.T) {
	if _, err := SegmentReader(bytes.NewReader(rawTestAudio), 48000, 24000); err == nil {
		t.Error("SegmentReader() expected error for end < start, got nil")
	}
	if _, err := SegmentReader(bytes.NewReader(rawTestAudio), 48000, 48000); err == nil {
		t.Error("SegmentReader() expected error for zero duration, got nil")
	}
}
