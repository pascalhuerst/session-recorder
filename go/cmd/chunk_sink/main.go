package main

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/grpc"
	"github.com/pascalhuerst/session-recorder/logger"
	"github.com/pascalhuerst/session-recorder/mdns"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/pascalhuerst/session-recorder/utils"
	"github.com/rs/zerolog/log"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

const (
	chunkSinkPort        = 8779
	chunkSinkService     = "_session-recorder-chunksink._tcp"
	sessionSourcePort    = 8780
	sessionSourceService = "_session-recorder-sessionsource._tcp"
	defaultLifetime      = 4 * 24 * time.Hour
)

var (
	version string
)

func main() {
	ctx := context.Background()

	logger.Setup()

	s3Endpoint := utils.MustGet("S3_ENDPOINT")
	s3AccessKey := utils.MustGet("S3_ACCESS_KEY")
	s3SecretKey := utils.MustGet("S3_SECRET_KEY")

	log.Info().Msg("Setting up storage server")

	s, err := storage.NewMinioStorage(s3Endpoint, s3AccessKey, s3SecretKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create storage. Giving up")

		return
	}
	var sessionStorage storage.Storage = s

	if err := sessionStorage.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Cannot start storage. Giving up")

		return
	}

	log.Info().Msg("Starting mdns server")

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

	var recorderUpdateCh chan *sspb.Recorder = make(chan *sspb.Recorder)
	var sessionUpdateCh chan *sspb.Session = make(chan *sspb.Session)

	chunkSinkHandler := NewChunkSinkHandler(sessionStorage, recorderUpdateCh)

	chunkSinkServer := grpc.NewChunkSinkServer(&grpc.ChunkSinkServerConfig{
		Name:               hostname,
		Version:            version,
		OnRecorderStatusCB: chunkSinkHandler.setRecorderStatus,
		OnChunksCB:         chunkSinkHandler.setChunks,
	})

	sessionSourceHandler := NewSessionSourceHandler(sessionStorage, chunkSinkServer, recorderUpdateCh, sessionUpdateCh)

	sessionSourceServer := grpc.NewSessionSourceServer(&grpc.SessionSourceServerConfig{
		Name:              hostname,
		Version:           version,
		StreamRecordersCB: sessionSourceHandler.streamRecorders,
		StreamSessionsCB:  sessionSourceHandler.streamSessions,
		DeleteSessionCB:   sessionSourceHandler.deleteSession,
		SetKeepSessionCB:  sessionSourceHandler.setKeepSession,
		SetNameCB:         sessionSourceHandler.setName,
		CutSessionCB:      sessionSourceHandler.cutSession,
	})

	port, err := grpc.StartProtocolServer(sessionSourceServer, mdnsServer, sessionSourceService, sessionSourcePort)
	if err != nil {
		log.Err(err).Msg("Cannot start session source server")

		return
	}
	log.Info().Msgf("Session source server is now being served on port %d", port)

	port, err = grpc.StartProtocolServer(chunkSinkServer, mdnsServer, chunkSinkService, chunkSinkPort)
	if err != nil {
		log.Err(err).Msg("Cannot start chunk sink server")

		return
	}
	log.Info().Msgf("Chunk sink server is now being served on port %d", port)

	/*
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
					meta, err := sessionStorage.GetSessionMetadata(ctx, recorderID, sessionID)
					if err != nil {
						log.Err(err).Msg("Cannot get session metadata")

						continue
					}

					sessionUpdateCh <- &sspb.Session{
						ID: sessionID.String(),
						Info: &sspb.Session_Updated{
							Updated: &sspb.SessionInfo{
								TimeCreated:      timestamppb.New(meta.StartTime),
								TimeFinished:     timestamppb.New(meta.EndTime),
								Lifetime:         durationpb.New(defaultLifetime),
								Name:             meta.GenericMetadata.Name,
								AudioFileName:    "data.ogg",
								WaveformDataFile: "waveform.dat",
								Keep:             false,
								State:            sspb.SessionState_SESSION_STATE_UNKNOWN,
							},
						},
					}

					fmt.Printf("    %s\n", sessionID)
				}
			}

		}()
	*/

	log.Info().Msg("chunk sink server setup successfully")

	<-ctx.Done()
}
