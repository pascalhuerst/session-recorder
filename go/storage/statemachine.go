package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/qmuntal/stateless"
)

// Session triggers define the events that cause state transitions.
type sessionTrigger string

const (
	triggerStartRecording sessionTrigger = "StartRecording"
	triggerCloseRecording sessionTrigger = "CloseRecording"
	triggerRenderSuccess  sessionTrigger = "RenderSuccess"
	triggerRenderFailure  sessionTrigger = "RenderFailure"
	triggerRetryRender    sessionTrigger = "RetryRender"
)

// SessionTransitionFunc is called during a state transition with full session context.
type SessionTransitionFunc func(ctx context.Context, recorderID, sessionID uuid.UUID, trigger sessionTrigger, source, destination SessionState)

// newSessionStateMachine creates a state machine for a specific session.
// The recorderID and sessionID are captured in closures so the onTransition callback
// receives them without needing to look them up.
func newSessionStateMachine(
	recorderID, sessionID uuid.UUID,
	initialState SessionState,
	onTransition SessionTransitionFunc,
) *stateless.StateMachine {
	// The state is stored in this closure variable. The stateless library reads/writes
	// it via the accessor/mutator functions, and its internal lock serializes access.
	currentState := initialState

	sm := stateless.NewStateMachineWithExternalStorage(
		func(_ context.Context) (stateless.State, error) {
			return currentState, nil
		},
		func(_ context.Context, state stateless.State) error {
			currentState = state.(SessionState)
			return nil
		},
		stateless.FiringQueued,
	)

	// Helper to wire OnEntryFrom with the captured IDs
	wireEntry := func(state SessionState, trigger sessionTrigger, source SessionState) {
		sm.Configure(state).
			OnEntryFrom(trigger, func(ctx context.Context, _ ...any) error {
				if onTransition != nil {
					onTransition(ctx, recorderID, sessionID, trigger, source, state)
				}
				return nil
			})
	}

	// Configure valid transitions
	sm.Configure(SessionStateUnknown).
		Permit(triggerStartRecording, SessionStateRecording)

	sm.Configure(SessionStateRecording).
		Permit(triggerCloseRecording, SessionStateProcessing)

	sm.Configure(SessionStateProcessing).
		Permit(triggerRenderSuccess, SessionStateFinished).
		Permit(triggerRenderFailure, SessionStateError)

	sm.Configure(SessionStateError).
		Permit(triggerRetryRender, SessionStateProcessing)

	// Terminal state — no transitions out
	sm.Configure(SessionStateFinished)

	// Wire transition callbacks
	wireEntry(SessionStateRecording, triggerStartRecording, SessionStateUnknown)
	wireEntry(SessionStateProcessing, triggerCloseRecording, SessionStateRecording)
	wireEntry(SessionStateFinished, triggerRenderSuccess, SessionStateProcessing)
	wireEntry(SessionStateError, triggerRenderFailure, SessionStateProcessing)
	wireEntry(SessionStateProcessing, triggerRetryRender, SessionStateError)

	return sm
}

// Segment transition validation

// validSegmentTransitions defines which segment state transitions are allowed.
var validSegmentTransitions = map[SegmentState]map[SegmentState]bool{
	SegmentStateUnknown: {
		SegmentStateQueued: true,
	},
	SegmentStateQueued: {
		SegmentStateRendering: true,
	},
	SegmentStateRendering: {
		SegmentStateFinished: true,
		SegmentStateError:    true,
	},
	SegmentStateFinished: {
		SegmentStateQueued: true, // re-render after update
	},
	SegmentStateError: {
		SegmentStateQueued: true, // retry
	},
}

// validateSegmentTransition checks if a segment state transition is valid.
func validateSegmentTransition(from, to SegmentState) error {
	targets, ok := validSegmentTransitions[from]
	if !ok {
		return fmt.Errorf("no transitions allowed from segment state %s", from)
	}
	if !targets[to] {
		return fmt.Errorf("invalid segment transition: %s -> %s", from, to)
	}
	return nil
}
