package main

import (
	"context"

	"github.com/google/uuid"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	success   = &cmpb.Respone{Success: true}
	noSuccess = &cmpb.Respone{Success: false}
)

type SessionSourceHandler struct {
	sessionStorage   storage.Storage
	recorderUpdateCh chan *sspb.Recorder
	sessionUpdateCh  chan *sspb.Session
}

func NewSessionSourceHandler(sessionStorage storage.Storage, recorderUpdateCh chan *sspb.Recorder, sessionUpdateCh chan *sspb.Session) *SessionSourceHandler {
	h := &SessionSourceHandler{
		sessionStorage:   sessionStorage,
		recorderUpdateCh: recorderUpdateCh,
		sessionUpdateCh:  sessionUpdateCh,
	}

	sessionStorage.RegisterOnSessionClosedCallback(
		func(session *storage.Session) {
			h.onSessionClosed(session)
		},
	)

	return h
}

// Called after a session has been closed and rendered by storage. Setup above in the constructor
func (h *SessionSourceHandler) onSessionClosed(session *storage.Session) {
	// TODO: Refine this
	h.sessionUpdateCh <- &sspb.Session{
		ID: session.ID.String(),
		Info: &sspb.Session_Updated{
			Updated: &sspb.SessionInfo{
				TimeCreated:      timestamppb.New(session.StartTime),
				TimeFinished:     timestamppb.New(session.EndTime),
				Lifetime:         durationpb.New(defaultLifetime), //TODO: This info needs to be stored in the session
				Name:             session.Name,
				AudioFileName:    "data.ogg",
				WaveformDataFile: "waveform.dat",
				Keep:             session.Keep,
				State:            sspb.SessionState_SESSION_STATE_FINISHED,
				Segments:         []*sspb.Segment{},
			},
		},
	}
}

func (h *SessionSourceHandler) streamRecorders(ctx context.Context, request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error {
	log.Debug().Msg("Streaming recorders")

	recorders := h.sessionStorage.GetRecorders()

	for _, recorder := range recorders {
		if err := server.SendMsg(
			&sspb.Recorder{
				RecorderID:   recorder.ID.String(),
				RecorderName: recorder.Name,
				Info: &sspb.Recorder_Status{
					Status: &cmpb.RecorderStatus{
						RecorderID:   recorder.ID.String(),
						RecorderName: recorder.Name,
						SignalStatus: cmpb.SignalStatus_UNKNOWN,
						RmsPercent:   0.0,
						Clipping:     false,
					},
				},
			},
		); err != nil {
			log.Err(err).Msg("Cannot send recorder data")
		}
	}

	for {
		select {
		case recorder := <-h.recorderUpdateCh:
			if err := server.SendMsg(recorder); err != nil {
				log.Err(err).Msg("Cannot send recorder data")
			}
		case <-ctx.Done():
			log.Debug().Msg("Done streaming recorders")

			return nil
		}
	}
}

func (h *SessionSourceHandler) streamSessions(ctx context.Context, request *sspb.StreamSessionRequest, server sspb.SessionSource_StreamSessionsServer) error {
	log.Debug().Msg("Streaming sessions")

	recorderID, err := uuid.Parse(request.RecorderID)
	if err != nil {
		log.Err(err).Str("recorder-id", request.RecorderID).Msg("Cannot parse recorder ID")

		return err
	}

	sessions := h.sessionStorage.GetSessions(recorderID)
	for _, session := range sessions {
		if session.IsClosed {
			if err := server.SendMsg(
				&sspb.Session{
					ID: session.ID.String(),
					Info: &sspb.Session_Updated{
						Updated: &sspb.SessionInfo{
							TimeCreated:      timestamppb.New(session.StartTime),
							TimeFinished:     timestamppb.New(session.EndTime),
							Lifetime:         durationpb.New(defaultLifetime),
							Name:             session.Name,
							AudioFileName:    "data.ogg",
							WaveformDataFile: "waveform.dat",
							Keep:             session.Keep,
							// If the session is closed, the state must be finished
							State: sspb.SessionState_SESSION_STATE_FINISHED,
						},
					},
				},
			); err != nil {
				log.Err(err).Msg("Cannot send session data")
			}
		}
	}

	for {
		select {
		case session := <-h.sessionUpdateCh:
			if err := server.SendMsg(session); err != nil {
				log.Err(err).Msg("Cannot send session data")
			}
		case <-ctx.Done():
			log.Debug().Msg("Done streaming sessions")

			return nil
		}
	}
}

func (h *SessionSourceHandler) cutSession(ctx context.Context, request *sspb.CutSessionRequest) (*cmpb.Respone, error) {
	return noSuccess, nil
}

func parseIDs(recorderID string, sessionID string) (uuid.UUID, uuid.UUID, error) {
	recorderIDParsed, err := uuid.Parse(recorderID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	sessionIDParsed, err := uuid.Parse(sessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return recorderIDParsed, sessionIDParsed, nil
}

func (h *SessionSourceHandler) deleteSession(ctx context.Context, request *sspb.DeleteSessionRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, err := parseIDs(request.RecorderID, request.SessionID)
	if err != nil {
		log.Err(err).Str("recorder-id", request.RecorderID).Str("session-id", request.SessionID).Msg("Cannot parse IDs")

		return noSuccess, err
	}

	if err := h.sessionStorage.DeleteSession(ctx, recorderID, sessionID); err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot delete session")

		return noSuccess, err
	}

	h.sessionUpdateCh <- &sspb.Session{
		ID:   sessionID.String(),
		Info: &sspb.Session_Removed{Removed: &sspb.SessionRemoved{}},
	}

	return success, nil
}

func (h *SessionSourceHandler) setKeepSession(ctx context.Context, request *sspb.SetKeepSessionRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, err := parseIDs(request.RecorderID, request.SessionID)
	if err != nil {
		log.Err(err).Str("recorder-id", request.RecorderID).Str("session-id", request.SessionID).Msg("Cannot parse IDs")

		return noSuccess, err
	}

	if err := h.sessionStorage.SetKeepSession(ctx, recorderID, sessionID, request.Keep); err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot set keep session")

		return noSuccess, err
	}

	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")

		return noSuccess, err
	}

	h.sessionUpdateCh <- &sspb.Session{
		ID: session.ID.String(),
		Info: &sspb.Session_Updated{
			Updated: &sspb.SessionInfo{
				TimeCreated:      timestamppb.New(session.StartTime),
				TimeFinished:     timestamppb.New(session.EndTime),
				Lifetime:         durationpb.New(defaultLifetime),
				Name:             session.Name,
				AudioFileName:    "data.ogg",
				WaveformDataFile: "waveform.dat",
				Keep:             session.Keep,
				// If we can modify the session in the frontend, it's closed
				// we might have to rework that though
				State: sspb.SessionState_SESSION_STATE_FINISHED,
			},
		},
	}

	log.Info().Str("session-id", request.SessionID).Bool("keep", request.Keep).Msg("Set keep session")

	return success, nil
}

func (h *SessionSourceHandler) setName(ctx context.Context, request *sspb.SetNameRequest) (*cmpb.Respone, error) {
	return noSuccess, nil
}
