package render

import (
	"bytes"
	"fmt"
	"io"

	"os/exec"

	"errors"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func CreateWaveform(raw io.Reader, zoom, width, height int) (*bytes.Buffer, error) {
	cmd := exec.Command("audiowaveform",
		"--input-format", "raw",
		"--output-format", "dat",
		"--zoom", "256",
		"-b", "8")

	return run(cmd, raw)
}

func CreateOverview(rawAudio io.Reader, zoom, width, height int) (*bytes.Buffer, error) {
	const (
		backgroundColor = "333333fe"
		waveformColor   = "ed730cfe"
		fontColor       = "0c86edfe"
		borderColor     = "0c86edfe"
	)

	strWidth := fmt.Sprintf("%d", width)
	strHeight := fmt.Sprintf("%d", height)

	cmd := exec.Command("audiowaveform",
		"--input-format", "raw",
		"--output-format", "png",
		"--zoom", "auto",
		"--width", strWidth,
		"--height", strHeight,
		"--background-color", backgroundColor,
		"--waveform-color", waveformColor,
		"--axis-label-color", fontColor,
		"--border-color", borderColor)

	return run(cmd, rawAudio)
}

func run(cmd *exec.Cmd, rawAudio io.Reader) (*bytes.Buffer, error) {
	eg := errgroup.Group{}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("Cannot read from stdin: %w", err)
	}

	eg.Go(func() error {
		defer stdin.Close()

		_, err := io.Copy(stdin, rawAudio)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Cannot write to stdin: %w", err)
		}

		return nil
	})

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Cannot read from stdout: %w", err)
	}

	stdoutBuffer := new(bytes.Buffer)

	eg.Go(func() error {
		defer stdout.Close()

		_, err := io.Copy(stdoutBuffer, stdout)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Cannot read from stdout: %w", err)
		}

		return nil
	})

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stderrBuffer := new(bytes.Buffer)

	eg.Go(func() error {
		_, err := io.Copy(stderrBuffer, stderr)

		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Cannot read from stderr: %w", err)
		}

		return nil
	})

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("Cannot create waveform: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Error().Str("stderr", stderrBuffer.String()).Msg("Cannot create waveform overview")

		return nil, fmt.Errorf("Cannot create waveform: %w", err)
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("Cannot create waveform: %w", err)
	}

	return stdoutBuffer, nil
}
