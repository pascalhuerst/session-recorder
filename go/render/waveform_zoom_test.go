package render

import (
	"bytes"
	"context"
	"os/exec"
	"testing"
)

/**
 * Test Plan: CreateWaveformStream uses zoom parameter
 *
 * Scenario: Different zoom values produce different output sizes
 *   Given the same raw PCM audio input
 *   When CreateWaveform is called with zoom=128 and zoom=512
 *   Then zoom=128 produces more data points (larger output) than zoom=512
 *   Because lower zoom means more samples per pixel, resulting in more data points
 */

func TestCreateWaveform_ZoomAffectsOutputSize(t *testing.T) {
	if _, err := exec.LookPath("audiowaveform"); err != nil {
		t.Skip("audiowaveform not installed, skipping test")
	}

	// Use zoom=128 (more data points) and zoom=512 (fewer data points)
	smallZoom := 128
	largeZoom := 512

	gotSmall, err := CreateWaveform(context.Background(), bytes.NewReader(rawTestAudio), smallZoom, 10000, 200)
	if err != nil {
		t.Fatalf("CreateWaveform(zoom=%d) error = %v", smallZoom, err)
	}

	gotLarge, err := CreateWaveform(context.Background(), bytes.NewReader(rawTestAudio), largeZoom, 10000, 200)
	if err != nil {
		t.Fatalf("CreateWaveform(zoom=%d) error = %v", largeZoom, err)
	}

	if gotSmall.Len() == 0 || gotLarge.Len() == 0 {
		t.Fatal("CreateWaveform returned empty output")
	}

	// Lower zoom = more data points = larger output
	if gotSmall.Len() <= gotLarge.Len() {
		t.Errorf("zoom=%d should produce more data than zoom=%d, but got %d <= %d bytes",
			smallZoom, largeZoom, gotSmall.Len(), gotLarge.Len())
	}
}
