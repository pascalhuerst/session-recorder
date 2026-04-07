package main

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	"github.com/pascalhuerst/session-recorder/email"
	"github.com/pascalhuerst/session-recorder/fileshare"
	"github.com/pascalhuerst/session-recorder/grpc"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	success         = &cmpb.Respone{Success: true}
	noSuccess       = &cmpb.Respone{Success: false}
	urlUnsafeChars  = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

type SessionSourceHandler struct {
	sessionStorage      storage.Storage
	chunkSinkServer     *grpc.ChunkSinkServer
	recorderBroadcaster *broadcast.RecorderBroadcaster
	sessionBroadcaster  *broadcast.SessionBroadcaster
	audioBroadcaster    *broadcast.AudioBroadcaster
	peakAccumulator     *broadcast.PeakAccumulator
	peakBroadcaster     *broadcast.PeakBroadcaster
	emailSender         *email.Sender
	fileSharer          fileshare.FileSharer
}

func NewSessionSourceHandler(
	sessionStorage storage.Storage,
	eventBus *storage.EventBus,
	chunkSinkServer *grpc.ChunkSinkServer,
	recorderBroadcaster *broadcast.RecorderBroadcaster,
	sessionBroadcaster *broadcast.SessionBroadcaster,
	audioBroadcaster *broadcast.AudioBroadcaster,
	peakAccumulator *broadcast.PeakAccumulator,
	peakBroadcaster *broadcast.PeakBroadcaster,
	emailSender *email.Sender,
	fileSharer fileshare.FileSharer,
) *SessionSourceHandler {
	h := &SessionSourceHandler{
		sessionStorage:      sessionStorage,
		chunkSinkServer:     chunkSinkServer,
		recorderBroadcaster: recorderBroadcaster,
		sessionBroadcaster:  sessionBroadcaster,
		audioBroadcaster:    audioBroadcaster,
		peakAccumulator:     peakAccumulator,
		peakBroadcaster:     peakBroadcaster,
		emailSender:         emailSender,
		fileSharer:          fileSharer,
	}

	// Register for lifecycle events
	eventBus.AddListener(h)

	// Register callback for audio chunk streaming
	sessionStorage.RegisterOnAudioChunkCallback(
		func(recorderID, sessionID uuid.UUID, samples []int16, chunkNumber int, timestamp time.Time) {
			h.onAudioChunk(recorderID, sessionID, samples, chunkNumber, timestamp)
		},
	)

	return h
}

// mapSessionState converts storage.SessionState to proto SessionState
func mapSessionState(state storage.SessionState) sspb.SessionState {
	switch state {
	case storage.SessionStateRecording:
		return sspb.SessionState_SESSION_STATE_RECORDING
	case storage.SessionStateProcessing:
		return sspb.SessionState_SESSION_STATE_PROCESSING
	case storage.SessionStateFinished:
		return sspb.SessionState_SESSION_STATE_FINISHED
	case storage.SessionStateError:
		return sspb.SessionState_SESSION_STATE_ERROR
	default:
		return sspb.SessionState_SESSION_STATE_UNKNOWN
	}
}

// mapSegmentState converts storage.SegmentState to proto SegmentState
func mapSegmentState(state storage.SegmentState) sspb.SegmentState {
	switch state {
	case storage.SegmentStateQueued:
		return sspb.SegmentState_SEGMENT_STATE_QUEUED
	case storage.SegmentStateRendering:
		return sspb.SegmentState_SEGMENT_STATE_RENDERING
	case storage.SegmentStateFinished:
		return sspb.SegmentState_SEGMENT_STATE_FINISHED
	case storage.SegmentStateError:
		return sspb.SegmentState_SEGMENT_STATE_ERROR
	default:
		return sspb.SegmentState_SEGMENT_STATE_UNKNOWN
	}
}

func getFileURL(ctx context.Context, h *SessionSourceHandler, session *storage.Session, filename storage.Filename, download bool) string {
	// Create URL-friendly session name
	urlFriendlyName := strings.ReplaceAll(session.Name, " ", "_")
	urlFriendlyName = urlUnsafeChars.ReplaceAllString(urlFriendlyName, "")

	// Format date as URL-friendly string
	dateStr := session.StartTime.Format("2006-01-02_15-04-05")

	// Get file extension from filename
	ext := filepath.Ext(string(filename))

	// Construct download filename
	downloadFilename := urlFriendlyName + "_" + dateStr + ext

	fileURL, err := h.sessionStorage.GetPresignedURL(
		ctx,
		storage.AssetOptions{
			RecorderID: session.RecorderID,
			SessionID:  session.ID,
			Filename:   filename,
		},
		storage.SigningOptions{
			Expires:          time.Hour * 24,
			Download:         download,
			DownloadFilename: downloadFilename,
		})

	if err != nil {
		log.Error().Str("filename", string(filename)).Err(err).Msg("Failed to get presigned URL for fileURL")
		return ""
	}

	return fileURL
}

func getSegmentFileURL(ctx context.Context, h *SessionSourceHandler, session *storage.Session, segment *storage.Segment, filename storage.Filename, download bool) string {
	// Create URL-friendly segment name
	urlFriendlyName := strings.ReplaceAll(segment.Comment, " ", "_")
	urlFriendlyName = urlUnsafeChars.ReplaceAllString(urlFriendlyName, "")
	if urlFriendlyName == "" {
		urlFriendlyName = "segment"
	}

	// Get file extension from filename
	ext := filepath.Ext(string(filename))

	// Construct download filename
	downloadFilename := urlFriendlyName + ext

	fileURL, err := h.sessionStorage.GetSegmentPresignedURL(
		ctx,
		storage.SegmentAssetOptions{
			RecorderID: session.RecorderID,
			SessionID:  session.ID,
			SegmentID:  segment.ID,
			Filename:   filename,
		},
		storage.SigningOptions{
			Expires:          time.Hour * 24,
			Download:         download,
			DownloadFilename: downloadFilename,
		})

	if err != nil {
		log.Error().
			Str("filename", string(filename)).
			Stringer("segment-id", segment.ID).
			Err(err).
			Msg("Failed to get presigned URL for segment file")
		return ""
	}

	return fileURL
}

func newSessionInfo(ctx context.Context, h *SessionSourceHandler, session *storage.Session) *sspb.SessionInfo {
	info := &sspb.SessionInfo{
		TimeCreated:  timestamppb.New(session.StartTime),
		Lifetime:     durationpb.New(defaultLifetime), //TODO: This info needs to be stored in the session
		Name:         session.Name,
		Keep:         session.Keep,
		State:        mapSessionState(session.State),
		ErrorMessage: session.ErrorMessage,
		Segments:     []*sspb.Segment{},
	}

	// Set duration if available (estimated at PROCESSING time, exact after render)
	if session.Duration > 0 {
		info.Duration = durationpb.New(session.Duration)
	}


	// Only set TimeFinished for finished sessions
	if session.State == storage.SessionStateFinished && !session.EndTime.IsZero() {
		info.TimeFinished = timestamppb.New(session.EndTime)
	}

	// Only add file URLs for finished sessions (files exist only after rendering)
	if session.State == storage.SessionStateFinished {
		getURL := func(filename storage.Filename, download bool) string {
			return getFileURL(ctx, h, session, filename, download)
		}

		info.InlineFiles = &sspb.SessionInfo_Files{
			Ogg:      getURL(storage.FILENAME_OGG, false),
			Flac:     getURL(storage.FILENAME_FLAC, false),
			Waveform: getURL(storage.FILENAME_WAVEFORM, false),
		}
		info.DownloadFiles = &sspb.SessionInfo_Files{
			Ogg:      getURL(storage.FILENAME_OGG, true),
			Flac:     getURL(storage.FILENAME_FLAC, true),
			Waveform: getURL(storage.FILENAME_WAVEFORM, true),
		}
	}

	// Add segments with their file URLs
	for segmentID, segment := range session.Segments {
		segmentInfo := &sspb.SegmentInfo{
			TimeStart:    timestamppb.New(session.StartTime.Add(time.Duration(segment.StartPoint) * time.Second / 48000)),
			TimeEnd:      timestamppb.New(session.StartTime.Add(time.Duration(segment.EndPoint) * time.Second / 48000)),
			Name:         segment.Comment,
			State:        mapSegmentState(segment.State),
			ErrorMessage: segment.ErrorMessage,
		}

		// Only add file URLs for finished segments
		if segment.State == storage.SegmentStateFinished {
			segmentCopy := segment
			segmentCopy.ID = segmentID

			segmentInfo.InlineFiles = &sspb.SegmentInfo_Files{
				Ogg:  getSegmentFileURL(ctx, h, session, &segmentCopy, storage.SEGMENT_FILENAME_OGG, false),
				Flac: getSegmentFileURL(ctx, h, session, &segmentCopy, storage.SEGMENT_FILENAME_FLAC, false),
			}
			segmentInfo.DownloadFiles = &sspb.SegmentInfo_Files{
				Ogg:  getSegmentFileURL(ctx, h, session, &segmentCopy, storage.SEGMENT_FILENAME_OGG, true),
				Flac: getSegmentFileURL(ctx, h, session, &segmentCopy, storage.SEGMENT_FILENAME_FLAC, true),
			}
		}

		info.Segments = append(info.Segments, &sspb.Segment{
			SegmentID: segmentID.String(),
			Info: &sspb.Segment_Updated{
				Updated: segmentInfo,
			},
		})
	}

	return info
}

// broadcastSessionUpdate sends a session update to all stream subscribers for the given recorder.
func (h *SessionSourceHandler) broadcastSessionUpdate(ctx context.Context, recorderID uuid.UUID, session *storage.Session) {
	h.sessionBroadcaster.Broadcast(broadcast.SessionUpdate{
		RecorderID: recorderID,
		Session: &sspb.Session{
			ID: session.ID.String(),
			Info: &sspb.Session_Updated{
				Updated: newSessionInfo(ctx, h, session),
			},
		},
	})
}

// OnSessionStateChanged implements storage.EventListener.
func (h *SessionSourceHandler) OnSessionStateChanged(event storage.SessionStateChangedEvent) {
	session := event.Session
	h.broadcastSessionUpdate(context.Background(), event.RecorderID, &session)

	// Clean up peak data when session leaves RECORDING state
	if event.PreviousState == storage.SessionStateRecording && event.NewState != storage.SessionStateRecording {
		h.peakAccumulator.RemoveSession(event.SessionID.String())
	}
}

// OnSegmentStateChanged implements storage.EventListener.
func (h *SessionSourceHandler) OnSegmentStateChanged(event storage.SegmentStateChangedEvent) {
	session := event.Session
	h.broadcastSessionUpdate(context.Background(), event.RecorderID, &session)
}


// onAudioChunk is called when audio samples are received. Broadcasts to audio and peak subscribers.
func (h *SessionSourceHandler) onAudioChunk(recorderID, sessionID uuid.UUID, samples []int16, chunkNumber int, timestamp time.Time) {
	// Convert int16 samples to int32 for proto (proto doesn't have int16)
	int32Samples := make([]int32, len(samples))
	for i, s := range samples {
		int32Samples[i] = int32(s)
	}

	h.audioBroadcaster.Broadcast(&sspb.AudioChunk{
		SessionID:   sessionID.String(),
		Samples:     int32Samples,
		ChunkNumber: uint32(chunkNumber),
		Timestamp:   timestamppb.New(timestamp),
	})

	// Compute and broadcast waveform peaks
	sid := sessionID.String()
	result := h.peakAccumulator.AddSamples(sid, samples)
	if len(result.Peaks) > 0 {
		peakValues := make([]int32, len(result.Peaks)*2)
		for i, p := range result.Peaks {
			peakValues[i*2] = int32(p.Min)
			peakValues[i*2+1] = int32(p.Max)
		}
		h.peakBroadcaster.Broadcast(&sspb.WaveformPeakData{
			SessionID:      sid,
			Peaks:          peakValues,
			IsInitial:      false,
			TotalPeakPairs: uint32(h.peakAccumulator.GetAccumulatedCount(sid)),
			PeakLevel:      result.PeakLevel,
			Clipping:       result.Clipping,
		})
	}
}

func (h *SessionSourceHandler) streamSessionAudio(ctx context.Context, request *sspb.StreamSessionAudioRequest, server sspb.SessionSource_StreamSessionAudioServer) error {
	log.Debug().Str("session-id", request.SessionID).Msg("Streaming session audio")

	requestedSessionID := request.SessionID

	// Subscribe to audio updates
	updateCh, unsubscribe := h.audioBroadcaster.Subscribe()
	defer unsubscribe()

	for {
		select {
		case chunk, ok := <-updateCh:
			if !ok {
				// Channel closed, subscriber was removed
				return nil
			}
			// Filter by session ID
			if chunk.SessionID == requestedSessionID {
				if err := server.Send(chunk); err != nil {
					log.Err(err).Msg("Cannot send audio chunk")
					return err
				}
			}
		case <-ctx.Done():
			log.Debug().Str("session-id", request.SessionID).Msg("Done streaming session audio")
			return nil
		}
	}
}

func (h *SessionSourceHandler) streamWaveformPeaks(ctx context.Context, request *sspb.StreamWaveformPeaksRequest, server sspb.SessionSource_StreamWaveformPeaksServer) error {
	log.Debug().Str("session-id", request.SessionID).Msg("Streaming waveform peaks")

	requestedSessionID := request.SessionID

	// Send accumulated peaks as initial backfill
	accumulated := h.peakAccumulator.GetAccumulated(requestedSessionID)
	if len(accumulated) > 0 {
		peakValues := make([]int32, len(accumulated)*2)
		for i, p := range accumulated {
			peakValues[i*2] = int32(p.Min)
			peakValues[i*2+1] = int32(p.Max)
		}
		if err := server.Send(&sspb.WaveformPeakData{
			SessionID:      requestedSessionID,
			Peaks:          peakValues,
			IsInitial:      true,
			TotalPeakPairs: uint32(len(accumulated)),
		}); err != nil {
			return err
		}
	}

	// Subscribe for incremental updates
	updateCh, unsubscribe := h.peakBroadcaster.Subscribe()
	defer unsubscribe()

	for {
		select {
		case data, ok := <-updateCh:
			if !ok {
				return nil
			}
			// Filter by session ID
			if data.SessionID == requestedSessionID {
				if err := server.Send(data); err != nil {
					log.Err(err).Msg("Cannot send peak data")
					return err
				}
			}
		case <-ctx.Done():
			log.Debug().Str("session-id", request.SessionID).Msg("Done streaming waveform peaks")
			return nil
		}
	}
}

func (h *SessionSourceHandler) streamRecorders(ctx context.Context, request *sspb.StreamRecordersRequest, server sspb.SessionSource_StreamRecordersServer) error {
	// Subscribe to recorder updates
	updateCh, unsubscribe := h.recorderBroadcaster.Subscribe()
	defer unsubscribe()

	recorders := h.sessionStorage.GetRecorders()

	for _, recorder := range recorders {
		// Try to get cached status from broadcaster first
		cachedStatus := h.recorderBroadcaster.GetCachedStatus(recorder.ID.String())

		var recorderMsg *sspb.Recorder
		if cachedStatus != nil {
			// Use cached status (has real signal/RMS data)
			recorderMsg = cachedStatus
		} else {
			// No cached status, send with UNKNOWN
			recorderMsg = &sspb.Recorder{
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
			}
		}

		if err := server.SendMsg(recorderMsg); err != nil {
			log.Err(err).Msg("Cannot send recorder data")
		}
	}

	for {
		select {
		case recorder, ok := <-updateCh:
			if !ok {
				// Channel closed, subscriber was removed
				return nil
			}
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

	// Subscribe to session updates
	updateCh, unsubscribe := h.sessionBroadcaster.Subscribe()
	defer unsubscribe()

	sessions := h.sessionStorage.GetSessions(recorderID)

	// Stream all sessions regardless of state (RECORDING, PROCESSING, FINISHED, ERROR)
	for _, session := range sessions {
		if err := server.SendMsg(
			&sspb.Session{
				ID: session.ID.String(),
				Info: &sspb.Session_Updated{
					Updated: newSessionInfo(ctx, h, &session),
				},
			},
		); err != nil {
			log.Err(err).Msg("Cannot send session data")
		}
	}

	for {
		select {
		case update, ok := <-updateCh:
			if !ok {
				// Channel closed, subscriber was removed
				return nil
			}
			// Only forward updates for the requested recorder
			if update.RecorderID != recorderID {
				continue
			}
			if err := server.SendMsg(update.Session); err != nil {
				log.Err(err).Msg("Cannot send session data")
			}
		case <-ctx.Done():
			log.Debug().Msg("Done streaming sessions")
			return nil
		}
	}
}

func (h *SessionSourceHandler) cutSession(ctx context.Context, request *sspb.CutSessionRequest) (*cmpb.Respone, error) {
	log.Debug().Str("recorder-id", request.RecorderID).Msg("Cutting session")

	recorderID, err := uuid.Parse(request.RecorderID)
	if err != nil {
		log.Err(err).Str("recorder-id", request.RecorderID).Msg("Cannot parse recorder ID")

		return nil, err
	}

	if !h.chunkSinkServer.IsRecorderConnected(recorderID) {
		msg := fmt.Sprintf("No connection to recorder %s", recorderID)
		log.Warn().Str("recorder-id", request.RecorderID).Msg(msg)
		return &cmpb.Respone{Success: false, ErrorMessage: msg}, fmt.Errorf("%s", msg)
	}

	if err := h.chunkSinkServer.CutSession(recorderID); err != nil {
		log.Err(err).Str("recorder-id", request.RecorderID).Msg("Cannot cut session")
		return &cmpb.Respone{Success: false, ErrorMessage: err.Error()}, err
	}

	return success, nil
}

func (h *SessionSourceHandler) retryRenderSession(ctx context.Context, request *sspb.RetryRenderSessionRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, err := parseIDs(request.RecorderID, request.SessionID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	log.Info().
		Str("recorder-id", request.RecorderID).
		Str("session-id", request.SessionID).
		Msg("Retrying session render")

	if err := h.sessionStorage.RetryRenderSession(ctx, recorderID, sessionID); err != nil {
		log.Err(err).
			Str("session-id", request.SessionID).
			Msg("Cannot retry render session")
		return &cmpb.Respone{Success: false, ErrorMessage: err.Error()}, err
	}

	return success, nil
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

func parseSegmentIDs(recorderID, sessionID, segmentID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	recorderIDParsed, sessionIDParsed, err := parseIDs(recorderID, sessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	segmentIDParsed, err := uuid.Parse(segmentID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	return recorderIDParsed, sessionIDParsed, segmentIDParsed, nil
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

	h.sessionBroadcaster.Broadcast(broadcast.SessionUpdate{
		RecorderID: recorderID,
		Session: &sspb.Session{
			ID:   sessionID.String(),
			Info: &sspb.Session_Removed{Removed: &sspb.SessionRemoved{}},
		},
	})

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

	h.broadcastSessionUpdate(ctx, recorderID, &session)

	log.Info().Str("session-id", request.SessionID).Bool("keep", request.Keep).Msg("Set keep session")

	return success, nil
}

func (h *SessionSourceHandler) setName(ctx context.Context, request *sspb.SetNameRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, err := parseIDs(request.RecorderID, request.SessionID)
	if err != nil {
		log.Err(err).Str("recorder-id", request.RecorderID).Str("session-id", request.SessionID).Msg("Cannot parse IDs")

		return noSuccess, err
	}

	if err := h.sessionStorage.SetName(ctx, recorderID, sessionID, request.Name); err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot set session name")

		return noSuccess, err
	}

	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")

		return noSuccess, err
	}

	h.broadcastSessionUpdate(ctx, recorderID, &session)

	log.Info().Str("session-id", request.SessionID).Str("name", request.Name).Msg("Set session name")

	return success, nil
}

func (h *SessionSourceHandler) createSegment(ctx context.Context, request *sspb.CreateSegmentRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, segmentID, err := parseSegmentIDs(request.RecorderID, request.SessionID, request.SegmentID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Str("segment-id", request.SegmentID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	info := request.GetInfo()
	if info == nil {
		return noSuccess, fmt.Errorf("segment info is required")
	}

	// Frontend sends offset times (seconds into session) as small Unix timestamps.
	// Convert to sample positions: nanoseconds * 48000 / 1e9 = seconds * 48000
	segment := storage.Segment{
		ID:         segmentID,
		Comment:    info.Name,
		StartPoint: info.TimeStart.AsTime().UnixNano() * 48000 / 1e9,
		EndPoint:   info.TimeEnd.AsTime().UnixNano() * 48000 / 1e9,
		State:      storage.SegmentStateUnknown,
	}

	if err := h.sessionStorage.CreateSegment(ctx, recorderID, sessionID, segmentID, segment); err != nil {
		log.Err(err).Str("segment-id", request.SegmentID).Msg("Cannot create segment")
		return noSuccess, err
	}

	// Broadcast session update
	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")
		return noSuccess, err
	}

	h.broadcastSessionUpdate(ctx, recorderID, &session)

	log.Info().
		Str("segment-id", request.SegmentID).
		Str("session-id", request.SessionID).
		Msg("Created segment")

	return success, nil
}

func (h *SessionSourceHandler) updateSegment(ctx context.Context, request *sspb.UpdateSegmentRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, segmentID, err := parseSegmentIDs(request.RecorderID, request.SessionID, request.SegmentID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Str("segment-id", request.SegmentID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	info := request.GetInfo()
	if info == nil {
		return noSuccess, fmt.Errorf("segment info is required")
	}

	// Frontend sends offset times (seconds into session) as small Unix timestamps.
	// Convert to sample positions: nanoseconds * 48000 / 1e9 = seconds * 48000
	segment := storage.Segment{
		ID:         segmentID,
		Comment:    info.Name,
		StartPoint: info.TimeStart.AsTime().UnixNano() * 48000 / 1e9,
		EndPoint:   info.TimeEnd.AsTime().UnixNano() * 48000 / 1e9,
	}

	if err := h.sessionStorage.UpdateSegment(ctx, recorderID, sessionID, segmentID, segment); err != nil {
		log.Err(err).Str("segment-id", request.SegmentID).Msg("Cannot update segment")
		return noSuccess, err
	}

	// Broadcast session update
	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")
		return noSuccess, err
	}

	h.broadcastSessionUpdate(ctx, recorderID, &session)

	log.Info().
		Str("segment-id", request.SegmentID).
		Str("session-id", request.SessionID).
		Msg("Updated segment")

	return success, nil
}

func (h *SessionSourceHandler) deleteSegment(ctx context.Context, request *sspb.DeleteSegmentRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, segmentID, err := parseSegmentIDs(request.RecorderID, request.SessionID, request.SegmentID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Str("segment-id", request.SegmentID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	if err := h.sessionStorage.DeleteSegment(ctx, recorderID, sessionID, segmentID); err != nil {
		log.Err(err).Str("segment-id", request.SegmentID).Msg("Cannot delete segment")
		return noSuccess, err
	}

	// Broadcast session update
	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")
		return noSuccess, err
	}

	h.broadcastSessionUpdate(ctx, recorderID, &session)

	log.Info().
		Str("segment-id", request.SegmentID).
		Str("session-id", request.SessionID).
		Msg("Deleted segment")

	return success, nil
}

func (h *SessionSourceHandler) renderSegment(ctx context.Context, request *sspb.RenderSegmentRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, segmentID, err := parseSegmentIDs(request.RecorderID, request.SessionID, request.SegmentID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Str("segment-id", request.SegmentID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	// Set segment state to QUEUED immediately
	if err := h.sessionStorage.SetSegmentState(ctx, recorderID, sessionID, segmentID, storage.SegmentStateQueued); err != nil {
		log.Err(err).
			Str("segment-id", segmentID.String()).
			Msg("Cannot set segment state to queued")
		return noSuccess, err
	}

	// Broadcast queued state to UI
	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", sessionID.String()).Msg("Cannot get session after queuing")
		return noSuccess, err
	}

	h.broadcastSessionUpdate(ctx, recorderID, &session)

	log.Info().
		Str("segment-id", request.SegmentID).
		Str("session-id", request.SessionID).
		Msg("Segment queued for rendering")

	// Enqueue render via the storage work queue (bounded concurrency, non-blocking).
	// The work queue worker handles QUEUED -> RENDERING -> FINISHED/ERROR transitions
	// and state persistence. UI broadcasts happen via the session state change callback.
	if err := h.sessionStorage.RenderSegment(ctx, recorderID, sessionID, segmentID); err != nil {
		log.Err(err).
			Str("segment-id", segmentID.String()).
			Msg("Cannot enqueue segment render")
		return noSuccess, err
	}

	return success, nil
}

func (h *SessionSourceHandler) shareSession(ctx context.Context, request *sspb.ShareSessionRequest) (*cmpb.Respone, error) {
	log.Debug().
		Str("recorder-id", request.RecorderID).
		Str("session-id", request.SessionID).
		Int("recipient-count", len(request.RecipientEmails)).
		Strs("recipients", request.RecipientEmails).
		Msg("Share session request received")

	recorderID, sessionID, err := parseIDs(request.RecorderID, request.SessionID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")
		return noSuccess, err
	}

	// Only allow sharing finished sessions
	if session.State != storage.SessionStateFinished {
		log.Warn().
			Str("session-id", request.SessionID).
			Str("state", session.State.String()).
			Msg("Cannot share session that is not finished")
		return noSuccess, fmt.Errorf("session is not finished")
	}

	// Check if email sender is configured
	if h.emailSender == nil {
		log.Error().Msg("Email sender is not configured")
		return noSuccess, fmt.Errorf("email sharing is not configured")
	}

	// Get a friendly session name
	sessionName := session.Name
	if sessionName == "" {
		sessionName = "Untitled Recording"
	}

	// Generate download filename
	urlFriendlyName := strings.ReplaceAll(sessionName, " ", "_")
	urlFriendlyName = urlUnsafeChars.ReplaceAllString(urlFriendlyName, "")
	dateStr := session.StartTime.Format("2006-01-02_15-04-05")
	downloadFilename := urlFriendlyName + "_" + dateStr + ".flac"

	log.Debug().
		Str("session-id", request.SessionID).
		Str("download-filename", downloadFilename).
		Msg("Generating shareable download URL")

	// Generate a download URL for the FLAC file
	shareResult, err := h.fileSharer.ShareSessionFile(ctx,
		storage.AssetOptions{
			RecorderID: session.RecorderID,
			SessionID:  session.ID,
			Filename:   storage.FILENAME_FLAC,
		},
		storage.SigningOptions{
			Expires:          time.Hour * 24 * 7, // 7 days
			Download:         true,
			DownloadFilename: downloadFilename,
		})
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot generate download URL")
		return noSuccess, fmt.Errorf("cannot generate download URL: %w", err)
	}

	log.Debug().
		Str("session-id", request.SessionID).
		Str("download-url", shareResult.URL).
		Time("expires-at", shareResult.ExpiresAt).
		Msg("Shareable download URL generated")

	// Send emails to all recipients
	sentCount := 0
	for _, recipient := range request.RecipientEmails {
		// Skip empty recipients
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			log.Warn().
				Str("session-id", request.SessionID).
				Msg("Skipping empty recipient email")
			continue
		}

		err = h.emailSender.SendShareEmail(email.ShareEmailData{
			RecipientEmail: recipient,
			SessionName:    sessionName,
			DownloadURL:    shareResult.URL,
			ExpiresAt:      shareResult.ExpiresAt,
		})
		if err != nil {
			log.Err(err).
				Str("session-id", request.SessionID).
				Str("recipient", recipient).
				Msg("Cannot send share email")
			return noSuccess, fmt.Errorf("failed to send email to %s: %w", recipient, err)
		}

		sentCount++
		log.Info().
			Str("session-id", request.SessionID).
			Str("recipient", recipient).
			Msg("Session shared via email")
	}

	if sentCount == 0 {
		log.Error().
			Str("session-id", request.SessionID).
			Int("recipient-count", len(request.RecipientEmails)).
			Msg("No valid recipient emails provided")
		return noSuccess, fmt.Errorf("no valid recipient emails provided")
	}

	return success, nil
}

func (h *SessionSourceHandler) shareSegment(ctx context.Context, request *sspb.ShareSegmentRequest) (*cmpb.Respone, error) {
	recorderID, sessionID, segmentID, err := parseSegmentIDs(request.RecorderID, request.SessionID, request.SegmentID)
	if err != nil {
		log.Err(err).
			Str("recorder-id", request.RecorderID).
			Str("session-id", request.SessionID).
			Str("segment-id", request.SegmentID).
			Msg("Cannot parse IDs")
		return noSuccess, err
	}

	session, err := h.sessionStorage.GetSession(recorderID, sessionID)
	if err != nil {
		log.Err(err).Str("session-id", request.SessionID).Msg("Cannot get session")
		return noSuccess, err
	}

	segment, ok := session.Segments[segmentID]
	if !ok {
		log.Err(err).Str("segment-id", request.SegmentID).Msg("Segment not found")
		return noSuccess, fmt.Errorf("segment not found")
	}

	// Only allow sharing rendered segments
	if segment.State != storage.SegmentStateFinished {
		log.Warn().
			Str("segment-id", request.SegmentID).
			Str("state", segment.State.String()).
			Msg("Cannot share segment that is not rendered")
		return noSuccess, fmt.Errorf("segment is not rendered")
	}

	// Check if email sender is configured
	if h.emailSender == nil {
		log.Error().Msg("Email sender is not configured")
		return noSuccess, fmt.Errorf("email sharing is not configured")
	}

	// Get a friendly segment name
	segmentName := segment.Comment
	if segmentName == "" {
		segmentName = "Segment"
	}

	// Generate download filename
	urlFriendlyName := strings.ReplaceAll(segmentName, " ", "_")
	urlFriendlyName = urlUnsafeChars.ReplaceAllString(urlFriendlyName, "")
	dateStr := session.StartTime.Format("2006-01-02_15-04-05")
	downloadFilename := urlFriendlyName + "_" + dateStr + ".flac"

	// Generate a download URL for the segment FLAC file
	shareResult, err := h.fileSharer.ShareSegmentFile(ctx,
		storage.SegmentAssetOptions{
			RecorderID: session.RecorderID,
			SessionID:  session.ID,
			SegmentID:  segmentID,
			Filename:   storage.SEGMENT_FILENAME_FLAC,
		},
		storage.SigningOptions{
			Expires:          time.Hour * 24 * 7, // 7 days
			Download:         true,
			DownloadFilename: downloadFilename,
		})
	if err != nil {
		log.Err(err).Str("segment-id", request.SegmentID).Msg("Cannot generate download URL")
		return noSuccess, fmt.Errorf("cannot generate download URL: %w", err)
	}

	// Send emails to all recipients
	sentCount := 0
	for _, recipient := range request.RecipientEmails {
		// Skip empty recipients
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			log.Warn().
				Str("segment-id", request.SegmentID).
				Msg("Skipping empty recipient email")
			continue
		}

		err = h.emailSender.SendShareEmail(email.ShareEmailData{
			RecipientEmail: recipient,
			SessionName:    segmentName,
			DownloadURL:    shareResult.URL,
			ExpiresAt:      shareResult.ExpiresAt,
		})
		if err != nil {
			log.Err(err).
				Str("segment-id", request.SegmentID).
				Str("recipient", recipient).
				Msg("Cannot send share email")
			return noSuccess, fmt.Errorf("failed to send email to %s: %w", recipient, err)
		}

		sentCount++
		log.Info().
			Str("segment-id", request.SegmentID).
			Str("recipient", recipient).
			Msg("Segment shared via email")
	}

	if sentCount == 0 {
		log.Error().
			Str("segment-id", request.SegmentID).
			Int("recipient-count", len(request.RecipientEmails)).
			Msg("No valid recipient emails provided")
		return noSuccess, fmt.Errorf("no valid recipient emails provided")
	}

	return success, nil
}
