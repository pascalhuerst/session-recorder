package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type OnRecorderStatusCB func(ctx context.Context, status *cmpb.RecorderStatus) error
type OnChunksCB func(ctx context.Context, chunks *cspb.Chunks) error

type ChunkSinkServerConfig struct {
	Name    string
	Version string

	OnRecorderStatusCB OnRecorderStatusCB
	OnChunksCB         OnChunksCB
}

type ChunkSinkServer struct {
	mutex  sync.Mutex
	config *ChunkSinkServerConfig
}

func NewChunkSinkServer(config *ChunkSinkServerConfig) *ChunkSinkServer {
	return &ChunkSinkServer{
		config: config,
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
		[]byte(fmt.Sprintf("Chunk Sink Server: %s", s.config.Name)),
		[]byte(fmt.Sprintf("Software Version: %s", s.config.Version)),
	}
}

func (s *ChunkSinkServer) SetRecorderStatus(ctx context.Context, in *cmpb.RecorderStatus) (*cmpb.Respone, error) {
	if s.config.OnRecorderStatusCB != nil {
		err := s.config.OnRecorderStatusCB(ctx, in)
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
	if s.config.OnChunksCB != nil {
		err := s.config.OnChunksCB(ctx, in)
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

func (s *ChunkSinkServer) GetCommands(request *cspb.GetCommandRequest, server cspb.ChunkSink_GetCommandsServer) error {
	for {
		select {
		case <-server.Context().Done():
			return server.Context().Err()
		case <-time.Tick(2 * time.Minute):
			log.Info().Msg("Sending cut session command")

			server.Send(&cspb.Command{
				Command: &cspb.Command_CmdCutSession{CmdCutSession: &cspb.CmdCutSession{}},
			})
		}
	}
}
