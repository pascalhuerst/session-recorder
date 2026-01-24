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
type OnRecorderConnectedCB func(recorderID uuid.UUID)
type OnRecorderDisconnectedCB func(recorderID uuid.UUID)

type ChunkSinkServerConfig struct {
	Name    string
	Version string

	OnRecorderStatusCB       OnRecorderStatusCB
	OnChunksCB               OnChunksCB
	OnRecorderConnectedCB    OnRecorderConnectedCB
	OnRecorderDisconnectedCB OnRecorderDisconnectedCB
}

type ChunkSinkServer struct {
	config *ChunkSinkServerConfig

	sendCommandFuncMapLock sync.Mutex
	sendCommandFuncMap     map[uuid.UUID]func(*cspb.Command) error
	connectedRecorders     map[uuid.UUID]struct{}
}

func NewChunkSinkServer(config *ChunkSinkServerConfig) *ChunkSinkServer {
	return &ChunkSinkServer{
		config:             config,
		sendCommandFuncMap: make(map[uuid.UUID]func(*cspb.Command) error),
		connectedRecorders: make(map[uuid.UUID]struct{}),
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
	var recorderID uuid.UUID
	newlyConnected := false

	if in != nil {
		if parsedID, err := uuid.Parse(in.GetRecorderID()); err == nil {
			recorderID = parsedID

			s.sendCommandFuncMapLock.Lock()
			if _, ok := s.connectedRecorders[recorderID]; !ok {
				s.connectedRecorders[recorderID] = struct{}{}
				newlyConnected = true
			}
			s.sendCommandFuncMapLock.Unlock()
		}
	}

	if newlyConnected && s.config.OnRecorderConnectedCB != nil {
		s.config.OnRecorderConnectedCB(recorderID)
	}

	if s.config.OnRecorderStatusCB != nil {
		if err := s.config.OnRecorderStatusCB(ctx, in); err != nil {
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
	recorderID, err := uuid.Parse(request.RecorderID)
	if err != nil {
		return fmt.Errorf("invalid recorder id %q: %w", request.RecorderID, err)
	}

	s.sendCommandFuncMapLock.Lock()
	s.sendCommandFuncMap[recorderID] = server.Send
	s.connectedRecorders[recorderID] = struct{}{}
	s.sendCommandFuncMapLock.Unlock()

	if s.config.OnRecorderConnectedCB != nil {
		s.config.OnRecorderConnectedCB(recorderID)
	}

	<-server.Context().Done()

	s.sendCommandFuncMapLock.Lock()
	delete(s.sendCommandFuncMap, recorderID)
	delete(s.connectedRecorders, recorderID)
	s.sendCommandFuncMapLock.Unlock()

	if s.config.OnRecorderDisconnectedCB != nil {
		s.config.OnRecorderDisconnectedCB(recorderID)
	}

	return server.Context().Err()
}

func (s *ChunkSinkServer) CutSession(recorderID uuid.UUID) error {
	s.sendCommandFuncMapLock.Lock()
	defer s.sendCommandFuncMapLock.Unlock()

	if sendCommandFunc, ok := s.sendCommandFuncMap[recorderID]; ok {
		return sendCommandFunc(&cspb.Command{Command: &cspb.Command_CmdCutSession{CmdCutSession: &cspb.CmdCutSession{}}})
	}

	return fmt.Errorf("no connection to recorder %s", recorderID)
}

func (s *ChunkSinkServer) IsRecorderConnected(recorderID uuid.UUID) bool {
	s.sendCommandFuncMapLock.Lock()
	defer s.sendCommandFuncMapLock.Unlock()

	_, ok := s.connectedRecorders[recorderID]
	return ok
}
