package storage

import "github.com/rs/zerolog/log"

// LogListener logs all lifecycle events using structured zerolog output.
type LogListener struct{}

func (l *LogListener) OnSessionStateChanged(event SessionStateChangedEvent) {
	log.Info().
		Stringer("recorder-id", event.RecorderID).
		Stringer("session-id", event.SessionID).
		Str("previous-state", event.PreviousState.String()).
		Str("new-state", event.NewState.String()).
		Str("trigger", event.Trigger).
		Str("error", event.ErrorMessage).
		Msg("Session state changed")
}

func (l *LogListener) OnSegmentStateChanged(event SegmentStateChangedEvent) {
	log.Info().
		Stringer("recorder-id", event.RecorderID).
		Stringer("session-id", event.SessionID).
		Stringer("segment-id", event.SegmentID).
		Str("previous-state", event.PreviousState.String()).
		Str("new-state", event.NewState.String()).
		Str("error", event.ErrorMessage).
		Msg("Segment state changed")
}

