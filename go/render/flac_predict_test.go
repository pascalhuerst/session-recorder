package render

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
)

/**
 * Test Plan: FLAC compression must stay lossless and must actually compress
 *
 * Scenario: Every kind of signal round-trips bit-exactly
 *   Given signals that each steer the encoder down a different path
 *     (constant, fixed predictors of different orders, verbatim fallback,
 *     each stereo representation, the 5-bit Rice parameter, short and
 *     odd-sized final blocks)
 *   When Flac() encodes them and the stream is decoded again
 *   Then the decoded PCM equals the input sample-for-sample
 *
 * Scenario: Predictable audio gets smaller
 *   Given a tone, and music-like correlated stereo
 *   When Flac() encodes them
 *   Then the stream is well below the raw size
 *
 * Scenario: Unpredictable audio does not get bigger
 *   Given full-scale white noise
 *   When Flac() encodes it
 *   Then the stream is no larger than the raw PCM plus framing
 */

// stereoRaw builds interleaved s16le PCM from a per-frame generator.
func stereoRaw(nFrames int, gen func(i int) (left, right int16)) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, nFrames*bytesPerFrame))
	for i := range nFrames {
		l, r := gen(i)
		binary.Write(buf, binary.LittleEndian, l)
		binary.Write(buf, binary.LittleEndian, r)
	}
	return buf.Bytes()
}

func sine(i int, freqHz, amplitude float64) float64 {
	return amplitude * math.Sin(2*math.Pi*freqHz*float64(i)/float64(sampleRate))
}

func flacRoundTrip(t *testing.T, in []byte) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := Flac(&encoded, bytes.NewReader(in)); err != nil {
		t.Fatalf("Flac: %v", err)
	}
	out := decodeFlacToRaw(t, encoded.Bytes())
	if len(out) != len(in) {
		t.Fatalf("decoded length %d != input length %d", len(out), len(in))
	}
	if !bytes.Equal(out, in) {
		for i := range in {
			if in[i] != out[i] {
				t.Fatalf("round-trip not lossless: first diff at byte %d (frame %d)", i, i/bytesPerFrame)
			}
		}
	}
	return encoded.Bytes()
}

func TestFlacCompressionRoundTrips(t *testing.T) {
	const block = 4096
	rng := rand.New(rand.NewSource(1))
	noise := func() int16 { return int16(rng.Intn(65536) - 32768) }

	signals := map[string][]byte{
		"digital silence":                 stereoRaw(3*block, func(int) (int16, int16) { return 0, 0 }),
		"different DC offset per channel": stereoRaw(3*block, func(int) (int16, int16) { return 1234, -4321 }),
		"independent tones (left/right)": stereoRaw(3*block, func(i int) (int16, int16) {
			return int16(sine(i, 440, 12000)), int16(sine(i, 1730, 9000))
		}),
		"nearly identical channels (mid/side)": stereoRaw(3*block, func(i int) (int16, int16) {
			v := sine(i, 220, 15000) + sine(i, 3300, 4000)
			return int16(v), int16(v + sine(i, 50, 30))
		}),
		"one channel silent (left/side or side/right)": stereoRaw(3*block, func(i int) (int16, int16) {
			return int16(sine(i, 660, 20000)), 0
		}),
		"ramp (low-order predictor)": stereoRaw(3*block, func(i int) (int16, int16) {
			return int16(i%20000 - 10000), int16(10000 - i%20000)
		}),
		"full-scale white noise (verbatim)": stereoRaw(3*block, func(int) (int16, int16) { return noise(), noise() }),
		"quiet noise (small Rice parameters)": stereoRaw(3*block, func(int) (int16, int16) {
			return int16(rng.Intn(7) - 3), int16(rng.Intn(3) - 1)
		}),
		"alternating extremes (largest residuals)": stereoRaw(3*block, func(i int) (int16, int16) {
			if i%2 == 0 {
				return math.MaxInt16, math.MinInt16
			}
			return math.MinInt16, math.MaxInt16
		}),
		"tone that changes level mid-block (partitioned Rice)": stereoRaw(3*block, func(i int) (int16, int16) {
			amplitude := 200.0
			if i%block > block/2 {
				amplitude = 25000
			}
			v := int16(sine(i, 5000, amplitude) + float64(rng.Intn(21)-10))
			return v, v
		}),
	}
	// Lengths around the block size, down to a single frame: the final block
	// is shorter than the rest and often not divisible into partitions.
	for _, n := range []int{1, 2, 3, 4, 5, 6, 17, 64, 4095, 4096, 4097, 2*block + 1, 2*block + 777} {
		signals["music-like, "+itoa(n)+" frames"] = stereoRaw(n, func(i int) (int16, int16) {
			v := sine(i, 110, 9000) + sine(i, 1870, 3000) + float64(rng.Intn(41)-20)
			return int16(v), int16(v*0.8 + sine(i, 7, 400))
		})
	}

	for name, in := range signals {
		t.Run(name, func(t *testing.T) {
			flacRoundTrip(t, in)
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for ; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}
	return string(digits)
}

func TestFlacCompressesPredictableAudio(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	cases := map[string]struct {
		in       []byte
		maxRatio float64
	}{
		"pure tone": {genSineRaw(1000, 5), 0.35},
		"music-like correlated stereo with a noise floor": {
			stereoRaw(5*sampleRate, func(i int) (int16, int16) {
				v := sine(i, 82, 8000) + sine(i, 660, 5000) + sine(i, 4100, 1500) + float64(rng.Intn(65)-32)
				return int16(v), int16(v*0.9 + sine(i, 3, 300))
			}),
			0.6,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			encoded := flacRoundTrip(t, tc.in)
			ratio := float64(len(encoded)) / float64(len(tc.in))
			if ratio > tc.maxRatio {
				t.Errorf("encoded to %.0f%% of the raw size, want at most %.0f%%", ratio*100, tc.maxRatio*100)
			}
		})
	}
}

func TestFlacDoesNotInflateNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	in := stereoRaw(5*sampleRate, func(int) (int16, int16) {
		return int16(rng.Intn(65536) - 32768), int16(rng.Intn(65536) - 32768)
	})
	encoded := flacRoundTrip(t, in)
	// Verbatim blocks: the raw samples plus a few bytes of framing per block.
	if limit := len(in) + len(in)/200 + 256; len(encoded) > limit {
		t.Errorf("noise encoded to %d bytes, raw is %d (limit %d)", len(encoded), len(in), limit)
	}
}
