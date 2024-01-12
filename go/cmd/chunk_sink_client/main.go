package main

// Implement a chunk sink client for testing purposes

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"time"

	_ "embed"

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
	recorderID    = "2bb00a6a-c468-41b0-b8b8-40e3cd22450e"
	chunkSize     = 480000 // 10 seconds at 48000kHz
)

//go:embed test_data/sine_1k_48k_s16_c2.raw
var byteSamples []byte

func main() {
	host := flag.String("host", "127.0.0.1", "host to connect to")
	flag.Parse()

	ctx := context.Background()

	consoleWriter := zerolog.ConsoleWriter{
		TimeFormat: time.StampMicro,
		Out:        colorable.NewColorableStdout(),
	}
	log.Logger = log.Output(consoleWriter)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", *host, chunkSinkPort), opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to session source. Giving up")
	}
	defer conn.Close()

	client := cspb.NewChunkSinkClient(conn)

	status := cmpb.RecorderStatus{
		RecorderID:   recorderID,
		RecorderName: "Test Recorder 1",
		SignalStatus: cmpb.SignalStatus_SIGNAL,
		RmsPercent:   0.5,
		Clipping:     false,
	}

	chunks := cspb.Chunks{
		RecorderID:  recorderID,
		SessionID:   uuid.New().String(),
		ChunkCount:  0,
		TimeCreated: timestamppb.Now(),
	}

	chunkData := make([]uint32, chunkSize)
	if err := binary.Read(bytes.NewReader(byteSamples), binary.LittleEndian, chunkData); err != nil {
		log.Fatal().Err(err).Msg("Cannot read samples")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.Tick(time.Millisecond * 500):
				c := proto.Clone(&chunks).(*cspb.Chunks)

				c.Data = chunkData

				_, err := client.SetChunks(ctx, c, grpc.EmptyCallOption{})
				if err != nil {
					log.Fatal().Err(err).Msg("Cannot send chunks. Giving up")
				}

				chunks.ChunkCount++

				log.Info().Str("session-id", c.SessionID).Msgf("Sent chunk %d", chunks.ChunkCount)
			}

		}
	}()

	go func() {
		i := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.Tick(time.Millisecond * 500):
				s := proto.Clone(&status).(*cmpb.RecorderStatus)

				_, err = client.SetRecorderStatus(ctx, s, grpc.EmptyCallOption{})
				if err != nil {
					log.Fatal().Err(err).Msg("Cannot send status. Giving up")
				}

				log.Info().Msgf("Sent status %d", i)

				i++
			}

		}
	}()

	<-ctx.Done()

}
