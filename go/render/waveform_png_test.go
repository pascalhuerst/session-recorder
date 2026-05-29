package render

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// envelopedSine builds interleaved s16le/2ch/48k PCM of a 1 kHz tone whose
// amplitude swells and fades, so the rendered waveform has a recognizable shape
// rather than a flat block.
func envelopedSine(seconds float64) []byte {
	nFrames := int(seconds * sampleRate)
	buf := bytes.NewBuffer(make([]byte, 0, nFrames*bytesPerFrame))
	for i := range nFrames {
		t := float64(i) / float64(sampleRate)
		env := 0.15 + 0.85*math.Abs(math.Sin(2*math.Pi*0.4*t)) // slow swell
		v := int16(env * 30000 * math.Sin(2*math.Pi*1000*t))
		for range numChannels {
			binary.Write(buf, binary.LittleEndian, v)
		}
	}
	return buf.Bytes()
}

func TestWaveformPNG(t *testing.T) {
	raw := envelopedSine(4)

	dat, err := CreateWaveform(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("CreateWaveform: %v", err)
	}

	const width, height = 600, 80
	pngBuf, err := WaveformPNG(bytes.NewReader(dat.Bytes()), width, height)
	if err != nil {
		t.Fatalf("WaveformPNG: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBuf.Bytes()))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != width || b.Dy() != height {
		t.Fatalf("size = %dx%d, want %dx%d", b.Dx(), b.Dy(), width, height)
	}

	// A waveform pixel is a slate tint (any brightness along the gradient):
	// #94a3b8 keeps B > G > R, whereas the gray background has R == G == B.
	isWave := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		return b>>8 > g>>8 && g>>8 > r>>8
	}

	// Every column should have at least one filled pixel, and the swell means
	// some columns are clearly taller than others (not a flat block).
	minRun, maxRun := height+1, 0
	for x := range width {
		run := 0
		for y := range height {
			if isWave(x, y) {
				run++
			}
		}
		if run == 0 {
			t.Fatalf("column %d has no waveform pixels", x)
		}
		if run < minRun {
			minRun = run
		}
		if run > maxRun {
			maxRun = run
		}
	}
	if maxRun <= minRun {
		t.Fatalf("waveform has no amplitude variation (min run %d, max run %d)", minRun, maxRun)
	}

	// Write the PNG out so it can be eyeballed against the frontend overview.
	outPath := filepath.Join(os.TempDir(), "waveform_preview.png")
	if err := os.WriteFile(outPath, pngBuf.Bytes(), 0o644); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	t.Logf("wrote %dx%d preview (%d bytes, runs %d..%d px) to %s",
		width, height, pngBuf.Len(), minRun, maxRun, outPath)
}
