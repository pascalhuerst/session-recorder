package grpc

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SessionSourceServer struct {
	mutex   sync.Mutex
	version string
	name    string
}

func NewSessionSourceServer(name, version string) *SessionSourceServer {
	return &SessionSourceServer{
		name:    name,
		version: version,
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

func (s *SessionSourceServer) HealthCheck(request *sspb.HealthCheck, server sspb.SessionSource_StreamRecordersServer) error {
	status := cspb.AudioInputStatus_NO_SIGNAL

	for {

		server.Send(&sspb.RecorderInfo{
			ID:               uuid.NewString(),
			Name:             "Test Recorder 1",
			AudioInputStatus: status,
		})

		server.Send(&sspb.RecorderInfo{
			ID:               uuid.NewString(),
			Name:             "Test Recorder 2",
			AudioInputStatus: status,
		})

		time.Sleep(5 * time.Second)

		if status == cspb.AudioInputStatus_NO_SIGNAL {
			status = cspb.AudioInputStatus_SIGNAL
		} else {
			status = cspb.AudioInputStatus_NO_SIGNAL
		}

	}

	return nil
}

func (s *SessionSourceServer) StreamRecorders(request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error {
	status := cspb.AudioInputStatus_NO_SIGNAL

	for {

		server.Send(&sspb.RecorderInfo{
			ID:               uuid.NewString(),
			Name:             "Test Recorder 1",
			AudioInputStatus: status,
		})

		server.Send(&sspb.RecorderInfo{
			ID:               uuid.NewString(),
			Name:             "Test Recorder 2",
			AudioInputStatus: status,
		})

		time.Sleep(5 * time.Second)

		if status == cspb.AudioInputStatus_NO_SIGNAL {
			status = cspb.AudioInputStatus_SIGNAL
		} else {
			status = cspb.AudioInputStatus_NO_SIGNAL
		}

	}

	return nil
}

func (s *SessionSourceServer) StreamSessions(request *sspb.StreamSeesionRequst, server sspb.SessionSource_StreamSessionsServer) error {
	now := timestamppb.Now()
	earlier := timestamppb.New(now.AsTime().Add(-time.Hour * 72))

	for i := 0; i < 10; i++ {
		server.Send(&sspb.SessionInfo{
			ID:               uuid.NewString(),
			TimeCreated:      earlier,
			TimeFinished:     now,
			Lifetime:         durationpb.New(time.Hour * 72),
			AudioFileName:    fmt.Sprintf("whatever_filename_%d.flac", i),
			WaveformDataFile: fmt.Sprintf("whatever_filename_%d.waveform", i),
			KeepSession:      false,
			State:            0,
		})
	}

	return nil
}
