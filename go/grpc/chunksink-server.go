package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
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
	config *ChunkSinkServerConfig

	sendCommandFuncMapLock sync.Mutex
	sendCommandFuncMap     map[string]func(*cspb.Command) error
}

func NewChunkSinkServer(config *ChunkSinkServerConfig) *ChunkSinkServer {
	return &ChunkSinkServer{
		config:             config,
		sendCommandFuncMap: make(map[string]func(*cspb.Command) error),
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
	s.sendCommandFuncMapLock.Lock()
	s.sendCommandFuncMap[request.RecorderID] = server.Send
	s.sendCommandFuncMapLock.Unlock()

	<-server.Context().Done()
	s.sendCommandFuncMapLock.Lock()
	delete(s.sendCommandFuncMap, request.RecorderID)
	s.sendCommandFuncMapLock.Unlock()

	return server.Context().Err()
}

func (s *ChunkSinkServer) CutSession(recorderID uuid.UUID) error {
	s.sendCommandFuncMapLock.Lock()
	defer s.sendCommandFuncMapLock.Unlock()

	if sendCommandFunc, ok := s.sendCommandFuncMap[recorderID.String()]; ok {
		return sendCommandFunc(&cspb.Command{Command: &cspb.Command_CmdCutSession{CmdCutSession: &cspb.CmdCutSession{}}})
	}

	return fmt.Errorf("No connection to recorder %s", recorderID)
}
