package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/pascalhuerst/session-recorder/protocols/go/common"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type StreamSessionsCB func(ctx context.Context, request *sspb.StreamSessionRequest, server sspb.SessionSource_StreamSessionsServer) error
type StreamRecordersCB func(ctx context.Context, request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error
type DeleteSessionCB func(ctx context.Context, request *sspb.DeleteSessionRequest) (*cmpb.Respone, error)
type SetKeepSessionCB func(ctx context.Context, request *sspb.SetKeepSessionRequest) (*cmpb.Respone, error)
type SetNameCB func(ctx context.Context, request *sspb.SetNameRequest) (*cmpb.Respone, error)

var noSuccess = &cmpb.Respone{Success: true}

type SessionSourceServerConfig struct {
	Name    string
	Version string

	StreamRecordersCB StreamRecordersCB
	StreamSessionsCB  StreamSessionsCB
	DeleteSessionCB   DeleteSessionCB
	SetKeepSessionCB  SetKeepSessionCB
	SetNameCB         SetNameCB
}

type SessionSourceServer struct {
	mutex  sync.Mutex
	config *SessionSourceServerConfig
}

func NewSessionSourceServer(config *SessionSourceServerConfig) *SessionSourceServer {
	return &SessionSourceServer{
		config: config,
	}
}

func (s *SessionSourceServer) registerGrpcServer(server *grpc.Server) {
	sspb.RegisterSessionSourceServer(server, s)
}

func (s *SessionSourceServer) serverOptions() []grpc.ServerOption {
	return []grpc.ServerOption{}
}

func (s *SessionSourceServer) announcement() [][]byte {
	return [][]byte{
		[]byte(fmt.Sprintf("Session Source Server: %s", s.config.Name)),
		[]byte(fmt.Sprintf("Software Version: %s", s.config.Version)),
	}
}

func (s *SessionSourceServer) StreamRecorders(request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error {
	if s.config.StreamRecordersCB != nil {
		return s.config.StreamRecordersCB(server.Context(), request, server)
	}

	return nil
}

func (s *SessionSourceServer) StreamSessions(request *sspb.StreamSessionRequest, server sspb.SessionSource_StreamSessionsServer) error {
	if s.config.StreamSessionsCB != nil {
		return s.config.StreamSessionsCB(server.Context(), request, server)
	}

	return nil
}

func (s *SessionSourceServer) SetKeepSession(ctx context.Context, in *sspb.SetKeepSessionRequest) (*cmpb.Respone, error) {
	if s.config.SetKeepSessionCB != nil {
		return s.config.SetKeepSessionCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) DeleteSession(ctx context.Context, in *sspb.DeleteSessionRequest) (*cmpb.Respone, error) {
	if s.config.DeleteSessionCB != nil {
		return s.config.DeleteSessionCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) SetName(ctx context.Context, in *sspb.SetNameRequest) (*cmpb.Respone, error) {
	if s.config.SetNameCB != nil {
		return s.config.SetNameCB(ctx, in)
	}

	return noSuccess, nil
}

// Segment API
func (s *SessionSourceServer) CreateSegment(ctx context.Context, in *sspb.CreateSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("CreateSegment not implemented")

	return noSuccess, nil
}

func (s *SessionSourceServer) DeleteSegment(ctx context.Context, in *sspb.DeleteSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("DeleteSegment not implemented")

	return noSuccess, nil
}

func (s *SessionSourceServer) RenderSegment(ctx context.Context, in *sspb.RenderSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("RenderSegment not implemented")

	return noSuccess, nil
}

func (s *SessionSourceServer) UpdateSegment(ctx context.Context, in *sspb.UpdateSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("UpdateSegment not implemented")

	return noSuccess, nil
}
