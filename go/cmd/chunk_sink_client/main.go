package main

// Implement a chunk sink client for testing purposes

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	chunkSinkPort = 8779
	recorderID    = "8fcb40a9-c184-4bd4-a733-8a7e808acb49"
	chunkSize     = 131072
)

func main() {
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

	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", chunkSinkPort), opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to session source. Giving up")
	}
	defer conn.Close()

	client := cspb.NewChunkSinkClient(conn)

	/*
		status := cmpb.RecorderStatus{
			RecorderID:   recorderID,
			RecorderName: "Test Recorder 1",
			SignalStatus: cmpb.SignalStatus_SIGNAL,
			RmsPercent:   0.5,
			Clipping:     false,
		}
	*/

	chunks := cspb.Chunks{
		RecorderID:  recorderID,
		SessionID:   uuid.New().String(),
		ChunkCount:  0,
		TimeCreated: timestamppb.Now(),
		Data:        make([]uint32, chunkSize),
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(time.Millisecond * 50):
			c := proto.Clone(&chunks).(*cspb.Chunks)

			resp, err := client.SetChunks(ctx, c, grpc.EmptyCallOption{})
			if err != nil {
				log.Fatal().Err(err).Msg("Cannot send chunks. Giving up")
			}

			log.Info().Msgf("Received response: %v", resp)

			chunks.ChunkCount++
		}

	}

}
