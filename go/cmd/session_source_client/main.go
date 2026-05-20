package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pascalhuerst/session-recorder/logger"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  session_source_client list-recorders")
	fmt.Fprintln(os.Stderr, "  session_source_client list-sessions <recorder-id>")
	fmt.Fprintln(os.Stderr, "  session_source_client delete-session <recorder-id> <session-id>")
	fmt.Fprintln(os.Stderr, "  session_source_client watch-recorders [recorder-id]")
	os.Exit(2)
}

func main() {
	logger.Setup()

	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial("localhost:8780", opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to session source. Giving up")
	}
	defer conn.Close()

	client := sspb.NewSessionSourceClient(conn)

	argv := os.Args
	if len(argv) < 2 {
		usage()
	}

	switch argv[1] {
	case "list-sessions":
		if len(argv) < 3 {
			log.Fatal().Msg("Missing recorder ID")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		listSessions(ctx, client, argv[2])

	case "list-recorders":
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		listRecorders(ctx, client)

	case "delete-session":
		if len(argv) < 4 {
			log.Fatal().Msg("Missing recorder ID or session ID")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		deleteSession(ctx, client, argv[2], argv[3])

	case "watch-recorders":
		filter := ""
		if len(argv) >= 3 {
			filter = argv[2]
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		watchRecorders(ctx, client, filter)

	default:
		usage()
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
	seen := map[string]struct{}{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			rec, err := sr.Recv()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
					return
				}
				log.Fatal().Err(err).Msg("Cannot receive recorder. Giving up")
			}
			if rec == nil {
				continue
			}
			if _, ok := seen[rec.RecorderID]; ok {
				continue
			}
			seen[rec.RecorderID] = struct{}{}
			fmt.Printf("%s %s%s\n", rec.RecorderID, rec.RecorderName, formatRecorderInfo(rec))
		}
	}
}

// watchRecorders subscribes to the recorder stream indefinitely and prints
// every update. If filter is non-empty, only updates for that recorder ID
// are shown.
func watchRecorders(ctx context.Context, client sspb.SessionSourceClient, filter string) {
	sr, err := client.StreamRecorders(ctx, &sspb.StreamRecordersRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot stream recorders. Giving up")
	}

	if filter == "" {
		fmt.Println("Watching all recorders (Ctrl-C to stop)…")
	} else {
		fmt.Printf("Watching recorder %s (Ctrl-C to stop)…\n", filter)
	}

	for {
		rec, err := sr.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Fatal().Err(err).Msg("Cannot receive recorder. Giving up")
		}
		if rec == nil {
			continue
		}
		if filter != "" && rec.RecorderID != filter {
			continue
		}

		ts := time.Now().Format("15:04:05.000")
		fmt.Printf("%s %s %s%s\n", ts, rec.RecorderID, rec.RecorderName, formatRecorderInfo(rec))
	}
}

// formatRecorderInfo renders the Recorder.info oneof into a printable suffix.
func formatRecorderInfo(rec *sspb.Recorder) string {
	switch info := rec.GetInfo().(type) {
	case *sspb.Recorder_Status:
		s := info.Status
		return fmt.Sprintf("  signal=%s rms=%.2f%% clip=%v", signalStatusName(s.SignalStatus), s.RmsPercent, s.Clipping)
	case *sspb.Recorder_Removed:
		return "  REMOVED"
	default:
		return ""
	}
}

func signalStatusName(s cmpb.SignalStatus) string {
	switch s {
	case cmpb.SignalStatus_SIGNAL:
		return "SIGNAL"
	case cmpb.SignalStatus_NO_SIGNAL:
		return "NO_SIGNAL"
	default:
		return s.String()
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
