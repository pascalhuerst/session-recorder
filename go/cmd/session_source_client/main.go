package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pascalhuerst/session-recorder/logger"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	logger.Setup()

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	conn, err := grpc.Dial("recorder.lan:8780", opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to session source. Giving up")
	}
	defer conn.Close()

	client := sspb.NewSessionSourceClient(conn)

	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	if argv := os.Args; len(argv) > 1 {
		switch argv[1] {
		case "list-sessions":
			if len(argv) < 3 {
				log.Fatal().Msg("Missing recorder ID")
			}
			listSessions(ctx, client, argv[2])
		case "list-recorders":
			listRecorders(ctx, client)
		case "delete-session":
			if len(argv) < 4 {
				log.Fatal().Msg("Missing recorder ID or session ID")
			}
			deleteSession(ctx, client, argv[2], argv[3])
		default:
			log.Fatal().Msg("Invalid command")
		}
	} else {
		log.Fatal().Msg("No command specified")
	}
}

func listSessions(ctx context.Context, client sspb.SessionSourceClient, recorderID string) {
	ss, err := client.StreamSessions(ctx, &sspb.StreamSessionRequest{RecorderID: recorderID}, grpc.EmptyCallOption{})
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot stream sessions. Giving up")
	}

	fmt.Printf("Sessions:\n")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			session, err := ss.Recv()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
					return
				}

				log.Fatal().Err(err).Msg("Cannot receive session. Giving up")
			}

			var extraInfo string

			switch si := session.GetInfo().(type) {
			case *sspb.Session_Removed:
				continue
			case *sspb.Session_Updated:
				extraInfo = si.Updated.Name + " (" + si.Updated.GetState().String() + ")"
			}

			fmt.Printf("%s %s\n", session.ID, extraInfo)
		}
	}
}

func listRecorders(ctx context.Context, client sspb.SessionSourceClient) {
	sr, err := client.StreamRecorders(ctx, &sspb.StreamRecordersRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot stream recorders. Giving up")
	}

	fmt.Printf("Recorders:\n")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			recorder, err := sr.Recv()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
					return
				}

				log.Fatal().Err(err).Msg("Cannot receive recorder. Giving up")
			}
			if recorder == nil {
				continue
			}

			fmt.Printf("%s %s\n", recorder.RecorderID, recorder.RecorderName)
		}
	}
}

func deleteSession(ctx context.Context, client sspb.SessionSourceClient, recorderID, sessionID string) {
	req := &sspb.DeleteSessionRequest{
		RecorderID: recorderID,
		SessionID:  sessionID,
	}

	resp, err := client.DeleteSession(ctx, req)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot delete session. Giving up")
	}

	if resp.Success {
		fmt.Printf("Session %s deleted\n", sessionID)
	} else {
		fmt.Printf("Session %s deletion failed: %s\n", sessionID, resp.ErrorMessage)
	}
}
