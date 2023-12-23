package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	"google.golang.org/grpc"
)

func main() {

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	conn, err := grpc.Dial("127.0.0.1:8779", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := cspb.NewChunkSinkClient(conn)

	status := &cspb.RecorderStatusRequest{
		Name:   "Paso",
		Uuid:   uuid.NewString(),
		Status: cspb.AudioInputStatus_NO_SIGNAL,
	}

	for {
		reply, err := client.SetRecorderStatus(context.Background(), status)
		if err != nil {
			log.Fatalf("fail to dial: %v", err)
		}

		log.Printf("Recorder status: %v", reply)

		time.Sleep(1 * time.Second)
	}
}
