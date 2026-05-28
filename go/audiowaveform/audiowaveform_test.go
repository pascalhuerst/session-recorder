package audiowaveform

import (
	"bytes"
	"os"
	"testing"
)

// TestGenerateDatMatchesGolden compares our Go output against golden data
// produced by the real C++ `audiowaveform` binary (v1.10.2) for the exact
// arguments render.CreateWaveform uses. Regenerate the golden with:
//
//	audiowaveform --input-filename testdata/input.raw --input-format raw \
//	  --raw-samplerate 48000 --raw-channels 2 --raw-format s16le \
//	  --output-filename testdata/golden.dat --output-format dat --zoom 256 -b 8
func TestGenerateDatMatchesGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/input.raw")
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	golden, err := os.ReadFile("testdata/golden.dat")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	got, err := GenerateDat(bytes.NewReader(raw), DefaultParams())
	if err != nil {
		t.Fatalf("GenerateDat: %v", err)
	}

	if !bytes.Equal(got.Bytes(), golden) {
		t.Fatalf("output mismatch with audiowaveform golden:\n got    (%d bytes): %x\n golden (%d bytes): %x",
			got.Len(), got.Bytes(), len(golden), golden)
	}
}
