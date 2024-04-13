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

type SessionSourceServer struct {
	mutex   sync.Mutex
	version string
	name    string

	streamRecordersCB StreamRecordersCB
	streamSessionsCB  StreamSessionsCB
}

func NewSessionSourceServer(name, version string, streamRecordersCB StreamRecordersCB, streamSessionsCB StreamSessionsCB) *SessionSourceServer {
	return &SessionSourceServer{
		name:              name,
		version:           version,
		streamRecordersCB: streamRecordersCB,
		streamSessionsCB:  streamSessionsCB,
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
		[]byte(fmt.Sprintf("Session Source Server: %s", s.name)),
		[]byte(fmt.Sprintf("Software Version: %s", s.version)),
	}
}

func (s *SessionSourceServer) StreamRecorders(request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error {
	if s.streamRecordersCB != nil {
		return s.streamRecordersCB(server.Context(), request, server)
	}

	return nil
}

func (s *SessionSourceServer) StreamSessions(request *sspb.StreamSessionRequest, server sspb.SessionSource_StreamSessionsServer) error {
	if s.streamSessionsCB != nil {
		return s.streamSessionsCB(server.Context(), request, server)
	}

	return nil
}

func (s *SessionSourceServer) SetKeepSession(ctx context.Context, in *sspb.SetKeepSessionRequest) (*cmpb.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("SetKeepSession not implemented")

	return nil, nil
}

func (s *SessionSourceServer) DeleteSession(ctx context.Context, in *sspb.DeleteSessionRequest) (*cmpb.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("DeleteSession not implemented")

	return nil, nil
}

func (s *SessionSourceServer) SetName(ctx context.Context, in *sspb.SetNameRequest) (*cmpb.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Str("name", in.GetName()).Msg("SetName not implemented")

	return nil, nil
}

// Segment API
func (s *SessionSourceServer) CreateSegment(ctx context.Context, in *sspb.CreateSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("CreateSegment not implemented")

	return nil, nil
}

func (s *SessionSourceServer) DeleteSegment(ctx context.Context, in *sspb.DeleteSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("DeleteSegment not implemented")

	return nil, nil
}

func (s *SessionSourceServer) RenderSegment(ctx context.Context, in *sspb.RenderSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("RenderSegment not implemented")

	return nil, nil
}

func (s *SessionSourceServer) UpdateSegment(ctx context.Context, in *sspb.UpdateSegmentRequest) (*common.Respone, error) {
	log.Warn().Str("session-id", in.GetSessionID()).Msg("UpdateSegment not implemented")

	return nil, nil
}
