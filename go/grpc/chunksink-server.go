package grpc

import (
	"context"
	"fmt"
	"sync"

	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type ChunkSinkServer struct {
	mutex   sync.Mutex
	version string
	name    string
}

func NewChunkSinkServer(name, version string) *ChunkSinkServer {
	return &ChunkSinkServer{
		name:    name,
		version: version,
	}
}

func (s *ChunkSinkServer) registerGrpcServer(server *grpc.Server) {
	cspb.RegisterChunkSinkServer(server, s)
}

func (s *ChunkSinkServer) serverOptions() []grpc.ServerOption {
	return []grpc.ServerOption{}
}

func (s *ChunkSinkServer) announcement() [][]byte {
	return [][]byte{
		[]byte(fmt.Sprintf("Chunk Sink Server: %s", s.name)),
		[]byte(fmt.Sprintf("Software Version: %s", s.version)),
	}
}

func (s *ChunkSinkServer) StreamChunkData(request *cspb.StreamChunkDataRequest, _ cspb.ChunkSink_StreamChunkDataServer) error {
	log.Debug().Msg("StreamChunkData")

	fmt.Printf("StreamChunkData request: %v\n", request.String())

	return nil
}

func (s *ChunkSinkServer) SetRecorderStatus(ctx context.Context, request *cspb.RecorderStatusRequest) (*cspb.RecorderStatusReply, error) {
	log.Debug().Msg("SetRecorderStatus")

	fmt.Printf("SetRecorderStatus request: %v\n", request.String())

	return &cspb.RecorderStatusReply{}, nil
}
