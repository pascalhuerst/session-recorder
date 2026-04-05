package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	chunkSinkPort = 8779
	sampleRate    = 48000
	channels      = 2
	// Send chunks every 500ms → 48000 * 2 channels * 0.5s = 48000 samples per chunk
	samplesPerChunk = sampleRate * channels / 2
)

func main() {
	host := flag.String("host", "127.0.0.1", "host to connect to")
	device := flag.String("device", "default", "ALSA capture device")
	recorderName := flag.String("name", "Go Test Recorder", "recorder name")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consoleWriter := zerolog.ConsoleWriter{
		TimeFormat: time.StampMicro,
		Out:        colorable.NewColorableStdout(),
	}
	log.Logger = log.Output(consoleWriter)

	recorderID := uuid.New().String()
	log.Info().Str("recorder-id", recorderID).Str("device", *device).Msg("Starting recorder")

	// Start arecord to capture from microphone
	cmd := exec.CommandContext(ctx, "arecord",
		"-D", *device,
		"-f", "S16_LE",
		"-r", fmt.Sprintf("%d", sampleRate),
		"-c", fmt.Sprintf("%d", channels),
		"-t", "raw",
		"--buffer-size", "16384",
	)
	cmd.Stderr = os.Stderr
	audioStream, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create stdout pipe for arecord")
	}
	if err := cmd.Start(); err != nil {
		log.Fatal().Err(err).Msg("Cannot start arecord")
	}
	log.Info().Str("device", *device).Msg("arecord started")

	// Connect to chunk sink
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", *host, chunkSinkPort), opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to chunk sink")
	}
	defer conn.Close()

	client := cspb.NewChunkSinkClient(conn)
	sessionID := uuid.New().String()

	status := cmpb.RecorderStatus{
		RecorderID:   recorderID,
		RecorderName: *recorderName,
		SignalStatus: cmpb.SignalStatus_SIGNAL,
		RmsPercent:   0.5,
		Clipping:     false,
	}

	chunks := cspb.Chunks{
		RecorderID:  recorderID,
		SessionID:   sessionID,
		ChunkCount:  0,
		TimeCreated: timestamppb.Now(),
	}

	// Register command stream so server considers this recorder "connected"
	go func() {
		cmdStream, err := client.GetCommands(ctx, &cspb.GetCommandRequest{
			RecorderID: recorderID,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot open command stream")
		}
		for {
			cmd, err := cmdStream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Err(err).Msg("Command stream error")
				return
			}
			log.Info().Str("command", cmd.String()).Msg("Received command")
			sessionID = uuid.New().String()
			chunks.SessionID = sessionID
			chunks.ChunkCount = 0
			log.Info().Str("new-session-id", sessionID).Msg("Session cut, starting new session")
		}
	}()

	// Status sender
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s := proto.Clone(&status).(*cmpb.RecorderStatus)
				if _, err := client.SetRecorderStatus(ctx, s, grpc.EmptyCallOption{}); err != nil {
					log.Fatal().Err(err).Msg("Cannot send status")
				}
			}
		}
	}()

	// Read audio from arecord and send chunks
	buf := make([]int16, samplesPerChunk)
	for {
		if err := binary.Read(audioStream, binary.LittleEndian, buf); err != nil {
			if err == io.EOF {
				log.Info().Msg("Audio stream ended")
				break
			}
			log.Fatal().Err(err).Msg("Cannot read audio")
		}

		c := proto.Clone(&chunks).(*cspb.Chunks)
		c.TimeCreated = timestamppb.Now()

		// Convert int16 → uint32 for proto (server does int16(x) to recover)
		c.Data = make([]uint32, len(buf))
		for i, s := range buf {
			c.Data[i] = uint32(s)
		}

		if _, err := client.SetChunks(ctx, c, grpc.EmptyCallOption{}); err != nil {
			log.Fatal().Err(err).Msg("Cannot send chunks")
		}

		chunks.ChunkCount++
		log.Info().Str("session-id", sessionID).Msgf("Sent chunk %d", chunks.ChunkCount)
	}

	_ = cmd.Wait()
}
