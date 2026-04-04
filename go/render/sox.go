package render

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// CreateAudioFileStream converts raw PCM audio to the specified format,
// writing the output directly to w instead of buffering in memory.
func CreateAudioFileStream(raw io.Reader, outFormat string, w io.Writer) error {
	soxCmd := exec.Command(
		"/usr/bin/sox",
		"-t", "raw",
		"-r", "48000",
		"-b", "16",
		"-c", "2",
		"--endian=little",
		"--encoding=signed-integer",
		"-",
		"-t", outFormat,
		"-",
	)

	soxStdin, err := soxCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("cannot get stdin pipe: %w", err)
	}

	soxStdout, err := soxCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot get stdout pipe: %w", err)
	}

	go func() {
		defer soxStdin.Close()
		if _, err := io.Copy(soxStdin, raw); err != nil {
			log.Err(err).Msg("Cannot write to stdin")
		}
	}()

	if err := soxCmd.Start(); err != nil {
		return fmt.Errorf("cannot start sox: %w", err)
	}

	if _, err := io.Copy(w, soxStdout); err != nil {
		return fmt.Errorf("cannot read from sox stdout: %w", err)
	}

	if err := soxCmd.Wait(); err != nil {
		return fmt.Errorf("sox failed: %w", err)
	}

	return nil
}

// CreateAudioFile converts raw PCM audio to the specified format, returning the result as a buffer.
func CreateAudioFile(raw io.Reader, outFile string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := CreateAudioFileStream(raw, outFile, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
