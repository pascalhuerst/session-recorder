package render

import (
	"bytes"
	"io"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func CreateAudioFile(raw io.Reader, outFile string) (*bytes.Buffer, error) {
	soxCmd := exec.Command(
		"/usr/bin/sox",
		"-t", "raw",
		"-r", "48000",
		"-b", "16",
		"-c", "2",
		"--endian=little",
		"--encoding=signed-integer",
		"-", //Red from STDIN
		"-t", outFile,
		"-", //Write to STDOUT
	)

	soxStdin, err := soxCmd.StdinPipe()
	if err != nil {
		log.Err(err).Msg("Cannot get stdin pipe")

		return nil, err
	}

	soxStdout, err := soxCmd.StdoutPipe()
	if err != nil {
		log.Err(err).Msg("Cannot get stdout pipe")

		return nil, err
	}

	go func() {
		defer soxStdin.Close()

		_, err := io.Copy(soxStdin, raw)
		if err != nil {
			log.Err(err).Msg("Cannot write to stdin")
		}
	}()

	if err := soxCmd.Start(); err != nil {
		log.Err(err).Msg("Cannot create file")

		return nil, err
	}

	buffer, err := io.ReadAll(soxStdout)
	if err != nil {
		log.Err(err).Msg("Cannot read from stdout")

		return nil, err
	}

	err = soxCmd.Wait()
	if err != nil {
		log.Err(err).Msg("Cannot create file")
	}

	return bytes.NewBuffer(buffer), nil
}
