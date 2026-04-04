package render

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"os/exec"

	"golang.org/x/sync/errgroup"
)

// CreateWaveformStream writes waveform data directly to w.
func CreateWaveformStream(ctx context.Context, raw io.Reader, zoom, width, height int, w io.Writer) error {
	cmd := exec.Command("audiowaveform",
		"--input-filename", "-",
		"--input-format", "raw",
		"--raw-samplerate", "48000",
		"--raw-channels", "2",
		"--raw-format", "s16le",
		"--output-filename", "-",
		"--output-format", "dat",
		"--zoom", "256",
		"-b", "8")

	cmd.Stdin = raw

	return runStream(ctx, cmd, w)
}

func CreateWaveform(ctx context.Context, raw io.Reader, zoom, width, height int) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := CreateWaveformStream(ctx, raw, zoom, width, height, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// CreateOverviewStream writes overview PNG directly to w.
func CreateOverviewStream(ctx context.Context, raw io.Reader, zoom, width, height int, w io.Writer) error {
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
		"--raw-samplerate", "48000",
		"--raw-channels", "2",
		"--raw-format", "s16le",
		"--output-format", "png",
		"--zoom", "auto",
		"--width", strWidth,
		"--height", strHeight,
		"--background-color", backgroundColor,
		"--waveform-color", waveformColor,
		"--axis-label-color", fontColor,
		"--border-color", borderColor)

	cmd.Stdin = raw

	return runStream(ctx, cmd, w)
}

func CreateOverview(ctx context.Context, raw io.Reader, zoom, width, height int) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := CreateOverviewStream(ctx, raw, zoom, width, height, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func runStream(ctx context.Context, cmd *exec.Cmd, w io.Writer) error {
	eg, _ := errgroup.WithContext(ctx)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot get stdout pipe: %w", err)
	}

	eg.Go(func() error {
		if _, err := io.Copy(w, stdout); err != nil {
			return fmt.Errorf("cannot read from stdout: %w", err)
		}
		return nil
	})

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cannot get stderr pipe: %w", err)
	}

	stderrBuffer := new(bytes.Buffer)

	eg.Go(func() error {
		if _, err := io.Copy(stderrBuffer, stderr); err != nil {
			return fmt.Errorf("cannot read from stderr: %w", err)
		}
		return nil
	})

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}

	return nil
}
