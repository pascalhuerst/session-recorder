package broadcast

import (
	"math"
	"testing"
)

func TestScaleInt16ToInt8(t *testing.T) {
	tests := []struct {
		in   int16
		want int8
	}{
		{0, 0},
		{32767, 127},
		{-32767, -127},
		{-32768, -127}, // -32768 * 127 / 32767 ≈ -127
		{16384, 63},    // ~half scale
		{-16384, -63},
	}
	for _, tt := range tests {
		got := scaleInt16ToInt8(tt.in)
		if got != tt.want {
			t.Errorf("scaleInt16ToInt8(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestAddSamples_ExactWindow(t *testing.T) {
	pa := NewPeakAccumulator(4) // 4 samples per peak for easy testing

	samples := []int16{-100, 50, -200, 300}
	result := pa.AddSamples("s1", samples)

	if len(result.Peaks) != 1 {
		t.Fatalf("expected 1 peak, got %d", len(result.Peaks))
	}

	// Min should be -200, Max should be 300
	wantMin := scaleInt16ToInt8(-200)
	wantMax := scaleInt16ToInt8(300)
	if result.Peaks[0].Min != wantMin {
		t.Errorf("min = %d, want %d", result.Peaks[0].Min, wantMin)
	}
	if result.Peaks[0].Max != wantMax {
		t.Errorf("max = %d, want %d", result.Peaks[0].Max, wantMax)
	}
}

func TestAddSamples_MultipleWindows(t *testing.T) {
	pa := NewPeakAccumulator(2)

	samples := []int16{-100, 200, -300, 400}
	result := pa.AddSamples("s1", samples)

	if len(result.Peaks) != 2 {
		t.Fatalf("expected 2 peaks, got %d", len(result.Peaks))
	}

	// Window 1: [-100, 200] → min=-100, max=200
	if result.Peaks[0].Min != scaleInt16ToInt8(-100) {
		t.Errorf("peak[0].min = %d, want %d", result.Peaks[0].Min, scaleInt16ToInt8(-100))
	}
	if result.Peaks[0].Max != scaleInt16ToInt8(200) {
		t.Errorf("peak[0].max = %d, want %d", result.Peaks[0].Max, scaleInt16ToInt8(200))
	}

	// Window 2: [-300, 400] → min=-300, max=400
	if result.Peaks[1].Min != scaleInt16ToInt8(-300) {
		t.Errorf("peak[1].min = %d, want %d", result.Peaks[1].Min, scaleInt16ToInt8(-300))
	}
	if result.Peaks[1].Max != scaleInt16ToInt8(400) {
		t.Errorf("peak[1].max = %d, want %d", result.Peaks[1].Max, scaleInt16ToInt8(400))
	}
}

func TestAddSamples_PartialWindow(t *testing.T) {
	pa := NewPeakAccumulator(4)

	// First call: 3 samples, not enough for a window
	result := pa.AddSamples("s1", []int16{10, 20, 30})
	if len(result.Peaks) != 0 {
		t.Fatalf("expected 0 peaks, got %d", len(result.Peaks))
	}

	// Second call: 1 more sample completes the window
	result = pa.AddSamples("s1", []int16{-50})
	if len(result.Peaks) != 1 {
		t.Fatalf("expected 1 peak, got %d", len(result.Peaks))
	}

	// Min should be -50, Max should be 30
	if result.Peaks[0].Min != scaleInt16ToInt8(-50) {
		t.Errorf("min = %d, want %d", result.Peaks[0].Min, scaleInt16ToInt8(-50))
	}
	if result.Peaks[0].Max != scaleInt16ToInt8(30) {
		t.Errorf("max = %d, want %d", result.Peaks[0].Max, scaleInt16ToInt8(30))
	}
}

func TestAddSamples_PeakLevel(t *testing.T) {
	pa := NewPeakAccumulator(4)

	samples := []int16{0, 0, 16384, 0} // peak at ~0.5
	result := pa.AddSamples("s1", samples)

	expected := float32(16384) / 32767.0
	if result.PeakLevel < expected-0.001 || result.PeakLevel > expected+0.001 {
		t.Errorf("peakLevel = %f, want ~%f", result.PeakLevel, expected)
	}
}

func TestAddSamples_ClippingPositive(t *testing.T) {
	pa := NewPeakAccumulator(4)

	samples := []int16{0, 0, math.MaxInt16, 0}
	result := pa.AddSamples("s1", samples)

	if !result.Clipping {
		t.Error("expected clipping=true for MaxInt16")
	}
}

func TestAddSamples_ClippingNegative(t *testing.T) {
	pa := NewPeakAccumulator(4)

	samples := []int16{0, 0, math.MinInt16, 0}
	result := pa.AddSamples("s1", samples)

	if !result.Clipping {
		t.Error("expected clipping=true for MinInt16")
	}
}

func TestAddSamples_NoClipping(t *testing.T) {
	pa := NewPeakAccumulator(4)

	samples := []int16{100, -100, 500, -500}
	result := pa.AddSamples("s1", samples)

	if result.Clipping {
		t.Error("expected clipping=false")
	}
}

func TestGetAccumulated(t *testing.T) {
	pa := NewPeakAccumulator(2)

	pa.AddSamples("s1", []int16{-10000, 20000})
	pa.AddSamples("s1", []int16{-30000, 10000})

	accumulated := pa.GetAccumulated("s1")
	if len(accumulated) != 2 {
		t.Fatalf("expected 2 accumulated peaks, got %d", len(accumulated))
	}

	// Verify it's a copy (modifying doesn't affect internal state)
	accumulated[0].Min = 0
	original := pa.GetAccumulated("s1")
	if original[0].Min == 0 {
		t.Error("GetAccumulated should return a copy")
	}
}

func TestGetAccumulated_UnknownSession(t *testing.T) {
	pa := NewPeakAccumulator(2)

	accumulated := pa.GetAccumulated("nonexistent")
	if accumulated != nil {
		t.Errorf("expected nil for unknown session, got %v", accumulated)
	}
}

func TestGetAccumulatedCount(t *testing.T) {
	pa := NewPeakAccumulator(2)

	pa.AddSamples("s1", []int16{1, 2, 3, 4, 5, 6})

	count := pa.GetAccumulatedCount("s1")
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
}

func TestRemoveSession(t *testing.T) {
	pa := NewPeakAccumulator(2)

	pa.AddSamples("s1", []int16{1, 2})
	pa.RemoveSession("s1")

	if pa.GetAccumulatedCount("s1") != 0 {
		t.Error("expected 0 after RemoveSession")
	}
	if pa.GetAccumulated("s1") != nil {
		t.Error("expected nil after RemoveSession")
	}
}

func TestAddSamples_EmptyInput(t *testing.T) {
	pa := NewPeakAccumulator(4)

	result := pa.AddSamples("s1", []int16{})
	if len(result.Peaks) != 0 {
		t.Fatalf("expected 0 peaks for empty input, got %d", len(result.Peaks))
	}
	if result.PeakLevel != 0 {
		t.Errorf("expected peakLevel=0 for empty input, got %f", result.PeakLevel)
	}
	if result.Clipping {
		t.Error("expected clipping=false for empty input")
	}
	if pa.GetAccumulatedCount("s1") != 0 {
		t.Errorf("expected 0 accumulated peaks, got %d", pa.GetAccumulatedCount("s1"))
	}
}

func TestAddSamples_SingleSample(t *testing.T) {
	pa := NewPeakAccumulator(4)

	result := pa.AddSamples("s1", []int16{500})
	if len(result.Peaks) != 0 {
		t.Fatalf("expected 0 peaks for single sample (window=4), got %d", len(result.Peaks))
	}

	expected := float32(500) / 32767.0
	if result.PeakLevel < expected-0.001 || result.PeakLevel > expected+0.001 {
		t.Errorf("peakLevel = %f, want ~%f", result.PeakLevel, expected)
	}

	// Pending should hold the sample
	result = pa.AddSamples("s1", []int16{100, 200, -600})
	if len(result.Peaks) != 1 {
		t.Fatalf("expected 1 peak after completing window, got %d", len(result.Peaks))
	}
	if result.Peaks[0].Min != scaleInt16ToInt8(-600) {
		t.Errorf("min = %d, want %d", result.Peaks[0].Min, scaleInt16ToInt8(-600))
	}
	if result.Peaks[0].Max != scaleInt16ToInt8(500) {
		t.Errorf("max = %d, want %d", result.Peaks[0].Max, scaleInt16ToInt8(500))
	}
}

func TestAddSamples_NegativePeakLevel(t *testing.T) {
	pa := NewPeakAccumulator(4)

	// Negative samples should still produce correct peak level
	samples := []int16{0, 0, -20000, 0}
	result := pa.AddSamples("s1", samples)

	expected := float32(20000) / 32767.0
	if result.PeakLevel < expected-0.001 || result.PeakLevel > expected+0.001 {
		t.Errorf("peakLevel = %f, want ~%f", result.PeakLevel, expected)
	}
}

func TestMultipleSessions(t *testing.T) {
	pa := NewPeakAccumulator(2)

	pa.AddSamples("s1", []int16{100, 200})
	pa.AddSamples("s2", []int16{-300, -400})

	s1 := pa.GetAccumulated("s1")
	s2 := pa.GetAccumulated("s2")

	if len(s1) != 1 || len(s2) != 1 {
		t.Fatalf("expected 1 peak each, got s1=%d s2=%d", len(s1), len(s2))
	}

	if s1[0].Max != scaleInt16ToInt8(200) {
		t.Errorf("s1 max = %d, want %d", s1[0].Max, scaleInt16ToInt8(200))
	}
	if s2[0].Min != scaleInt16ToInt8(-400) {
		t.Errorf("s2 min = %d, want %d", s2[0].Min, scaleInt16ToInt8(-400))
	}
}
