package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/pascalhuerst/session-recorder/grpc"
	"github.com/pascalhuerst/session-recorder/mdns"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/pascalhuerst/session-recorder/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
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

func main() {
	ctx := context.Background()

	consoleWriter := zerolog.ConsoleWriter{
		TimeFormat: time.StampMicro,
		Out:        colorable.NewColorableStdout(),
	}
	log.Logger = log.Output(consoleWriter)

	s3Endpoint := utils.MustGet("S3_ENDPOINT")
	s3AccessKey := utils.MustGet("S3_ACCESS_KEY")
	s3SecretKey := utils.MustGet("S3_SECRET_KEY")

	s, err := storage.NewMinioStorage(s3Endpoint, s3AccessKey, s3SecretKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create storage. Giving up")

		return
	}
	var sessionStorage storage.Storage = s

	system, err := storage.NewSystem(sessionStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create system model. Giving up")
	}

	for _, recorder := range system.Recorders {
		recorderID, err := uuid.Parse(recorder.Metadata.GenericMetadata.ID)
		if err != nil {
			continue
		}

		sessionStorage.CloseOpenSessions(ctx, recorderID)
	}

	mdnsServer, err := mdns.ServerNew()
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create mdns server. Giving up")

		return
	}

	log.Info().Msg("Storage server setup successfully")

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown_hostname_" + uuid.NewString()
	}

	recorderUpdateCh := make(chan *sspb.Recorder)
	sessionUpdateCh := make(chan *sspb.Session)

	sessionSourceServer := grpc.NewSessionSourceServer(hostname, version,
		// This is called for every frontend client that connects
		func(ctx context.Context, request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error {
			recordersIDs, err := sessionStorage.GetRecorderIDs(ctx)
			if err != nil {
				log.Err(err).Msg("Cannot get recorders")
			}

			for _, recorderID := range recordersIDs {
				if err := server.SendMsg(
					&sspb.Recorder{
						RecorderID:   recorderID.String(),
						RecorderName: "TODO",
						Info: &sspb.Recorder_Status{
							Status: &cmpb.RecorderStatus{
								RecorderID:   recorderID.String(),
								RecorderName: "TODO",
								SignalStatus: 0,
								RmsPercent:   0,
								Clipping:     false,
							},
						},
					},
				); err != nil {
					log.Err(err).Msg("Cannot send recorder data")
				}
			}

			for {
				select {
				case recorder := <-recorderUpdateCh:
					if err := server.SendMsg(recorder); err != nil {
						log.Err(err).Msg("Cannot send recorder data")
					}
				case <-ctx.Done():
					return nil
				}
			}
		},
		func(ctx context.Context, request *sspb.StreamSessionRequest, server sspb.SessionSource_StreamSessionsServer) error {
			recorderID, err := uuid.Parse(request.RecorderID)
			if err != nil {
				log.Err(err).Str("recorder-id", request.RecorderID).Msg("Cannot parse recorder ID")

				return err
			}

			sessionIDs, err := sessionStorage.GetSessionIDs(ctx, recorderID)
			if err != nil {
				log.Err(err).Msg("Cannot get sessions")
			}

			for _, sessionID := range sessionIDs {
				server.SendMsg(
					&sspb.Session{
						ID: sessionID.String(),
						Info: &sspb.Session_Updated{
							Updated: &sspb.SessionInfo{
								//TODO Fill with metadata
							},
						},
					},
				)
			}

			for {
				select {
				case session := <-sessionUpdateCh:
					server.SendMsg(session)
				case <-ctx.Done():
					return nil
				}
			}
		},
	)
	if err != nil {
		log.Err(err).Msg("Cannot create session source server")

		return
	}

	port, err := grpc.StartProtocolServer(sessionSourceServer, mdnsServer, sessionSourceService, sessionSourcePort)
	if err != nil {
		log.Err(err).Msg("Cannot start session source server")

		return
	}
	log.Info().Msgf("Session source server is now being served on port %d", port)

	chunkSinkServer := grpc.NewChunkSinkServer(hostname, version,
		// Recorder Status Updates. These are forwarded to session source clients
		func(ctx context.Context, status *cmpb.RecorderStatus) error {
			recorderID, err := uuid.Parse(status.RecorderID)
			if err != nil {
				log.Err(err).Str("recorder-id", status.RecorderID).Msg("Cannot parse recorder ID")

				return err
			}

			select {
			case recorderUpdateCh <- &sspb.Recorder{
				RecorderID:   recorderID.String(),
				RecorderName: "TODO",
				Info: &sspb.Recorder_Status{
					Status: status,
				},
			}:
			default:
			}

			return nil
		},
		// Chunks are pushed to a storage backend
		func(ctx context.Context, chunks *cspb.Chunks) error {
			chunkID := fmt.Sprintf("%016d", chunks.ChunkCount)
			sessionID, err := uuid.Parse(chunks.SessionID)
			if err != nil {
				log.Err(err).Str("sesstion-id", chunks.SessionID).Msg("Cannot parse session ID")

				return err
			}

			recorderID, err := uuid.Parse(chunks.RecorderID)
			if err != nil {
				log.Err(err).Str("recorder-id", chunks.RecorderID).Msg("Cannot parse recorder ID")

				return err
			}

			samples := make([]int16, 0)

			for _, sample := range chunks.Data {
				samples = append(samples, int16(sample))
			}

			if err = sessionStorage.SafeChunks(ctx, recorderID, sessionID, chunkID, samples); err != nil {
				log.Err(err).Msg("Cannot save chunks")
			}

			return nil
		},
	)

	port, err = grpc.StartProtocolServer(chunkSinkServer, mdnsServer, chunkSinkService, chunkSinkPort)
	if err != nil {
		log.Err(err).Msg("Cannot start chunk sink server")

		return
	}
	log.Info().Msgf("Chunk sink server is now being served on port %d", port)

	go func() {
		time.Sleep(10 * time.Second)

		recorders, err := sessionStorage.GetRecorderIDs(ctx)
		if err != nil {
			log.Err(err).Msg("Cannot get recorders")
		}

		fmt.Printf("Recorders:\n")

		for _, recorderID := range recorders {
			fmt.Printf("  %s\n", recorderID)

			sessions, err := sessionStorage.GetSessionIDs(ctx, recorderID)
			if err != nil {
				log.Err(err).Msg("Cannot get sessions")
			}

			for _, sessionID := range sessions {
				fmt.Printf("    %s\n", sessionID)
			}
		}

	}()

	<-ctx.Done()
}
