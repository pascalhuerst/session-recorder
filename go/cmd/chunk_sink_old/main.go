package main

import (
	"flag"
	"fmt"
	"io"

	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/pascalhuerst/session-recorder/grpc"
	"github.com/pascalhuerst/session-recorder/mdns"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	chunkSinkPort        = 8779
	chunkSinkService     = "_session-recorder-chunksink._tcp"
	sessionSourcePort    = 8780
	sessionSourceService = "_session-recorder-sessionsource._tcp"
)

var (
	version string
)

type indexer struct {
	chunkDir           string
	sessionDir         string
	sessionForRecorder map[string]string

	mdns *mdns.Server
	name string
	uuid string
}

type chunk struct {
	recorderID string
	sessionID  string
	chunkID    string
	timestamp  string
}

func parseFileName(fileName string) (chunk, error) {

	tokens := strings.Split(fileName, "_")
	if len(tokens) != 4 {
		return chunk{}, fmt.Errorf("Cannot parse file name: %s", fileName)
	}

	return chunk{
		recorderID: tokens[0],
		sessionID:  tokens[1],
		chunkID:    tokens[2],
		timestamp:  strings.Replace(tokens[3], ".raw", "", -1),
	}, nil
}

func (i indexer) cleanupChunks() {
	recorders, err := os.ReadDir(i.chunkDir)
	if err != nil {
		fmt.Printf("Cannot read recorders in: %v\n", i.chunkDir)
		return
	}

	for _, recorder := range recorders {
		fmt.Printf("Cleaning up %s\n", recorder.Name())

		sessionsPath := filepath.Join(i.chunkDir, recorder.Name())
		sessions, err := os.ReadDir(sessionsPath)
		if err != nil {
			fmt.Printf("Cannot read sessions in: %v\n", sessionsPath)
			return
		}

		for _, session := range sessions {
			fmt.Printf(" Closing session: %s\n", recorder.Name())
			i.closeSession(recorder.Name(), session.Name())
		}
	}
}

func (i indexer) closeSession(recorderID, sessionID string) {

	targetPath := filepath.Join(i.sessionDir, recorderID, sessionID)
	os.MkdirAll(targetPath, os.ModePerm)

	sourcePath := filepath.Join(i.chunkDir, recorderID, sessionID)
	chunks, err := os.ReadDir(sourcePath)
	if err != nil {
		fmt.Printf("Cannot read chunks in: %v\n", sourcePath)
		return
	}

	targetRawFilePath := filepath.Join(targetPath, "data.raw")
	targetFile, err := os.Create(targetRawFilePath)
	if err != nil {
		fmt.Printf("Cannot create target file: %v\n", targetFile)
		return
	}
	defer targetFile.Close()

	nSamples := 0

	for _, chunk := range chunks {
		chunkFilePath := filepath.Join(sourcePath, chunk.Name())
		d, err := os.ReadFile(chunkFilePath)
		if err != nil {
			fmt.Printf("Cannot read file: %v\n", chunkFilePath)
			return
		}

		n, err := targetFile.Write(d)
		if err != nil {
			fmt.Printf("Cannot write chunk to target file: %v\n", targetFile)
			return
		}
		nSamples += n
	}

	err = os.RemoveAll(sourcePath)
	if err != nil {
		fmt.Printf("Cannot remove source directoy: %v\n", sourcePath)
		return
	}

	createAudioFile := func(fileExtension string) {
		targetAudioFilePath := filepath.Join(targetPath, fmt.Sprintf("data.%s", fileExtension))
		soxCmd := exec.Command("/usr/bin/sox", "-r", "48000", "-b", "16", "-c", "2", "--endian=little", "--encoding=signed-integer", targetRawFilePath, targetAudioFilePath)
		err = soxCmd.Start()
		if err != nil {
			fmt.Printf("Cannot create wav file: %v\n", err)
			return
		}
		err = soxCmd.Wait()
		if err != nil {
			fmt.Printf("Cannot create wav file: %v\n", err)
			return
		}

	}

	createAudioFile("flac")
	createAudioFile("ogg")
	createAudioFile("mp3")

	err = os.Remove(targetRawFilePath)
	if err != nil {
		fmt.Printf("Cannot remove raw audio file: %v\n", err)
		return
	}

	createWaveform := func(inFile, outFile string, zoom, width, height int) error {

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
		err = cmd.Start()
		if err != nil {
			errorBuffer, _ := io.ReadAll(stderr)
			return fmt.Errorf("%s", string(errorBuffer))
		}

		err = cmd.Wait()
		if err != nil {
			return err
		}

		return nil
	}

	waveformSourceFile := filepath.Join(targetPath, "data.flac")
	targetOverviewWaveformImagePath, err := filepath.Abs(filepath.Join(targetPath, "overview.png"))
	if err != nil {
		fmt.Printf("Cannot get absolute path: %v\n", err)
		return
	}
	err = createWaveform(waveformSourceFile, targetOverviewWaveformImagePath, 300, 1000, 200)
	if err != nil {
		fmt.Printf("Cannot create waveform file: %v\n", err)
		return
	}

	targetFullWaveformImagePath, err := filepath.Abs(filepath.Join(targetPath, "full.png"))
	if err != nil {
		fmt.Printf("Cannot get absolute path: %v\n", err)
		return
	}
	err = createWaveform(waveformSourceFile, targetFullWaveformImagePath, 300, 10000, 200)
	if err != nil {
		fmt.Printf("Cannot create waveform file: %v\n", err)
		return
	}

	targetFullWaveformDataPath, err := filepath.Abs(filepath.Join(targetPath, "waveform.dat"))
	if err != nil {
		fmt.Printf("Cannot get absolute path: %v\n", err)
		return
	}
	//TODO:  audiowaveform --input-filename=./data.wav --output-filename=waveform.dat -z 256 -b 8
	err = createWaveform(waveformSourceFile, targetFullWaveformDataPath, 300, 10000, 200)
	if err != nil {
		fmt.Printf("Cannot create waveform file: %v\n", err)
		return
	}

	fmt.Printf("Successfully closed session: %s\n", sessionID)
}

func (i indexer) uploadFile(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("got data...\n")

	r.ParseMultipartForm(1024 * 1024 * 10)
	file, handler, err := r.FormFile("raw_audio")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
	}

	chk, err := parseFileName(handler.Filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	if sessionID, ok := i.sessionForRecorder[chk.recorderID]; !ok {
		i.sessionForRecorder[chk.recorderID] = chk.sessionID
	} else {
		if sessionID != chk.sessionID {
			fmt.Printf("Closing session: %s\n", sessionID)
			go i.closeSession(chk.recorderID, sessionID)
		}
		i.sessionForRecorder[chk.recorderID] = chk.sessionID
	}

	targetPath := filepath.Join(i.chunkDir, chk.recorderID, chk.sessionID)
	os.MkdirAll(targetPath, os.ModePerm)
	targetFilePath := filepath.Join(targetPath, fmt.Sprintf("%s_%s.raw", chk.chunkID, chk.timestamp))

	err = os.WriteFile(targetFilePath, fileBytes, 0664)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	fmt.Printf("[%s] [%v]: session=%s chunk=%s\n", chk.recorderID, time.Now().Format("2006-01-02 15:04:05"), chk.sessionID, chk.chunkID)
	w.WriteHeader(http.StatusOK)
}

func (i indexer) setup() {
	http.HandleFunc("/upload", i.uploadFile)
	http.ListenAndServe(fmt.Sprintf(":%d", chunkSinkPort), nil)
}

func main() {
	chunkDir := flag.String("chunk", "chunks", "Directory to store chunks")
	sessionDir := flag.String("session", "sessions", "Directory to store sessions")
	flag.Parse()

	consoleWriter := zerolog.ConsoleWriter{
		TimeFormat: time.StampMicro,
		Out:        colorable.NewColorableStdout(),
	}
	log.Logger = log.Output(consoleWriter)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown_hostname_" + uuid.NewString()
	}

	i := indexer{
		sessionForRecorder: make(map[string]string),
		name:               "session-recorder-" + hostname,
		uuid:               uuid.NewString(),
	}

	// mdns: announce that we can take chunks
	i.mdns, err = mdns.ServerNew()
	if err != nil {
		log.Err(err).Msg("Cannot create mdns server")

		return
	}

	if err = os.MkdirAll(*chunkDir, os.ModePerm); err != nil {
		fmt.Printf("Cannot create directory: %s\n", *chunkDir)
		return
	}
	i.chunkDir = *chunkDir

	err = os.MkdirAll(*sessionDir, 0773)
	if err != nil {
		fmt.Printf("Cannot create directory: %s\n", *sessionDir)
		return
	}
	i.sessionDir = *sessionDir

	fmt.Printf("Cleaning up old chunks...\n")
	i.cleanupChunks()

	chunkSinkServer := grpc.NewChunkSinkServer(hostname, version)

	port, err := grpc.StartProtocolServer(chunkSinkServer, i.mdns, chunkSinkService, chunkSinkPort)
	if err != nil {
		log.Err(err).Msg("Cannot start chunk sink server")

		return
	}
	log.Info().Msgf("Chunk sink server is now being served on port %d", port)

	sessionSourceServer := grpc.NewSessionSourceServer(hostname, version)
	if err != nil {
		log.Err(err).Msg("Cannot create session source server")

		return
	}

	port, err = grpc.StartProtocolServer(sessionSourceServer, i.mdns, sessionSourceService, sessionSourcePort)
	if err != nil {
		log.Err(err).Msg("Cannot start session source server")

		return
	}

	log.Info().Msgf("Session source server is now being served on port %d", port)

	i.setup()

	select {}
}
