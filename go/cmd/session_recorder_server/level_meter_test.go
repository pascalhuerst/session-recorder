package main

import (
	"testing"

	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

func TestParseLevelMeter(t *testing.T) {
	cases := map[string]sspb.LevelMeter{
		"both":   sspb.LevelMeter_LEVEL_METER_BOTH,
		"rms":    sspb.LevelMeter_LEVEL_METER_RMS,
		"Peak":   sspb.LevelMeter_LEVEL_METER_PEAK,
		" peak ": sspb.LevelMeter_LEVEL_METER_PEAK,
	}
	for name, want := range cases {
		got, err := parseLevelMeter(name)
		if err != nil || got != want {
			t.Errorf("parseLevelMeter(%q) = %v, %v; want %v", name, got, err, want)
		}
	}
	if _, err := parseLevelMeter("vu"); err == nil {
		t.Error("parseLevelMeter(\"vu\") succeeded, want an error")
	}
}

func TestWithLevelMeterLeavesSharedMessageAlone(t *testing.T) {
	h := &SessionSourceHandler{levelMeter: sspb.LevelMeter_LEVEL_METER_PEAK}
	shared := &sspb.Recorder{
		RecorderID: "rec",
		Info: &sspb.Recorder_Status{
			Status: &cmpb.RecorderStatus{RmsPercent: 20, PeakPercent: 60},
		},
	}

	out := h.withLevelMeter(shared)

	if out.LevelMeter != sspb.LevelMeter_LEVEL_METER_PEAK {
		t.Errorf("LevelMeter = %v, want PEAK", out.LevelMeter)
	}
	if out.GetStatus().GetPeakPercent() != 60 || out.GetStatus().GetRmsPercent() != 20 {
		t.Errorf("status not carried over: %v", out.GetStatus())
	}
	if shared.LevelMeter != sspb.LevelMeter_LEVEL_METER_BOTH {
		t.Errorf("shared message was modified: LevelMeter = %v", shared.LevelMeter)
	}
}
