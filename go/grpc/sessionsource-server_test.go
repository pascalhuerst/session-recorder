package grpc

import (
	"context"
	"testing"

	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

func TestNewSessionSourceServer(t *testing.T) {
	config := &SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	}

	server := NewSessionSourceServer(config)

	if server == nil {
		t.Error("NewSessionSourceServer() returned nil")
	}

	if server.config != config {
		t.Error("NewSessionSourceServer() config not set correctly")
	}
}

func TestSessionSourceServer_SetKeepSession_NoCallback(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.SetKeepSessionRequest{
		RecorderID: "test-recorder",
		SessionID:  "test-session",
		Keep:       true,
	}

	resp, err := server.SetKeepSession(context.Background(), request)
	if err != nil {
		t.Errorf("SetKeepSession() error = %v", err)
	}

	if resp == nil {
		t.Error("SetKeepSession() returned nil response")
		return
	}

	if !resp.Success {
		t.Error("SetKeepSession() response.Success = false, want true")
	}
}

func TestSessionSourceServer_SetKeepSession_WithCallback(t *testing.T) {
	callbackCalled := false
	expectedRecorderID := "test-recorder"

	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
		SetKeepSessionCB: func(ctx context.Context, request *sspb.SetKeepSessionRequest) (*cmpb.Respone, error) {
			callbackCalled = true
			if request.RecorderID != expectedRecorderID {
				t.Errorf("callback RecorderID = %v, want %v", request.RecorderID, expectedRecorderID)
			}
			return &cmpb.Respone{Success: true}, nil
		},
	})

	request := &sspb.SetKeepSessionRequest{
		RecorderID: expectedRecorderID,
		SessionID:  "test-session",
		Keep:       true,
	}

	resp, err := server.SetKeepSession(context.Background(), request)
	if err != nil {
		t.Errorf("SetKeepSession() error = %v", err)
	}

	if !callbackCalled {
		t.Error("SetKeepSession() callback not called")
	}

	if !resp.Success {
		t.Error("SetKeepSession() response.Success = false, want true")
	}
}

func TestSessionSourceServer_DeleteSession_NoCallback(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.DeleteSessionRequest{
		RecorderID: "test-recorder",
		SessionID:  "test-session",
	}

	resp, err := server.DeleteSession(context.Background(), request)
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	if resp == nil {
		t.Error("DeleteSession() returned nil response")
		return
	}

	if !resp.Success {
		t.Error("DeleteSession() response.Success = false, want true")
	}
}

func TestSessionSourceServer_DeleteSession_WithCallback(t *testing.T) {
	callbackCalled := false
	expectedRecorderID := "test-recorder"

	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
		DeleteSessionCB: func(ctx context.Context, request *sspb.DeleteSessionRequest) (*cmpb.Respone, error) {
			callbackCalled = true
			if request.RecorderID != expectedRecorderID {
				t.Errorf("callback RecorderID = %v, want %v", request.RecorderID, expectedRecorderID)
			}
			return &cmpb.Respone{Success: true}, nil
		},
	})

	request := &sspb.DeleteSessionRequest{
		RecorderID: expectedRecorderID,
		SessionID:  "test-session",
	}

	resp, err := server.DeleteSession(context.Background(), request)
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	if !callbackCalled {
		t.Error("DeleteSession() callback not called")
	}

	if !resp.Success {
		t.Error("DeleteSession() response.Success = false, want true")
	}
}

func TestSessionSourceServer_SetName_NoCallback(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.SetNameRequest{
		RecorderID: "test-recorder",
		SessionID:  "test-session",
		Name:       "New Session Name",
	}

	resp, err := server.SetName(context.Background(), request)
	if err != nil {
		t.Errorf("SetName() error = %v", err)
	}

	if resp == nil {
		t.Error("SetName() returned nil response")
		return
	}

	if !resp.Success {
		t.Error("SetName() response.Success = false, want true")
	}
}

func TestSessionSourceServer_SetName_WithCallback(t *testing.T) {
	callbackCalled := false
	expectedName := "New Session Name"

	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
		SetNameCB: func(ctx context.Context, request *sspb.SetNameRequest) (*cmpb.Respone, error) {
			callbackCalled = true
			if request.Name != expectedName {
				t.Errorf("callback Name = %v, want %v", request.Name, expectedName)
			}
			return &cmpb.Respone{Success: true}, nil
		},
	})

	request := &sspb.SetNameRequest{
		RecorderID: "test-recorder",
		SessionID:  "test-session",
		Name:       expectedName,
	}

	resp, err := server.SetName(context.Background(), request)
	if err != nil {
		t.Errorf("SetName() error = %v", err)
	}

	if !callbackCalled {
		t.Error("SetName() callback not called")
	}

	if !resp.Success {
		t.Error("SetName() response.Success = false, want true")
	}
}

func TestSessionSourceServer_CutSession_NoCallback(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.CutSessionRequest{
		RecorderID: "test-recorder",
	}

	resp, err := server.CutSession(context.Background(), request)
	if err != nil {
		t.Errorf("CutSession() error = %v", err)
	}

	if resp == nil {
		t.Error("CutSession() returned nil response")
		return
	}

	if !resp.Success {
		t.Error("CutSession() response.Success = false, want true")
	}
}

func TestSessionSourceServer_CutSession_WithCallback(t *testing.T) {
	callbackCalled := false
	expectedRecorderID := "test-recorder"

	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
		CutSessionCB: func(ctx context.Context, request *sspb.CutSessionRequest) (*cmpb.Respone, error) {
			callbackCalled = true
			if request.RecorderID != expectedRecorderID {
				t.Errorf("callback RecorderID = %v, want %v", request.RecorderID, expectedRecorderID)
			}
			return &cmpb.Respone{Success: true}, nil
		},
	})

	request := &sspb.CutSessionRequest{
		RecorderID: expectedRecorderID,
	}

	resp, err := server.CutSession(context.Background(), request)
	if err != nil {
		t.Errorf("CutSession() error = %v", err)
	}

	if !callbackCalled {
		t.Error("CutSession() callback not called")
	}

	if !resp.Success {
		t.Error("CutSession() response.Success = false, want true")
	}
}

func TestSessionSourceServer_CreateSegment(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.CreateSegmentRequest{
		SessionID: "test-session",
	}

	resp, err := server.CreateSegment(context.Background(), request)
	if err != nil {
		t.Errorf("CreateSegment() error = %v", err)
	}

	if resp == nil {
		t.Error("CreateSegment() returned nil response")
		return
	}
}

func TestSessionSourceServer_DeleteSegment(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.DeleteSegmentRequest{
		SessionID: "test-session",
		SegmentID: "test-segment",
	}

	resp, err := server.DeleteSegment(context.Background(), request)
	if err != nil {
		t.Errorf("DeleteSegment() error = %v", err)
	}

	if resp == nil {
		t.Error("DeleteSegment() returned nil response")
		return
	}
}

func TestSessionSourceServer_RenderSegment(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.RenderSegmentRequest{
		SessionID: "test-session",
		SegmentID: "test-segment",
	}

	resp, err := server.RenderSegment(context.Background(), request)
	if err != nil {
		t.Errorf("RenderSegment() error = %v", err)
	}

	if resp == nil {
		t.Error("RenderSegment() returned nil response")
		return
	}
}

func TestSessionSourceServer_UpdateSegment(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.UpdateSegmentRequest{
		SessionID: "test-session",
		SegmentID: "test-segment",
	}

	resp, err := server.UpdateSegment(context.Background(), request)
	if err != nil {
		t.Errorf("UpdateSegment() error = %v", err)
	}

	if resp == nil {
		t.Error("UpdateSegment() returned nil response")
		return
	}
}

func TestSessionSourceServer_announcement(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	announcement := server.announcement()
	if len(announcement) != 2 {
		t.Errorf("announcement() returned %d items, want 2", len(announcement))
	}
}

func TestSessionSourceServer_serverOptions(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	options := server.serverOptions()
	if options == nil {
		t.Error("serverOptions() returned nil")
	}
}

func TestSessionSourceServer_StreamRecorders_NoCallback(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.StreamRecordersRequest{}

	// Without a callback, should return nil
	err := server.StreamRecorders(request, nil)
	if err != nil {
		t.Errorf("StreamRecorders() error = %v", err)
	}
}

func TestSessionSourceServer_StreamSessions_NoCallback(t *testing.T) {
	server := NewSessionSourceServer(&SessionSourceServerConfig{
		Name:    "Test Server",
		Version: "1.0.0",
	})

	request := &sspb.StreamSessionRequest{}

	// Without a callback, should return nil
	err := server.StreamSessions(request, nil)
	if err != nil {
		t.Errorf("StreamSessions() error = %v", err)
	}
}
