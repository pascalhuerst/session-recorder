package grpc

import (
	"context"
	"fmt"
	"sync"

	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	"google.golang.org/grpc"
)

type OnRecorderStatusCB func(ctx context.Context, status *cmpb.RecorderStatus) error
type OnChunksCB func(ctx context.Context, chunks *cspb.Chunks) error

type ChunkSinkServer struct {
	mutex   sync.Mutex
	version string
	name    string

	onRecorderStatusCB OnRecorderStatusCB
	onChunksCB         OnChunksCB
}

func NewChunkSinkServer(name, version string, onRecorderStatusCB OnRecorderStatusCB, onChunkCB OnChunksCB) *ChunkSinkServer {
	return &ChunkSinkServer{
		name:               name,
		version:            version,
		onRecorderStatusCB: onRecorderStatusCB,
		onChunksCB:         onChunkCB,
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

func (s *ChunkSinkServer) SetRecorderStatus(ctx context.Context, in *cmpb.RecorderStatus) (*cmpb.Respone, error) {
	if s.onRecorderStatusCB != nil {
		err := s.onRecorderStatusCB(ctx, in)
		if err != nil {
			response := &cmpb.Respone{
				Success:      false,
				ErrorMessage: err.Error(),
			}

			return response, err
		}
	}

	return &cmpb.Respone{
		Success: true,
	}, nil
}

func (s *ChunkSinkServer) SetChunks(ctx context.Context, in *cspb.Chunks) (*cmpb.Respone, error) {
	if s.onChunksCB != nil {
		err := s.onChunksCB(ctx, in)
		if err != nil {
			response := &cmpb.Respone{
				Success:      false,
				ErrorMessage: err.Error(),
			}

			return response, err
		}
	}

	return &cmpb.Respone{
		Success: true,
	}, nil
}
