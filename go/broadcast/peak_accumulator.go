package broadcast

import (
	"math"
	"sync"
)

// PeakPair represents a min/max waveform data point.
type PeakPair struct {
	Min int8
	Max int8
}

// PeakResult holds the output of AddSamples: new peaks plus metering info.
type PeakResult struct {
	Peaks     []PeakPair
	PeakLevel float32 // max |sample| / 32767.0 across the batch
	Clipping  bool    // true if any sample hit int16 bounds
}

type sessionPeaks struct {
	mu      sync.RWMutex
	pairs   []PeakPair
	pending []int16 // partial window buffer
}

// PeakAccumulator computes waveform min/max peaks from int16 PCM samples.
// It accumulates peaks per session and supports backfill on reconnect.
type PeakAccumulator struct {
	mu              sync.RWMutex
	sessions        map[string]*sessionPeaks
	samplesPerPixel int // number of interleaved samples per peak (e.g. 512 for 256 stereo frames)
}

// NewPeakAccumulator creates a PeakAccumulator.
// samplesPerPixel is the number of interleaved int16 samples per peak pair
// (e.g. 512 for 256 stereo frames at 48kHz).
func NewPeakAccumulator(samplesPerPixel int) *PeakAccumulator {
	return &PeakAccumulator{
		sessions:        make(map[string]*sessionPeaks),
		samplesPerPixel: samplesPerPixel,
	}
}

func (pa *PeakAccumulator) getOrCreate(sessionID string) *sessionPeaks {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	sp, ok := pa.sessions[sessionID]
	if !ok {
		sp = &sessionPeaks{}
		pa.sessions[sessionID] = sp
	}
	return sp
}

// AddSamples feeds interleaved int16 PCM samples and returns newly produced peaks,
// the peak level (0.0-1.0), and whether clipping was detected.
func (pa *PeakAccumulator) AddSamples(sessionID string, samples []int16) PeakResult {
	sp := pa.getOrCreate(sessionID)

	sp.mu.Lock()
	defer sp.mu.Unlock()

	var result PeakResult
	var maxAbs int16

	// Check all incoming samples for peak level and clipping
	for _, s := range samples {
		abs := s
		if abs < 0 {
			// Handle int16 min edge case: -(-32768) overflows, treat as 32767
			if abs == math.MinInt16 {
				abs = math.MaxInt16
				result.Clipping = true
			} else {
				abs = -abs
			}
		} else if abs == math.MaxInt16 {
			result.Clipping = true
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	result.PeakLevel = float32(maxAbs) / 32767.0

	// Append to pending buffer
	sp.pending = append(sp.pending, samples...)

	// Process complete windows
	for len(sp.pending) >= pa.samplesPerPixel {
		window := sp.pending[:pa.samplesPerPixel]
		sp.pending = sp.pending[pa.samplesPerPixel:]

		var wMin, wMax int16
		wMin = window[0]
		wMax = window[0]
		for _, s := range window[1:] {
			if s < wMin {
				wMin = s
			}
			if s > wMax {
				wMax = s
			}
		}

		pair := PeakPair{
			Min: scaleInt16ToInt8(wMin),
			Max: scaleInt16ToInt8(wMax),
		}
		sp.pairs = append(sp.pairs, pair)
		result.Peaks = append(result.Peaks, pair)
	}

	return result
}

// GetAccumulated returns a snapshot of all accumulated peaks for a session.
func (pa *PeakAccumulator) GetAccumulated(sessionID string) []PeakPair {
	pa.mu.RLock()
	sp, ok := pa.sessions[sessionID]
	pa.mu.RUnlock()

	if !ok {
		return nil
	}

	sp.mu.RLock()
	defer sp.mu.RUnlock()

	out := make([]PeakPair, len(sp.pairs))
	copy(out, sp.pairs)
	return out
}

// GetAccumulatedCount returns the number of accumulated peak pairs for a session.
func (pa *PeakAccumulator) GetAccumulatedCount(sessionID string) int {
	pa.mu.RLock()
	sp, ok := pa.sessions[sessionID]
	pa.mu.RUnlock()

	if !ok {
		return 0
	}

	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return len(sp.pairs)
}

// RemoveSession cleans up accumulated data for a session.
func (pa *PeakAccumulator) RemoveSession(sessionID string) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	delete(pa.sessions, sessionID)
}

// scaleInt16ToInt8 maps an int16 value to int8 range.
func scaleInt16ToInt8(v int16) int8 {
	return int8(int32(v) * 127 / 32767)
}
