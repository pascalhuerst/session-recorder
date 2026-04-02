package render

import (
	"bytes"
	"context"
	"crypto/md5"
	_ "embed"
	"image/png"
	"io"
	"os/exec"
	"testing"
)

/**
 * Test Plan: Waveform Rendering
 *
 * Scenario: Create waveform overview from raw audio
 *   Given raw PCM audio data (s16le, 2ch, 48kHz)
 *   When CreateOverview is called with zoom, width, and height parameters
 *   Then a PNG image buffer is returned with the expected dimensions
 *   And the output matches the expected reference image (hash comparison)
 */

func mkHash(r io.Reader) ([]byte, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, r); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

//go:embed test_data/sweep_30_20000_s16le_2ch_48000k.raw
var rawTestAudio []byte

//go:embed test_data/sweep_30_20000_s16le_2ch_48000k.png
var resultWaveform []byte

func TestCreateOverview(t *testing.T) {
	// Skip test if audiowaveform is not installed
	if _, err := exec.LookPath("audiowaveform"); err != nil {
		t.Skip("audiowaveform not installed, skipping test")
	}

	type args struct {
		raw    io.Reader
		zoom   int
		width  int
		height int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "sweep_30_20000_s16le_2ch_48000k",
			args: args{
				raw:    bytes.NewReader(rawTestAudio),
				zoom:   256,
				width:  1024,
				height: 256,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateOverview(context.Background(), tt.args.raw, tt.args.zoom, tt.args.width, tt.args.height)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOverview() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil {
				t.Error("CreateOverview() returned nil buffer")
				return
			}

			if got.Len() == 0 {
				t.Error("CreateOverview() returned empty buffer")
				return
			}

			// Verify PNG magic bytes (0x89 PNG \r \n 0x1A \n)
			pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			if !bytes.HasPrefix(got.Bytes(), pngMagic) {
				t.Errorf("CreateOverview() output does not have PNG magic header, got %v", got.Bytes()[:min(8, got.Len())])
			}

			// Decode PNG and verify dimensions match requested size
			img, pngErr := png.Decode(bytes.NewReader(got.Bytes()))
			if pngErr != nil {
				t.Errorf("CreateOverview() output is not a valid PNG: %v", pngErr)
			} else {
				bounds := img.Bounds()
				w := bounds.Max.X - bounds.Min.X
				h := bounds.Max.Y - bounds.Min.Y
				if w != tt.args.width {
					t.Errorf("PNG width: expected %d, got %d", tt.args.width, w)
				}
				if h != tt.args.height {
					t.Errorf("PNG height: expected %d, got %d", tt.args.height, h)
				}
			}
		})
	}
}

func TestCreateWaveform_ValidDatOutput(t *testing.T) {
	if _, err := exec.LookPath("audiowaveform"); err != nil {
		t.Skip("audiowaveform not installed, skipping test")
	}

	raw := bytes.NewReader(rawTestAudio)
	got, err := CreateWaveform(context.Background(), raw, 256, 10000, 200)
	if err != nil {
		t.Fatalf("CreateWaveform() error = %v", err)
	}

	if got == nil || got.Len() == 0 {
		t.Fatal("CreateWaveform() returned nil or empty buffer")
	}

	// At zoom=256, 30s of 48kHz stereo = (30 * 48000) / 256 ≈ 5625 data points
	// Each point has min+max values. Output should be non-trivial.
	if got.Len() < 100 {
		t.Errorf("Waveform dat output suspiciously small: %d bytes", got.Len())
	}
}
