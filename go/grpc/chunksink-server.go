package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// Callbacks that can be provided by the caller to react to events.
type OnRecorderStatusCB func(ctx context.Context, status *cmpb.RecorderStatus) error
type OnChunksCB func(ctx context.Context, chunks *cspb.Chunks) error
type OnRecorderConnectedCB func(recorderID uuid.UUID)
type OnRecorderDisconnectedCB func(recorderID uuid.UUID)

// ChunkSinkServerConfig contains all configuration and optional callbacks.
type ChunkSinkServerConfig struct {
	Name    string
	Version string

	OnRecorderStatusCB       OnRecorderStatusCB
	OnChunksCB               OnChunksCB
	OnRecorderConnectedCB    OnRecorderConnectedCB
	OnRecorderDisconnectedCB OnRecorderDisconnectedCB
}

// ChunkSinkServer implements the ChunkSink gRPC service and keeps track of
// active recorder connections so that commands (CutSession, etc.) can be routed.
type ChunkSinkServer struct {
	config *ChunkSinkServerConfig

	mu              sync.Mutex
	sendCommandFunc map[uuid.UUID]func(*cspb.Command) error
}

// NewChunkSinkServer constructs a new server instance with the provided config.
func NewChunkSinkServer(config *ChunkSinkServerConfig) *ChunkSinkServer {
	return &ChunkSinkServer{
		config:          config,
		sendCommandFunc: make(map[uuid.UUID]func(*cspb.Command) error),
	}
}

// registerGrpcServer registers the service implementation with a gRPC server.
func (s *ChunkSinkServer) registerGrpcServer(server *grpc.Server) {
	cspb.RegisterChunkSinkServer(server, s)
}

// serverOptions allows future customization of server options.
func (s *ChunkSinkServer) serverOptions() []grpc.ServerOption {
	return []grpc.ServerOption{}
}

// announcement data is published via mDNS to advertise the service.
func (s *ChunkSinkServer) announcement() [][]byte {
	return [][]byte{
		[]byte(fmt.Sprintf("Chunk Sink Server: %s", s.config.Name)),
		[]byte(fmt.Sprintf("Software Version: %s", s.config.Version)),
	}
}

// SetRecorderStatus processes status updates from recorders.
func (s *ChunkSinkServer) SetRecorderStatus(ctx context.Context, in *cmpb.RecorderStatus) (*cmpb.Respone, error) {
	if s.config.OnRecorderStatusCB != nil {
		if err := s.config.OnRecorderStatusCB(ctx, in); err != nil {
			return &cmpb.Respone{Success: false, ErrorMessage: err.Error()}, err
		}
	}

	return &cmpb.Respone{Success: true}, nil
}

// SetChunks processes incoming audio chunks from recorders.
func (s *ChunkSinkServer) SetChunks(ctx context.Context, in *cspb.Chunks) (*cmpb.Respone, error) {
	if s.config.OnChunksCB != nil {
		if err := s.config.OnChunksCB(ctx, in); err != nil {
			return &cmpb.Respone{Success: false, ErrorMessage: err.Error()}, err
		}
	}

	return &cmpb.Respone{Success: true}, nil
}

// GetCommands establishes a command stream for a recorder. The stream's send
// function is stored so commands can be routed later (e.g., CutSession).
func (s *ChunkSinkServer) GetCommands(request *cspb.GetCommandRequest, server cspb.ChunkSink_GetCommandsServer) error {
	recorderID, err := uuid.Parse(request.GetRecorderID())
	if err != nil {
		log.Warn().Str("recorder-id", request.GetRecorderID()).Err(err).Msg("GetCommands: invalid recorder id")
		return fmt.Errorf("invalid recorder id %q: %w", request.GetRecorderID(), err)
	}

	s.mu.Lock()
	s.sendCommandFunc[recorderID] = server.Send
	mapSize := len(s.sendCommandFunc)
	if s.config.OnRecorderConnectedCB != nil {
		go s.config.OnRecorderConnectedCB(recorderID)
	}
	s.mu.Unlock()

	log.Info().
		Stringer("recorder-id", recorderID).
		Int("connected-recorders", mapSize).
		Msg("GetCommands: stream opened")

	<-server.Context().Done()
	ctxErr := server.Context().Err()

	s.mu.Lock()
	delete(s.sendCommandFunc, recorderID)
	mapSize = len(s.sendCommandFunc)
	if s.config.OnRecorderDisconnectedCB != nil {
		go s.config.OnRecorderDisconnectedCB(recorderID)
	}
	s.mu.Unlock()

	log.Info().
		Stringer("recorder-id", recorderID).
		Int("connected-recorders", mapSize).
		AnErr("ctx-err", ctxErr).
		Msg("GetCommands: stream closed")

	return ctxErr
}

// CutSession forwards a cut command to the recorder if a command stream exists.
func (s *ChunkSinkServer) CutSession(recorderID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if send := s.sendCommandFunc[recorderID]; send != nil {
		return send(&cspb.Command{
			Command: &cspb.Command_CmdCutSession{
				CmdCutSession: &cspb.CmdCutSession{},
			},
		})
	}

	return fmt.Errorf("no connection to recorder %s", recorderID)
}
