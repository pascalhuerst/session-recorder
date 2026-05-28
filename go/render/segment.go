package render

import (
	"fmt"
	"io"
	"time"
)

const (
	sampleRate    = 48000
	numChannels   = 2
	bytesPerFrame = numChannels * 2 // s16 => 2 bytes per sample per channel
)

// SamplePositionToDuration converts a sample position to a time.Duration.
// Sample position is the frame index (not byte offset).
func SamplePositionToDuration(samplePosition int64) time.Duration {
	seconds := float64(samplePosition) / float64(sampleRate)
	return time.Duration(seconds * float64(time.Second))
}

// SegmentReader returns a reader yielding only the raw PCM frames in
// [startPos, endPos) of src. Positions are frame indices (s16le, 2ch); the byte
// offset of a frame is pos*bytesPerFrame. The leading frames are skipped by
// reading and discarding them, and the result is length-bounded, so memory use
// is O(1) regardless of segment length — a segment can be longer than RAM.
//
// If the stream ends before endPos, the reader simply yields fewer frames.
// Callers that can address the underlying storage by range (e.g. an S3 range
// request or a file Seek) should do so and feed the windowed reader straight
// into Opus/Flac instead, to avoid transferring the skipped prefix.
func SegmentReader(src io.Reader, startPos, endPos int64) (io.Reader, error) {
	if startPos < 0 || endPos <= startPos {
		return nil, fmt.Errorf("invalid segment range: start=%d end=%d", startPos, endPos)
	}

	skip := startPos * bytesPerFrame
	if _, err := io.CopyN(io.Discard, src, skip); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("segment start %d is beyond end of audio", startPos)
		}
		return nil, err
	}

	want := (endPos - startPos) * bytesPerFrame
	return io.LimitReader(src, want), nil
}
