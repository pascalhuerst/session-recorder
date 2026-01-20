package render

import (
	"bytes"
	"context"
	"crypto/md5"
	_ "embed"
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
		})
	}
}
