package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/email"
	"github.com/pascalhuerst/session-recorder/fileshare"
	"github.com/pascalhuerst/session-recorder/grpc"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/pascalhuerst/session-recorder/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const sampleRate = 48000

type ShowHandler struct {
	sessionStorage  storage.Storage
	chunkSinkServer *grpc.ChunkSinkServer
	emailSender     *email.Sender
	fileSharer      fileshare.FileSharer
	renderSemaphore chan struct{} // Shared with SessionSourceHandler
}

func NewShowHandler(
	sessionStorage storage.Storage,
	chunkSinkServer *grpc.ChunkSinkServer,
	emailSender *email.Sender,
	fileSharer fileshare.FileSharer,
	renderSemaphore chan struct{},
) *ShowHandler {
	return &ShowHandler{
		sessionStorage:  sessionStorage,
		chunkSinkServer: chunkSinkServer,
		emailSender:     emailSender,
		fileSharer:      fileSharer,
		renderSemaphore: renderSemaphore,
	}
}

func (h *ShowHandler) createShow(ctx context.Context, request *sspb.CreateShowRequest) (*cmpb.Respone, error) {
	info := request.GetShow()
	if info == nil {
		return noSuccess, fmt.Errorf("show info is required")
	}

	showID := uuid.New()
	if info.ShowID != "" {
		parsed, err := uuid.Parse(info.ShowID)
		if err == nil {
			showID = parsed
		}
	}

	recorderID, err := uuid.Parse(info.RecorderID)
	if err != nil {
		return noSuccess, fmt.Errorf("invalid recorder ID: %w", err)
	}

	show := storage.Show{
		ID:         showID,
		Name:       info.Name,
		Date:       info.Date.AsTime(),
		State:      storage.ShowStateDraft,
		RecorderID: recorderID,
		Acts:       protoActsToStorage(info.Acts),
	}

	if err := h.sessionStorage.SaveShow(ctx, show); err != nil {
		log.Err(err).Msg("Cannot save show")
		return noSuccess, err
	}

	log.Info().Str("show-id", showID.String()).Str("name", show.Name).Msg("Created show")
	return success, nil
}

func (h *ShowHandler) updateShow(ctx context.Context, request *sspb.UpdateShowRequest) (*cmpb.Respone, error) {
	info := request.GetShow()
	if info == nil {
		return noSuccess, fmt.Errorf("show info is required")
	}

	showID, err := uuid.Parse(info.ShowID)
	if err != nil {
		return noSuccess, fmt.Errorf("invalid show ID: %w", err)
	}

	show, err := h.sessionStorage.GetShow(showID)
	if err != nil {
		return noSuccess, err
	}

	if show.State != storage.ShowStateDraft && show.State != storage.ShowStateEnded {
		return noSuccess, fmt.Errorf("can only update shows in DRAFT or ENDED state")
	}

	if info.Name != "" {
		show.Name = info.Name
	}
	if info.Date != nil {
		show.Date = info.Date.AsTime()
	}
	if info.RecorderID != "" {
		recorderID, err := uuid.Parse(info.RecorderID)
		if err != nil {
			return noSuccess, fmt.Errorf("invalid recorder ID: %w", err)
		}
		show.RecorderID = recorderID
	}
	if info.Acts != nil {
		show.Acts = protoActsToStorage(info.Acts)
	}

	if err := h.sessionStorage.SaveShow(ctx, show); err != nil {
		log.Err(err).Msg("Cannot save show")
		return noSuccess, err
	}

	log.Info().Str("show-id", showID.String()).Msg("Updated show")
	return success, nil
}

func (h *ShowHandler) deleteShow(ctx context.Context, request *sspb.DeleteShowRequest) (*cmpb.Respone, error) {
	showID, err := uuid.Parse(request.ShowID)
	if err != nil {
		return noSuccess, fmt.Errorf("invalid show ID: %w", err)
	}

	show, err := h.sessionStorage.GetShow(showID)
	if err != nil {
		return noSuccess, err
	}

	if show.State == storage.ShowStateLive {
		return noSuccess, fmt.Errorf("cannot delete a LIVE show")
	}

	if err := h.sessionStorage.DeleteShow(ctx, showID); err != nil {
		log.Err(err).Msg("Cannot delete show")
		return noSuccess, err
	}

	log.Info().Str("show-id", showID.String()).Msg("Deleted show")
	return success, nil
}

func (h *ShowHandler) listShows(_ context.Context, _ *sspb.ListShowsRequest) (*sspb.ListShowsResponse, error) {
	shows := h.sessionStorage.GetShows()

	protoShows := make([]*sspb.ShowInfo, 0, len(shows))
	for _, show := range shows {
		protoShows = append(protoShows, storageShowToProto(show))
	}

	return &sspb.ListShowsResponse{Shows: protoShows}, nil
}

func (h *ShowHandler) startShow(ctx context.Context, request *sspb.StartShowRequest) (*cmpb.Respone, error) {
	showID, err := uuid.Parse(request.ShowID)
	if err != nil {
		return noSuccess, fmt.Errorf("invalid show ID: %w", err)
	}

	show, err := h.sessionStorage.GetShow(showID)
	if err != nil {
		return noSuccess, err
	}

	if show.State != storage.ShowStateDraft {
		return noSuccess, fmt.Errorf("can only start shows in DRAFT state")
	}

	// Cut session on the recorder to start a fresh recording
	if err := h.chunkSinkServer.CutSession(show.RecorderID); err != nil {
		log.Err(err).Str("recorder-id", show.RecorderID.String()).Msg("Cannot cut session on recorder")
		return noSuccess, fmt.Errorf("cannot start recording on recorder: %w", err)
	}

	show.State = storage.ShowStateLive
	if err := h.sessionStorage.SaveShow(ctx, show); err != nil {
		log.Err(err).Msg("Cannot save show")
		return noSuccess, err
	}

	log.Info().Str("show-id", showID.String()).Str("recorder-id", show.RecorderID.String()).Msg("Show started")
	return success, nil
}

// OnSessionStateChanged is called by the session state change callback.
// It links new recording sessions to LIVE shows and transitions shows to ENDED when sessions finish.
func (h *ShowHandler) OnSessionStateChanged(session *storage.Session, previousState storage.SessionState) {
	shows := h.sessionStorage.GetShows()

	for _, show := range shows {
		if show.State != storage.ShowStateLive || show.RecorderID != session.RecorderID {
			continue
		}

		// Link new session to show
		if session.State == storage.SessionStateRecording && show.SessionID == uuid.Nil {
			show.SessionID = session.ID
			if err := h.sessionStorage.SaveShow(context.Background(), show); err != nil {
				log.Err(err).Str("show-id", show.ID.String()).Msg("Cannot link session to show")
			} else {
				log.Info().
					Str("show-id", show.ID.String()).
					Str("session-id", session.ID.String()).
					Msg("Linked session to show")
			}
		}

		// Transition to ENDED when linked session finishes
		if session.State == storage.SessionStateFinished && show.SessionID == session.ID {
			show.State = storage.ShowStateEnded

			// Auto-create segments from act boundaries
			h.createSegmentsFromActs(context.Background(), &show, session)

			if err := h.sessionStorage.SaveShow(context.Background(), show); err != nil {
				log.Err(err).Str("show-id", show.ID.String()).Msg("Cannot transition show to ENDED")
			} else {
				log.Info().
					Str("show-id", show.ID.String()).
					Str("session-id", session.ID.String()).
					Msg("Show ended, segments created from acts")
			}
		}
	}
}

func (h *ShowHandler) createSegmentsFromActs(ctx context.Context, show *storage.Show, session *storage.Session) {
	for i := range show.Acts {
		act := &show.Acts[i]

		// Use actual times if set, otherwise planned times
		start := act.PlannedStart
		end := act.PlannedEnd
		if !act.ActualStart.IsZero() {
			start = act.ActualStart
		}
		if !act.ActualEnd.IsZero() {
			end = act.ActualEnd
		}

		// Calculate sample positions relative to session start
		startOffset := start.Sub(session.StartTime)
		endOffset := end.Sub(session.StartTime)

		if startOffset < 0 {
			startOffset = 0
		}
		if endOffset <= startOffset {
			log.Warn().Str("act", act.Name).Msg("Skipping act with invalid time range")
			continue
		}

		startSample := int64(startOffset.Seconds() * sampleRate)
		endSample := int64(endOffset.Seconds() * sampleRate)

		segmentID := uuid.New()
		segment := storage.Segment{
			ID:         segmentID,
			Comment:    act.Name,
			StartPoint: startSample,
			EndPoint:   endSample,
			State:      storage.SegmentStateUnknown,
		}

		if err := h.sessionStorage.CreateSegment(ctx, show.RecorderID, session.ID, segmentID, segment); err != nil {
			log.Err(err).Str("act", act.Name).Msg("Cannot create segment for act")
			continue
		}

		act.SegmentID = segmentID
		log.Info().
			Str("act", act.Name).
			Str("segment-id", segmentID.String()).
			Msg("Created segment for act")
	}
}

func (h *ShowHandler) renderAll(ctx context.Context, request *sspb.RenderAllRequest) (*cmpb.Respone, error) {
	showID, err := uuid.Parse(request.ShowID)
	if err != nil {
		return noSuccess, fmt.Errorf("invalid show ID: %w", err)
	}

	show, err := h.sessionStorage.GetShow(showID)
	if err != nil {
		return noSuccess, err
	}

	if show.State != storage.ShowStateEnded {
		return noSuccess, fmt.Errorf("can only render shows in ENDED state")
	}

	if show.SessionID == uuid.Nil {
		return noSuccess, fmt.Errorf("show has no linked session")
	}

	// Queue rendering for all acts that have segments
	for _, act := range show.Acts {
		if act.SegmentID == uuid.Nil {
			continue
		}

		segmentID := act.SegmentID
		recorderID := show.RecorderID
		sessionID := show.SessionID

		// Set segment state to QUEUED
		if err := h.sessionStorage.SetSegmentState(ctx, recorderID, sessionID, segmentID, storage.SegmentStateQueued); err != nil {
			log.Err(err).Str("segment-id", segmentID.String()).Msg("Cannot queue segment for rendering")
			continue
		}

		// Render asynchronously with concurrency control
		go func() {
			h.renderSemaphore <- struct{}{}
			defer func() { <-h.renderSemaphore }()

			if err := h.sessionStorage.SetSegmentState(context.Background(), recorderID, sessionID, segmentID, storage.SegmentStateRendering); err != nil {
				log.Err(err).Str("segment-id", segmentID.String()).Msg("Cannot set segment state to rendering")
			}

			if err := h.sessionStorage.RenderSegment(context.Background(), recorderID, sessionID, segmentID); err != nil {
				log.Err(err).Str("segment-id", segmentID.String()).Msg("Cannot render segment")
				return
			}

			log.Info().Str("segment-id", segmentID.String()).Msg("Segment rendered for show")
		}()
	}

	log.Info().Str("show-id", showID.String()).Msg("Queued all segments for rendering")
	return success, nil
}

func (h *ShowHandler) distributeAll(ctx context.Context, request *sspb.DistributeAllRequest) (*cmpb.Respone, error) {
	showID, err := uuid.Parse(request.ShowID)
	if err != nil {
		return noSuccess, fmt.Errorf("invalid show ID: %w", err)
	}

	show, err := h.sessionStorage.GetShow(showID)
	if err != nil {
		return noSuccess, err
	}

	if show.State != storage.ShowStateEnded {
		return noSuccess, fmt.Errorf("can only distribute shows in ENDED state")
	}

	if h.emailSender == nil {
		return noSuccess, fmt.Errorf("email sharing is not configured")
	}

	session, err := h.sessionStorage.GetSession(show.RecorderID, show.SessionID)
	if err != nil {
		return noSuccess, fmt.Errorf("cannot get linked session: %w", err)
	}

	sentCount := 0
	for _, act := range show.Acts {
		if act.SegmentID == uuid.Nil || len(act.Emails) == 0 {
			continue
		}

		segment, ok := session.Segments[act.SegmentID]
		if !ok || segment.State != storage.SegmentStateFinished {
			log.Warn().Str("act", act.Name).Msg("Skipping act: segment not rendered")
			continue
		}

		// Generate download URL
		downloadFilename := fmt.Sprintf("%s_%s.flac", act.Name, session.StartTime.Format("2006-01-02"))
		shareResult, err := h.fileSharer.ShareSegmentFile(ctx,
			storage.SegmentAssetOptions{
				RecorderID: show.RecorderID,
				SessionID:  show.SessionID,
				SegmentID:  act.SegmentID,
				Filename:   storage.SEGMENT_FILENAME_FLAC,
			},
			storage.SigningOptions{
				Expires:          time.Hour * 24 * 7,
				Download:         true,
				DownloadFilename: downloadFilename,
			})
		if err != nil {
			log.Err(err).Str("act", act.Name).Msg("Cannot generate download URL")
			continue
		}

		// Send to all recipients
		for _, recipient := range act.Emails {
			if err := h.emailSender.SendShareEmail(email.ShareEmailData{
				RecipientEmail: recipient,
				SessionName:    act.Name,
				DownloadURL:    shareResult.URL,
				ExpiresAt:      shareResult.ExpiresAt,
			}); err != nil {
				log.Err(err).Str("recipient", recipient).Str("act", act.Name).Msg("Cannot send email")
				continue
			}
			sentCount++
		}
	}

	// Transition to archived
	show.State = storage.ShowStateArchived
	if err := h.sessionStorage.SaveShow(ctx, show); err != nil {
		log.Err(err).Msg("Cannot archive show")
	}

	log.Info().Str("show-id", showID.String()).Int("emails-sent", sentCount).Msg("Distributed show recordings")
	return success, nil
}

// Conversion helpers

func protoActsToStorage(acts []*sspb.ActInfo) []storage.Act {
	result := make([]storage.Act, len(acts))
	for i, a := range acts {
		actID := uuid.New()
		if a.ActID != "" {
			if parsed, err := uuid.Parse(a.ActID); err == nil {
				actID = parsed
			}
		}

		result[i] = storage.Act{
			ID:           actID,
			Name:         a.Name,
			PlannedStart: a.PlannedStart.AsTime(),
			PlannedEnd:   a.PlannedEnd.AsTime(),
			Emails:       a.Emails,
		}

		if a.SegmentID != "" {
			if parsed, err := uuid.Parse(a.SegmentID); err == nil {
				result[i].SegmentID = parsed
			}
		}
		if a.ActualStart != nil {
			result[i].ActualStart = a.ActualStart.AsTime()
		}
		if a.ActualEnd != nil {
			result[i].ActualEnd = a.ActualEnd.AsTime()
		}
	}
	return result
}

func storageShowToProto(show storage.Show) *sspb.ShowInfo {
	acts := make([]*sspb.ActInfo, len(show.Acts))
	for i, a := range show.Acts {
		act := &sspb.ActInfo{
			ActID:        a.ID.String(),
			Name:         a.Name,
			PlannedStart: timestamppb.New(a.PlannedStart),
			PlannedEnd:   timestamppb.New(a.PlannedEnd),
			Emails:       a.Emails,
		}
		if a.SegmentID != uuid.Nil {
			act.SegmentID = a.SegmentID.String()
		}
		if !a.ActualStart.IsZero() {
			act.ActualStart = timestamppb.New(a.ActualStart)
		}
		if !a.ActualEnd.IsZero() {
			act.ActualEnd = timestamppb.New(a.ActualEnd)
		}
		acts[i] = act
	}

	state := sspb.ShowState_SHOW_STATE_DRAFT
	switch show.State {
	case storage.ShowStateLive:
		state = sspb.ShowState_SHOW_STATE_LIVE
	case storage.ShowStateEnded:
		state = sspb.ShowState_SHOW_STATE_ENDED
	case storage.ShowStateArchived:
		state = sspb.ShowState_SHOW_STATE_ARCHIVED
	}

	info := &sspb.ShowInfo{
		ShowID:     show.ID.String(),
		Name:       show.Name,
		Date:       timestamppb.New(show.Date),
		State:      state,
		RecorderID: show.RecorderID.String(),
		Acts:       acts,
	}
	if show.SessionID != uuid.Nil {
		info.SessionID = show.SessionID.String()
	}

	return info
}
