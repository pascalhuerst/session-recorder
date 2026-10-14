package render

import (
	"bytes"
	"crypto/md5"
	"io"
	"strings"
	"testing"

	"github.com/mewkiz/flac"
)

/**
 * Test Plan: FLAC header carries the stream's length and MD5
 *
 * Scenario: A summarized stream states its length and checksum
 *   Given raw PCM and its PCMSummary
 *   When PCMSummary.Flac() encodes it into a writer that cannot seek
 *   Then STREAMINFO holds the frame count and the MD5 of the raw bytes
 *   And the stream still round-trips losslessly
 *
 * Scenario: A summary ignores a trailing partial frame
 *   Given raw PCM with a few stray bytes after the last whole frame
 *   When SummarizePCM() reads it
 *   Then the count and MD5 cover the whole frames only, as the encoder does
 *
 * Scenario: A summary of different audio is refused
 *   Given a PCMSummary taken from other audio than what is being encoded
 *   When PCMSummary.Flac() encodes
 *   Then it fails instead of writing a header that contradicts the stream
 *
 * Scenario: Encoding without a summary still works
 *   Given raw PCM and no summary
 *   When Flac() encodes it
 *   Then STREAMINFO states an unknown length and no MD5
 */

// onlyWriter hides any Seek method of the wrapped writer, like the pipes and
// buffered writers the storage backends encode into.
type onlyWriter struct{ w io.Writer }

func (o onlyWriter) Write(p []byte) (int, error) { return o.w.Write(p) }

func TestFlacSummaryIsWrittenToHeader(t *testing.T) {
	in := genSineRaw(440, 1.37) // not a multiple of the block size

	summary, err := SummarizePCM(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("SummarizePCM: %v", err)
	}
	if want := uint64(len(in) / bytesPerFrame); summary.Frames != want {
		t.Fatalf("summary.Frames = %d, want %d", summary.Frames, want)
	}
	if want := md5.Sum(in); summary.MD5 != want {
		t.Fatalf("summary.MD5 = %x, want %x", summary.MD5, want)
	}

	var encoded bytes.Buffer
	if err := summary.Flac(onlyWriter{&encoded}, bytes.NewReader(in)); err != nil {
		t.Fatalf("PCMSummary.Flac: %v", err)
	}

	stream, err := flac.New(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("flac.New: %v", err)
	}
	if stream.Info.NSamples != summary.Frames {
		t.Errorf("STREAMINFO NSamples = %d, want %d", stream.Info.NSamples, summary.Frames)
	}
	if stream.Info.MD5sum != summary.MD5 {
		t.Errorf("STREAMINFO MD5 = %x, want %x", stream.Info.MD5sum, summary.MD5)
	}
	if out := decodeFlacToRaw(t, encoded.Bytes()); !bytes.Equal(out, in) {
		t.Error("summarized stream does not round-trip losslessly")
	}
}

func TestSummarizePCMIgnoresTrailingPartialFrame(t *testing.T) {
	whole := genSineRaw(440, 0.5)
	in := append(append([]byte{}, whole...), 1, 2, 3)

	summary, err := SummarizePCM(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("SummarizePCM: %v", err)
	}
	if want := uint64(len(whole) / bytesPerFrame); summary.Frames != want {
		t.Errorf("summary.Frames = %d, want %d", summary.Frames, want)
	}
	if want := md5.Sum(whole); summary.MD5 != want {
		t.Errorf("summary.MD5 = %x, want %x (MD5 of the whole frames)", summary.MD5, want)
	}

	// And the encoder agrees with that summary for the very same input.
	var encoded bytes.Buffer
	if err := summary.Flac(&encoded, bytes.NewReader(in)); err != nil {
		t.Errorf("PCMSummary.Flac on input with a trailing partial frame: %v", err)
	}
}

func TestFlacRefusesSummaryOfOtherAudio(t *testing.T) {
	cases := map[string][]byte{
		"same length, different content": genSineRaw(880, 1),
		"different length":               genSineRaw(440, 0.9),
	}
	summary, err := SummarizePCM(bytes.NewReader(genSineRaw(440, 1)))
	if err != nil {
		t.Fatalf("SummarizePCM: %v", err)
	}
	for name, other := range cases {
		t.Run(name, func(t *testing.T) {
			var encoded bytes.Buffer
			err := summary.Flac(&encoded, bytes.NewReader(other))
			if err == nil {
				t.Fatal("PCMSummary.Flac accepted audio the summary was not taken from")
			}
			if !strings.Contains(err.Error(), "summary") {
				t.Errorf("error does not name the summary mismatch: %v", err)
			}
		})
	}
}

func TestFlacWithoutSummaryLeavesHeaderUnknown(t *testing.T) {
	var encoded bytes.Buffer
	if err := Flac(onlyWriter{&encoded}, bytes.NewReader(genSineRaw(440, 1))); err != nil {
		t.Fatalf("Flac: %v", err)
	}
	stream, err := flac.New(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("flac.New: %v", err)
	}
	if stream.Info.NSamples != 0 {
		t.Errorf("STREAMINFO NSamples = %d, want 0 (unknown)", stream.Info.NSamples)
	}
	if stream.Info.MD5sum != [md5.Size]byte{} {
		t.Errorf("STREAMINFO MD5 = %x, want all zero (none)", stream.Info.MD5sum)
	}
}
