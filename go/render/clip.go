package render

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	sampleRate = 48000
	channels   = 2
)

// SamplePositionToDuration converts a sample position to a time.Duration.
// Sample position is the frame index (not byte offset).
func SamplePositionToDuration(samplePosition int64) time.Duration {
	seconds := float64(samplePosition) / float64(sampleRate)
	return time.Duration(seconds * float64(time.Second))
}

// ClipAndEncodeOgg clips raw audio data and encodes to OGG format.
// startPos and endPos are sample positions (frame indices).
func ClipAndEncodeOgg(raw io.Reader, startPos, endPos int64) (*bytes.Buffer, error) {
	return clipAndEncode(raw, startPos, endPos, "ogg")
}

// ClipAndEncodeFlac clips raw audio data and encodes to FLAC format.
// startPos and endPos are sample positions (frame indices).
func ClipAndEncodeFlac(raw io.Reader, startPos, endPos int64) (*bytes.Buffer, error) {
	return clipAndEncode(raw, startPos, endPos, "flac")
}

// clipAndEncode uses sox to clip audio to the specified sample range and encode to the given format.
func clipAndEncode(raw io.Reader, startPos, endPos int64, outputFormat string) (*bytes.Buffer, error) {
	startTime := SamplePositionToDuration(startPos)
	endTime := SamplePositionToDuration(endPos)
	duration := endTime - startTime

	if duration <= 0 {
		return nil, fmt.Errorf("invalid segment range: start=%d end=%d", startPos, endPos)
	}

	// Format times as seconds with decimal precision
	startSec := fmt.Sprintf("%.6f", startTime.Seconds())
	durationSec := fmt.Sprintf("%.6f", duration.Seconds())

	log.Debug().
		Int64("startPos", startPos).
		Int64("endPos", endPos).
		Str("startTime", startSec).
		Str("duration", durationSec).
		Str("format", outputFormat).
		Msg("Clipping audio segment")

	soxCmd := exec.Command(
		"/usr/bin/sox",
		"-t", "raw",
		"-r", "48000",
		"-b", "16",
		"-c", "2",
		"--endian=little",
		"--encoding=signed-integer",
		"-",
		"-t", outputFormat,
		"-",
		"trim", startSec, durationSec,
	)

	soxStdin, err := soxCmd.StdinPipe()
	if err != nil {
		log.Err(err).Msg("Cannot get stdin pipe for sox clip")
		return nil, err
	}

	soxStdout, err := soxCmd.StdoutPipe()
	if err != nil {
		log.Err(err).Msg("Cannot get stdout pipe for sox clip")
		return nil, err
	}

	soxStderr, err := soxCmd.StderrPipe()
	if err != nil {
		log.Err(err).Msg("Cannot get stderr pipe for sox clip")
		return nil, err
	}

	go func() {
		defer soxStdin.Close()
		_, err := io.Copy(soxStdin, raw)
		// Ignore broken pipe errors - sox closes stdin after reading enough data for trim
		if err != nil && !strings.Contains(err.Error(), "broken pipe") && !strings.Contains(err.Error(), "file already closed") {
			log.Err(err).Msg("Cannot write to sox stdin")
		}
	}()

	if err := soxCmd.Start(); err != nil {
		log.Err(err).Msg("Cannot start sox clip command")
		return nil, err
	}

	buffer, err := io.ReadAll(soxStdout)
	if err != nil {
		log.Err(err).Msg("Cannot read from sox stdout")
		return nil, err
	}

	stderrOutput, _ := io.ReadAll(soxStderr)

	if err := soxCmd.Wait(); err != nil {
		log.Err(err).Str("stderr", string(stderrOutput)).Msg("Sox clip command failed")
		return nil, fmt.Errorf("sox clip failed: %w, stderr: %s", err, string(stderrOutput))
	}

	log.Debug().
		Int("outputBytes", len(buffer)).
		Str("format", outputFormat).
		Msg("Segment clipping complete")

	return bytes.NewBuffer(buffer), nil
}
