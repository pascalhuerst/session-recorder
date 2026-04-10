package render

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

/**
 * Test Plan: Sox Audio File Streaming
 *
 * Scenario: Stream raw audio to FLAC format
 *   Given raw PCM audio data and sox is available
 *   When CreateAudioFileStream is called with "flac" format
 *   Then valid FLAC data with fLaC magic header is written to the output
 *
 * Scenario: Stream raw audio to OGG format
 *   Given raw PCM audio data and sox is available
 *   When CreateAudioFileStream is called with "ogg" format
 *   Then valid OGG data with OggS magic header is written to the output
 *
 * Scenario: Handle empty input
 *   Given an empty reader and sox is available
 *   When CreateAudioFileStream is called
 *   Then sox either returns an error or produces a valid FLAC header
 *
 * Scenario: Handle concurrent streams
 *   Given multiple raw PCM readers and sox is available
 *   When CreateAudioFileStream is called concurrently
 *   Then all streams produce valid output without interference
 *
 * Scenario: Handle large input
 *   Given a large raw PCM input and sox is available
 *   When CreateAudioFileStream is called
 *   Then the output is valid and larger than the small-input case
 */

func TestCreateAudioFileStream_ToFlac(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	raw := bytes.NewReader(rawTestAudio)
	var buf bytes.Buffer

	err := CreateAudioFileStream(raw, "flac", &buf)
	if err != nil {
		t.Fatalf("CreateAudioFileStream() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("CreateAudioFileStream() produced empty output")
	}

	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	if !bytes.HasPrefix(buf.Bytes(), flacMagic) {
		t.Errorf("output does not have FLAC magic header, got %v", buf.Bytes()[:min(4, buf.Len())])
	}
}

func TestCreateAudioFileStream_ToOgg(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	raw := bytes.NewReader(rawTestAudio)
	var buf bytes.Buffer

	err := CreateAudioFileStream(raw, "ogg", &buf)
	if err != nil {
		t.Fatalf("CreateAudioFileStream() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("CreateAudioFileStream() produced empty output")
	}

	oggMagic := []byte{0x4F, 0x67, 0x67, 0x53}
	if !bytes.HasPrefix(buf.Bytes(), oggMagic) {
		t.Errorf("output does not have OGG magic header, got %v", buf.Bytes()[:min(4, buf.Len())])
	}
}

func TestCreateAudioFileStream_EmptyInput(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	raw := bytes.NewReader(nil)
	var buf bytes.Buffer

	err := CreateAudioFileStream(raw, "flac", &buf)

	// Sox may return an error for empty input — that's acceptable.
	// We document the behavior: either an error or a valid FLAC header.
	if err != nil {
		t.Logf("CreateAudioFileStream() returned error for empty input (expected): %v", err)
		return
	}

	if buf.Len() >= 4 {
		flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
		if !bytes.HasPrefix(buf.Bytes(), flacMagic) {
			t.Errorf("non-error output does not have FLAC magic header, got %v", buf.Bytes()[:4])
		}
	}
}

func TestCreateAudioFileStream_ConcurrentStreams(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	const numStreams = 3
	var wg sync.WaitGroup
	errs := make([]error, numStreams)
	bufs := make([]*bytes.Buffer, numStreams)

	for i := 0; i < numStreams; i++ {
		bufs[i] = &bytes.Buffer{}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			raw := bytes.NewReader(rawTestAudio)
			errs[idx] = CreateAudioFileStream(raw, "flac", bufs[idx])
		}(i)
	}

	wg.Wait()

	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	for i := 0; i < numStreams; i++ {
		if errs[i] != nil {
			t.Errorf("stream %d: CreateAudioFileStream() error = %v", i, errs[i])
			continue
		}
		if bufs[i].Len() == 0 {
			t.Errorf("stream %d: produced empty output", i)
			continue
		}
		if !bytes.HasPrefix(bufs[i].Bytes(), flacMagic) {
			t.Errorf("stream %d: output does not have FLAC magic header, got %v", i, bufs[i].Bytes()[:min(4, bufs[i].Len())])
		}
	}
}

func TestCreateAudioFileStream_LargeInput(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	// Get baseline output size with single copy
	rawSingle := bytes.NewReader(rawTestAudio)
	var smallBuf bytes.Buffer
	err := CreateAudioFileStream(rawSingle, "flac", &smallBuf)
	if err != nil {
		t.Fatalf("baseline CreateAudioFileStream() error = %v", err)
	}

	// Create large input by repeating rawTestAudio 5 times
	readers := make([]io.Reader, 5)
	for i := range readers {
		readers[i] = bytes.NewReader(rawTestAudio)
	}
	largeRaw := io.MultiReader(readers...)

	var largeBuf bytes.Buffer
	err = CreateAudioFileStream(largeRaw, "flac", &largeBuf)
	if err != nil {
		t.Fatalf("large CreateAudioFileStream() error = %v", err)
	}

	if largeBuf.Len() == 0 {
		t.Fatal("large input produced empty output")
	}

	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	if !bytes.HasPrefix(largeBuf.Bytes(), flacMagic) {
		t.Errorf("large output does not have FLAC magic header, got %v", largeBuf.Bytes()[:min(4, largeBuf.Len())])
	}

	if largeBuf.Len() <= smallBuf.Len() {
		t.Errorf("large input output (%d bytes) should be larger than small input output (%d bytes)", largeBuf.Len(), smallBuf.Len())
	}
}
