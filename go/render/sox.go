package render

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func CreateAudioFile(inputFilePath, outputFilePath, fileExtension string) error {
	targetAudioFilePath := filepath.Join(outputFilePath, fmt.Sprintf("data.%s", fileExtension))
	soxCmd := exec.Command("/usr/bin/sox", "-r", "48000", "-b", "16", "-c", "2", "--endian=little", "--encoding=signed-integer", inputFilePath, targetAudioFilePath)
	err := soxCmd.Start()
	if err != nil {
		log.Err(err).Str("extension", fileExtension).Msg("Cannot create file")

		return err
	}
	err = soxCmd.Wait()
	if err != nil {
		log.Err(err).Str("extension", fileExtension).Msg("Cannot create file")

		return err
	}

	return nil
}

func CreateWaveform(inFile, outFile string, zoom, width, height int) error {

	const (
		// --background-color
		backgroundColor = "333333"
		// --waveform-color
		waveformColor = "ed730c"
		// --axis-label-color
		fontColor = "0c86ed"
		// --border-color
		borderColor = "0c86ed"
	)

	//strZoom := fmt.Sprintf("%d", zoom)
	strWidth := fmt.Sprintf("%d", width)
	strHeight := fmt.Sprintf("%d", height)
	cmd := exec.Command("audiowaveform",
		"--input-filename", inFile,
		"--output-filename", outFile,
		"--zoom", "auto",
		"--width", strWidth,
		"--height", strHeight,
		"--background-color", backgroundColor,
		"--waveform-color", waveformColor,
		"--axis-label-color", fontColor,
		"--border-color", borderColor)

	// Actual waveform dat files have other params
	if filepath.Ext(outFile) == ".dat" {
		cmd = exec.Command("audiowaveform",
			"--input-filename", inFile,
			"--output-filename", outFile,
			"--zoom", "256",
			"-b", "8")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		errorBuffer, _ := io.ReadAll(stderr)
		errorBuffer2, _ := io.ReadAll(stdout)

		return fmt.Errorf("stderr=%s, stdout=%s", string(errorBuffer), string(errorBuffer2))
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("Canot create waveform %s: %w", outFile, err)
	}

	return nil
}
