// Package audiowaveform generates BBC audiowaveform ".dat" files directly in
// Go, replacing the external `audiowaveform` binary for the one configuration
// the recorder uses: raw interleaved S16LE input, mono output, 8-bit samples.
//
// It is a faithful, minimal reimplementation of the relevant parts of
// audiowaveform's WaveformGenerator + WaveformBuffer (see the vendored C++ in
// ../../audiowaveform/src). Only what we need is implemented; there is no PNG
// rendering, no resampling, no multi-format input.
//
// Algorithm (mono, no --split-channels): each input frame is mixed to a single
// sample as sum(channels)/channels (integer division, clamped to int16). Over
// each `SamplesPerPixel` frames the min and max are tracked and emitted as one
// pixel; a trailing partial bucket is emitted too.
package audiowaveform

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	maxSample = math.MaxInt16 // 32767
	minSample = math.MinInt16 // -32768

	flag8Bit = uint32(0x00000001)
)

// Params describes the (fixed-ish) input/output configuration. Defaults match
// the recorder's waveform.dat invocation: 48 kHz, 2 channels, zoom 256, 8-bit.
type Params struct {
	SampleRate      int // input sample rate, e.g. 48000
	Channels        int // input channels, e.g. 2
	SamplesPerPixel int // "zoom", e.g. 256 (minimum 2)
	Bits            int // output resolution: 8 or 16
}

// DefaultParams mirrors render.CreateWaveform's audiowaveform arguments.
func DefaultParams() Params {
	return Params{SampleRate: 48000, Channels: 2, SamplesPerPixel: 256, Bits: 8}
}

// GenerateDat streams interleaved little-endian S16 samples from r and writes
// an audiowaveform v1 (mono) .dat to a buffer. The output is byte-for-byte
// identical to `audiowaveform --input-format raw --raw-format s16le
// --output-format dat --zoom <spp> -b <bits>` for the same input.
func GenerateDat(r io.Reader, p Params) (*bytes.Buffer, error) {
	if p.Channels < 1 {
		return nil, fmt.Errorf("invalid channels: %d", p.Channels)
	}
	if p.SamplesPerPixel < 2 {
		return nil, fmt.Errorf("invalid samples-per-pixel: %d (minimum 2)", p.SamplesPerPixel)
	}
	if p.Bits != 8 && p.Bits != 16 {
		return nil, fmt.Errorf("invalid bits: %d (must be 8 or 16)", p.Bits)
	}

	// Output is mono (we never split channels), so each pixel is one (min,max).
	mins := make([]int16, 0, 4096)
	maxs := make([]int16, 0, 4096)

	curMin := int32(maxSample)
	curMax := int32(minSample)
	count := 0

	flush := func() {
		mins = append(mins, int16(curMin))
		maxs = append(maxs, int16(curMax))
		curMin = maxSample
		curMax = minSample
		count = 0
	}

	br := bufio.NewReaderSize(r, 64*1024)
	frame := make([]byte, p.Channels*2) // one interleaved frame, 2 bytes/sample

	for {
		// Read exactly one frame; tolerate a trailing partial frame by stopping.
		if _, err := io.ReadFull(br, frame); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("cannot read samples: %w", err)
		}

		// Mix the frame's channels to a single mono sample (sum / channels),
		// matching WaveformGenerator's integer-division behaviour.
		var sum int32
		for ch := 0; ch < p.Channels; ch++ {
			sum += int32(int16(binary.LittleEndian.Uint16(frame[ch*2:])))
		}
		sample := sum / int32(p.Channels)
		if sample > maxSample {
			sample = maxSample
		} else if sample < minSample {
			sample = minSample
		}

		if sample < curMin {
			curMin = sample
		}
		if sample > curMax {
			curMax = sample
		}

		count++
		if count == p.SamplesPerPixel {
			flush()
		}
	}

	// Trailing partial bucket (matches WaveformGenerator::done()).
	if count > 0 {
		flush()
	}

	return writeDat(mins, maxs, p)
}

func writeDat(mins, maxs []int16, p Params) (*bytes.Buffer, error) {
	const channels = 1 // mono output → file version 1, no channels field
	size := len(mins)

	buf := new(bytes.Buffer)
	w := func(v any) {
		// bytes.Buffer never errors; binary.Write only errors on bad types.
		_ = binary.Write(buf, binary.LittleEndian, v)
	}

	w(int32(1)) // version
	flags := uint32(0)
	if p.Bits == 8 {
		flags |= flag8Bit
	}
	w(flags)
	w(int32(p.SampleRate))
	w(int32(p.SamplesPerPixel))
	w(uint32(size))
	// version 1 (mono) has no channels field.

	if p.Bits == 8 {
		for i := range size {
			w(int8(mins[i] / 256))
			w(int8(maxs[i] / 256))
		}
	} else {
		for i := range size {
			w(mins[i])
			w(maxs[i])
		}
	}

	_ = channels
	return buf, nil
}
