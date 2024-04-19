package render

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"os/exec"

	"golang.org/x/sync/errgroup"
)

func CreateWaveform(ctx context.Context, raw io.Reader, zoom, width, height int) (*bytes.Buffer, error) {
	cmd := exec.Command("audiowaveform",
		"--input-filename", "-",
		"--input-format", "raw",
		"--output-filename", "-",
		"--output-format", "dat",
		"--zoom", "256",
		"-b", "8")

	cmd.Stdin = raw

	return run(ctx, cmd)
}

func CreateOverview(ctx context.Context, raw io.Reader, zoom, width, height int) (*bytes.Buffer, error) {
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

	cmd.Stdin = raw

	return run(ctx, cmd)
}

func run(ctx context.Context, cmd *exec.Cmd) (*bytes.Buffer, error) {
	eg, _ := errgroup.WithContext(ctx)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Cannot get stdout pipe: %w", err)
	}

	stdoutBuffer := new(bytes.Buffer)

	// Read from stdout
	eg.Go(func() error {
		_, err := io.Copy(stdoutBuffer, stdout)
		if err != nil {
			return fmt.Errorf("Cannot read from stdout: %w", err)
		}

		return nil
	})

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("Cannot get stderr pipe: %w", err)
	}

	stderrBuffer := new(bytes.Buffer)

	// Read from stderr
	eg.Go(func() error {
		_, err := io.Copy(stderrBuffer, stderr)
		if err != nil {
			return fmt.Errorf("Cannot read from stderr: %w", err)
		}

		return nil
	})

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute: %w", err)
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("Failed to execute: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute: %w", err)
	}

	return stdoutBuffer, nil
}
