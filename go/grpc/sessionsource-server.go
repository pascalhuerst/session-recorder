package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/pascalhuerst/session-recorder/protocols/go/common"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"google.golang.org/grpc"
)

type StreamSessionsCB func(ctx context.Context, request *sspb.StreamSessionRequest, server sspb.SessionSource_StreamSessionsServer) error
type StreamRecordersCB func(ctx context.Context, request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error
type StreamSessionAudioCB func(ctx context.Context, request *sspb.StreamSessionAudioRequest, server sspb.SessionSource_StreamSessionAudioServer) error
type DeleteSessionCB func(ctx context.Context, request *sspb.DeleteSessionRequest) (*cmpb.Respone, error)
type SetKeepSessionCB func(ctx context.Context, request *sspb.SetKeepSessionRequest) (*cmpb.Respone, error)
type SetNameCB func(ctx context.Context, request *sspb.SetNameRequest) (*cmpb.Respone, error)
type CutSessionCB func(ctx context.Context, request *sspb.CutSessionRequest) (*cmpb.Respone, error)

type CreateSegmentCB func(ctx context.Context, request *sspb.CreateSegmentRequest) (*cmpb.Respone, error)
type UpdateSegmentCB func(ctx context.Context, request *sspb.UpdateSegmentRequest) (*cmpb.Respone, error)
type DeleteSegmentCB func(ctx context.Context, request *sspb.DeleteSegmentRequest) (*cmpb.Respone, error)
type RenderSegmentCB func(ctx context.Context, request *sspb.RenderSegmentRequest) (*cmpb.Respone, error)

type ShareSessionCB func(ctx context.Context, request *sspb.ShareSessionRequest) (*cmpb.Respone, error)
type ShareSegmentCB func(ctx context.Context, request *sspb.ShareSegmentRequest) (*cmpb.Respone, error)

var noSuccess = &cmpb.Respone{Success: true}

type SessionSourceServerConfig struct {
	Name    string
	Version string

	StreamRecordersCB    StreamRecordersCB
	StreamSessionsCB     StreamSessionsCB
	StreamSessionAudioCB StreamSessionAudioCB
	DeleteSessionCB      DeleteSessionCB
	SetKeepSessionCB     SetKeepSessionCB
	SetNameCB            SetNameCB
	CutSessionCB         CutSessionCB

	CreateSegmentCB CreateSegmentCB
	UpdateSegmentCB UpdateSegmentCB
	DeleteSegmentCB DeleteSegmentCB
	RenderSegmentCB RenderSegmentCB

	ShareSessionCB ShareSessionCB
	ShareSegmentCB ShareSegmentCB
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

func (s *SessionSourceServer) StreamSessionAudio(request *sspb.StreamSessionAudioRequest, server sspb.SessionSource_StreamSessionAudioServer) error {
	if s.config.StreamSessionAudioCB != nil {
		return s.config.StreamSessionAudioCB(server.Context(), request, server)
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

func (s *SessionSourceServer) CutSession(ctx context.Context, in *sspb.CutSessionRequest) (*common.Respone, error) {
	if s.config.CutSessionCB != nil {
		return s.config.CutSessionCB(ctx, in)
	}
	return noSuccess, nil
}

// Segment API
func (s *SessionSourceServer) CreateSegment(ctx context.Context, in *sspb.CreateSegmentRequest) (*common.Respone, error) {
	if s.config.CreateSegmentCB != nil {
		return s.config.CreateSegmentCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) DeleteSegment(ctx context.Context, in *sspb.DeleteSegmentRequest) (*common.Respone, error) {
	if s.config.DeleteSegmentCB != nil {
		return s.config.DeleteSegmentCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) RenderSegment(ctx context.Context, in *sspb.RenderSegmentRequest) (*common.Respone, error) {
	if s.config.RenderSegmentCB != nil {
		return s.config.RenderSegmentCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) UpdateSegment(ctx context.Context, in *sspb.UpdateSegmentRequest) (*common.Respone, error) {
	if s.config.UpdateSegmentCB != nil {
		return s.config.UpdateSegmentCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) ShareSession(ctx context.Context, in *sspb.ShareSessionRequest) (*common.Respone, error) {
	if s.config.ShareSessionCB != nil {
		return s.config.ShareSessionCB(ctx, in)
	}

	return noSuccess, nil
}

func (s *SessionSourceServer) ShareSegment(ctx context.Context, in *sspb.ShareSegmentRequest) (*common.Respone, error) {
	if s.config.ShareSegmentCB != nil {
		return s.config.ShareSegmentCB(ctx, in)
	}

	return noSuccess, nil
}
