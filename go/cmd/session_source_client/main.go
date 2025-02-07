package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"google.golang.org/grpc"
)

func main() {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	conn, err := grpc.Dial("127.0.0.1:8780", opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to session source. Giving up")
	}
	defer conn.Close()

	client := sspb.NewSessionSourceClient(conn)

	sr, err := client.StreamRecorders(context.Background(), &sspb.StreamRecordersRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot stream recorders. Giving up")
	}

	go func() {
		for {
			recorder, err := sr.Recv()
			if err != nil {
				log.Fatal().Err(err).Msg("Cannot receive recorder. Giving up")
			}

			if recorder.RecorderID == "9ea55551-5b65-4c0c-9d91-531053677a79" {
				streamSessions(client, recorder)
			}
		}
	}()

	select {}
}

func streamSessions(client sspb.SessionSourceClient, recorder *sspb.Recorder) {
	ss, err := client.StreamSessions(context.Background(), &sspb.StreamSessionRequest{RecorderID: recorder.RecorderID}, grpc.EmptyCallOption{})
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot stream sessions. Giving up")
	}

	fmt.Printf("Recorder: %v\n", recorder)

	for {
		session, err := ss.Recv()
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot receive session. Giving up")
		}

		//time.Sleep(1 * time.Second)
		//
		//log.Info().Msgf("Setting keep for session: %v", session.ID)
		//
		//client.SetKeepSession(context.Background(), &sspb.SetKeepSessionRequest{RecorderID: recorder.RecorderID, SessionID: session.ID, Keep: true}, grpc.EmptyCallOption{})

		fmt.Printf("  Session: %v\n", session)
	}
}
