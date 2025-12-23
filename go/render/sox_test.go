package render

import (
	"bytes"
	"os/exec"
	"testing"
)

func soxAvailable() bool {
	_, err := exec.LookPath("/usr/bin/sox")
	return err == nil
}

func TestCreateAudioFile_ToOgg(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	// Use embedded test data
	raw := bytes.NewReader(rawTestAudio)

	got, err := CreateAudioFile(raw, "ogg")
	if err != nil {
		t.Errorf("CreateAudioFile() error = %v", err)
		return
	}

	if got == nil {
		t.Error("CreateAudioFile() returned nil buffer")
		return
	}

	if got.Len() == 0 {
		t.Error("CreateAudioFile() returned empty buffer")
		return
	}

	// OGG files start with "OggS"
	oggMagic := []byte{0x4F, 0x67, 0x67, 0x53}
	if !bytes.HasPrefix(got.Bytes(), oggMagic) {
		t.Errorf("CreateAudioFile() output does not have OGG magic header, got %v", got.Bytes()[:min(4, got.Len())])
	}
}

func TestCreateAudioFile_ToFlac(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	// Use embedded test data
	raw := bytes.NewReader(rawTestAudio)

	got, err := CreateAudioFile(raw, "flac")
	if err != nil {
		t.Errorf("CreateAudioFile() error = %v", err)
		return
	}

	if got == nil {
		t.Error("CreateAudioFile() returned nil buffer")
		return
	}

	if got.Len() == 0 {
		t.Error("CreateAudioFile() returned empty buffer")
		return
	}

	// FLAC files start with "fLaC"
	flacMagic := []byte{0x66, 0x4C, 0x61, 0x43}
	if !bytes.HasPrefix(got.Bytes(), flacMagic) {
		t.Errorf("CreateAudioFile() output does not have FLAC magic header, got %v", got.Bytes()[:min(4, got.Len())])
	}
}

func TestCreateAudioFile_SmallInput(t *testing.T) {
	if !soxAvailable() {
		t.Skip("sox not available, skipping test")
	}

	// Create minimal stereo 16-bit samples
	samples := make([]byte, 1024) // Some samples
	for i := range samples {
		samples[i] = byte(i % 256)
	}
	raw := bytes.NewReader(samples)

	got, err := CreateAudioFile(raw, "ogg")
	if err != nil {
		t.Errorf("CreateAudioFile() error = %v", err)
		return
	}

	if got == nil {
		t.Error("CreateAudioFile() returned nil buffer")
		return
	}
}
