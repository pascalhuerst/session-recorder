package render

import (
	"fmt"
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
