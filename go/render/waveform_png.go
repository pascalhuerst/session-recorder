package render

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
)

// PreviewWidth and PreviewHeight are the fixed dimensions of the session
// overview thumbnail that storage backends render via WaveformPNG.
const (
	PreviewWidth  = 600
	PreviewHeight = 80
)

// waveformCenterBrightness is the fraction of the full fill color used at the
// 0 line (vertical center); the fill ramps to full color toward the top/bottom.
const waveformCenterBrightness = 0.6

// waveformFill is the slate-400 the frontend overview waveform uses (#94a3b8).
var waveformFill = color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff}

// waveformBackground is a dark gray (transparent) behind the waveform.
var waveformBackground = color.NRGBA{R: 0x2d, G: 0x2d, B: 0x2d, A: 0x0}

// WaveformPNG renders an audiowaveform ".dat" (as produced by CreateWaveform)
// into a width×height PNG that mirrors the frontend overview waveform: the
// min/max envelope filled in #94a3b8 on a dark gray background, centered
// vertically. The .dat's pixel columns are resampled to the requested width, so
// the result is a fixed-size preview thumbnail regardless of recording length.
func WaveformPNG(dat io.Reader, width, height int) (*bytes.Buffer, error) {
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("invalid dimensions %dx%d", width, height)
	}

	mins, maxs, err := parseDat(dat)
	if err != nil {
		return nil, err
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{waveformBackground}, image.Point{}, draw.Src)
	n := len(mins)
	mid := float64(height-1) / 2.0

	// Precompute the fill color per row: darker at the 0 line, ramping to the
	// full color at the top/bottom edges (a subtle vertical gradient).
	rowColor := make([]color.NRGBA, height)
	for y := range height {
		dist := 0.0
		if mid > 0 {
			dist = math.Abs(float64(y)-mid) / mid // 0 at center → 1 at edges
			if dist > 1 {
				dist = 1
			}
		}
		f := waveformCenterBrightness + (1-waveformCenterBrightness)*dist
		rowColor[y] = color.NRGBA{
			R: uint8(float64(waveformFill.R) * f),
			G: uint8(float64(waveformFill.G) * f),
			B: uint8(float64(waveformFill.B) * f),
			A: 0xff,
		}
	}

	for x := range width {
		// Envelope for this output column: the overall min/max of the input
		// pixels it covers (normalized to [-1, 1]). hi >= 0 >= lo for real audio.
		var lo, hi float64
		if n > 0 {
			start := x * n / width
			end := (x + 1) * n / width
			if end <= start {
				end = start + 1
			}
			if end > n {
				end = n
			}
			lo, hi = mins[start], maxs[start]
			for i := start + 1; i < end; i++ {
				if mins[i] < lo {
					lo = mins[i]
				}
				if maxs[i] > hi {
					hi = maxs[i]
				}
			}
		}

		// y grows downward; the positive peak (hi) sits near the top.
		yTop := int(mid - hi*mid + 0.5)
		yBot := int(mid - lo*mid + 0.5)
		if yTop > yBot {
			yTop, yBot = yBot, yTop
		}
		if yTop < 0 {
			yTop = 0
		}
		if yBot > height-1 {
			yBot = height - 1
		}
		for y := yTop; y <= yBot; y++ {
			img.SetNRGBA(x, y, rowColor[y])
		}
	}

	out := new(bytes.Buffer)
	if err := png.Encode(out, img); err != nil {
		return nil, err
	}
	return out, nil
}

// parseDat reads an audiowaveform .dat (the format CreateWaveform emits) and
// returns the per-pixel min/max envelopes normalized to [-1, 1]. The whole file
// is read into memory — a .dat is tiny relative to the audio (~16 MB even for a
// 12 h 8-bit recording).
func parseDat(r io.Reader) (mins, maxs []float64, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read dat: %w", err)
	}
	// Header: version, flags, sampleRate, samplesPerPixel, length (5×4 bytes).
	if len(data) < 20 {
		return nil, nil, fmt.Errorf("dat too short (%d bytes)", len(data))
	}
	version := int32(binary.LittleEndian.Uint32(data[0:]))
	flags := binary.LittleEndian.Uint32(data[4:])
	length := int(binary.LittleEndian.Uint32(data[16:]))

	off := 20
	if version >= 2 {
		off = 24 // version 2 adds a channels field
	}

	eightBit := flags&0x1 != 0
	bytesPerSample := 2
	if eightBit {
		bytesPerSample = 1
	}
	if len(data) < off+length*2*bytesPerSample {
		return nil, nil, fmt.Errorf("dat truncated: have %d bytes, need %d", len(data), off+length*2*bytesPerSample)
	}

	mins = make([]float64, length)
	maxs = make([]float64, length)
	for i := range length {
		if eightBit {
			p := off + i*2
			mins[i] = float64(int8(data[p])) / 128.0
			maxs[i] = float64(int8(data[p+1])) / 128.0
		} else {
			p := off + i*4
			mins[i] = float64(int16(binary.LittleEndian.Uint16(data[p:]))) / 32768.0
			maxs[i] = float64(int16(binary.LittleEndian.Uint16(data[p+2:]))) / 32768.0
		}
	}
	return mins, maxs, nil
}
