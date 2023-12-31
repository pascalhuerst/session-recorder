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

			fmt.Printf("Received recorder: %v\n", recorder)

			go func() {
				ss, err := client.StreamSessions(context.Background(), &sspb.StreamSessionRequest{RecorderID: recorder.RecorderID}, grpc.EmptyCallOption{})
				if err != nil {
					log.Fatal().Err(err).Msg("Cannot stream sessions. Giving up")
				}

				for {
					session, err := ss.Recv()
					if err != nil {
						log.Fatal().Err(err).Msg("Cannot receive session. Giving up")
					}

					fmt.Printf("Received session: %v\n", session)

				}
			}()
		}
	}()

	select {}

}
