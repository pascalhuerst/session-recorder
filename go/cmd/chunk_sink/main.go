package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	"github.com/pascalhuerst/session-recorder/email"
	"github.com/pascalhuerst/session-recorder/fileshare"
	"github.com/pascalhuerst/session-recorder/grpc"
	"github.com/pascalhuerst/session-recorder/logger"
	"github.com/pascalhuerst/session-recorder/mdns"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/pascalhuerst/session-recorder/utils"
	"github.com/rs/zerolog/log"
)

const (
	defaultChunkSinkPort     = 8779
	chunkSinkService         = "_session-recorder-chunksink._tcp"
	defaultSessionSourcePort = 8780
	sessionSourceService     = "_session-recorder-sessionsource._tcp"
	defaultLifetime          = 4 * 24 * time.Hour
)

var (
	version string
)

// buildEndpoint constructs an endpoint string from host and port.
// If host is empty, falls back to extracting from fallbackEndpoint.
// If port is 0, falls back to the port from fallbackEndpoint.
func buildEndpoint(host string, port int, fallbackEndpoint string) string {
	// Parse fallback to get defaults
	fallbackHost := fallbackEndpoint
	fallbackPort := ""
	if idx := strings.LastIndex(fallbackEndpoint, ":"); idx != -1 {
		fallbackHost = fallbackEndpoint[:idx]
		fallbackPort = fallbackEndpoint[idx+1:]
	}

	// Use provided values or fallback
	if host == "" {
		host = fallbackHost
	}
	if port == 0 && fallbackPort != "" {
		return host + ":" + fallbackPort
	}
	if port != 0 {
		return fmt.Sprintf("%s:%d", host, port)
	}
	return host
}

func main() {
	// CLI flags with env var fallback
	chunkSinkPort := flag.Int("chunk-sink-port", utils.GetIntWithDefault("CHUNK_SINK_PORT", defaultChunkSinkPort), "Port for ChunkSink gRPC service (env: CHUNK_SINK_PORT)")
	sessionSourcePort := flag.Int("session-source-port", utils.GetIntWithDefault("SESSION_SOURCE_PORT", defaultSessionSourcePort), "Port for SessionSource gRPC service (env: SESSION_SOURCE_PORT)")

	// S3/MinIO configuration
	s3Endpoint := flag.String("s3-endpoint", utils.GetWithDefault("S3_ENDPOINT", "localhost:9000"), "S3/MinIO internal endpoint host:port (env: S3_ENDPOINT)")
	s3AccessKey := flag.String("s3-access-key", utils.GetWithDefault("S3_ACCESS_KEY", ""), "S3 access key (env: S3_ACCESS_KEY)")
	s3SecretKey := flag.String("s3-secret-key", utils.GetWithDefault("S3_SECRET_KEY", ""), "S3 secret key (env: S3_SECRET_KEY)")

	// Local endpoint configuration (for UI/browser URLs)
	s3LocalHost := flag.String("s3-local-host", utils.GetWithDefault("S3_LOCAL_HOST", ""), "S3 host for UI URLs (env: S3_LOCAL_HOST, default: s3-endpoint host)")
	s3LocalPort := flag.Int("s3-local-port", utils.GetIntWithDefault("S3_LOCAL_PORT", 0), "S3 port for UI URLs (env: S3_LOCAL_PORT, default: s3-endpoint port)")

	// Public endpoint configuration (for email sharing URLs)
	s3PublicHost := flag.String("s3-public-host", utils.GetWithDefault("S3_PUBLIC_HOST", ""), "S3 host for email sharing URLs (env: S3_PUBLIC_HOST, default: s3-local-host)")
	s3PublicPort := flag.Int("s3-public-port", utils.GetIntWithDefault("S3_PUBLIC_PORT", 0), "S3 port for email sharing URLs (env: S3_PUBLIC_PORT, default: s3-local-port)")

	flag.Parse()

	// Build endpoint strings from host:port configuration
	s3LocalEndpoint := buildEndpoint(*s3LocalHost, *s3LocalPort, *s3Endpoint)
	s3PublicEndpoint := buildEndpoint(*s3PublicHost, *s3PublicPort, s3LocalEndpoint)

	// Validate required fields
	if *s3AccessKey == "" || *s3SecretKey == "" {
		log.Fatal().Msg("S3_ACCESS_KEY and S3_SECRET_KEY are required (via flags or env vars)")
	}

	ctx := context.Background()

	logger.Setup()

	log.Info().Msg("Setting up storage server")

	minio, err := storage.NewMinioStorage(*s3Endpoint, s3LocalEndpoint, s3PublicEndpoint, *s3AccessKey, *s3SecretKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create storage. Giving up")

		return
	}
	var sessionStorage storage.Storage = minio

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

	// Create broadcasters for fan-out to multiple clients
	// Buffer size of 10 provides headroom for slower consumers
	recorderBroadcaster := broadcast.NewRecorderBroadcaster(10)
	recorderBroadcaster.Start(ctx)
	sessionBroadcaster := broadcast.NewSessionBroadcaster(10)
	audioBroadcaster := broadcast.NewAudioBroadcaster(10)

	chunkSinkHandler := NewChunkSinkHandler(sessionStorage, recorderBroadcaster)

	chunkSinkServer := grpc.NewChunkSinkServer(&grpc.ChunkSinkServerConfig{
		Name:                     hostname,
		Version:                  version,
		OnRecorderStatusCB:       chunkSinkHandler.setRecorderStatus,
		OnChunksCB:               chunkSinkHandler.setChunks,
		OnRecorderConnectedCB:    chunkSinkHandler.OnRecorderConnected,
		OnRecorderDisconnectedCB: chunkSinkHandler.OnRecorderDisconnected,
	})

	// Create email sender if SMTP config is provided
	var emailSender *email.Sender
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost != "" {
		emailConfig := email.Config{
			Host:     smtpHost,
			Port:     utils.GetEnvOrDefault("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     utils.GetEnvOrDefault("SMTP_FROM", "noreply@session-recorder.local"),
			FromName: utils.GetEnvOrDefault("SMTP_FROM_NAME", "Session Recorder"),
		}
		emailSender = email.NewSender(emailConfig)
		log.Info().Str("host", smtpHost).Msg("Email sharing enabled")
	} else {
		log.Warn().Msg("Email sharing disabled (SMTP_HOST not set)")
	}

	// Create file sharer based on environment configuration
	fileSharer, err := fileshare.NewFileSharer(sessionStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create file sharer. Giving up")
		return
	}

	sessionSourceHandler := NewSessionSourceHandler(sessionStorage, chunkSinkServer, recorderBroadcaster, sessionBroadcaster, audioBroadcaster, emailSender, fileSharer)

	sessionSourceServer := grpc.NewSessionSourceServer(&grpc.SessionSourceServerConfig{
		Name:                 hostname,
		Version:              version,
		StreamRecordersCB:    sessionSourceHandler.streamRecorders,
		StreamSessionsCB:     sessionSourceHandler.streamSessions,
		StreamSessionAudioCB: sessionSourceHandler.streamSessionAudio,
		DeleteSessionCB:      sessionSourceHandler.deleteSession,
		SetKeepSessionCB:     sessionSourceHandler.setKeepSession,
		SetNameCB:            sessionSourceHandler.setName,
		CutSessionCB:         sessionSourceHandler.cutSession,
		CreateSegmentCB:      sessionSourceHandler.createSegment,
		UpdateSegmentCB:      sessionSourceHandler.updateSegment,
		DeleteSegmentCB:      sessionSourceHandler.deleteSegment,
		RenderSegmentCB:      sessionSourceHandler.renderSegment,
		ShareSessionCB:       sessionSourceHandler.shareSession,
		ShareSegmentCB:       sessionSourceHandler.shareSegment,
	})

	port, err := grpc.StartProtocolServer(sessionSourceServer, mdnsServer, sessionSourceService, uint16(*sessionSourcePort))
	if err != nil {
		log.Err(err).Msg("Cannot start session source server")

		return
	}
	log.Info().Msgf("Session source server is now being served on port %d", port)

	port, err = grpc.StartProtocolServer(chunkSinkServer, mdnsServer, chunkSinkService, uint16(*chunkSinkPort))
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
