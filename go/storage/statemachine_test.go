package storage

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestSessionStateMachine_ValidTransitions(t *testing.T) {
	tests := []struct {
		name         string
		initial      SessionState
		trigger      sessionTrigger
		wantState    SessionState
		wantCallback bool
	}{
		{"UNKNOWN -> RECORDING", SessionStateUnknown, triggerStartRecording, SessionStateRecording, true},
		{"RECORDING -> PROCESSING", SessionStateRecording, triggerCloseRecording, SessionStateProcessing, true},
		{"PROCESSING -> FINISHED", SessionStateProcessing, triggerRenderSuccess, SessionStateFinished, true},
		{"PROCESSING -> ERROR", SessionStateProcessing, triggerRenderFailure, SessionStateError, true},
		{"ERROR -> PROCESSING", SessionStateError, triggerRetryRender, SessionStateProcessing, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callbackCalled bool
			var cbSource, cbDest SessionState

			recorderID := uuid.New()
			sessionID := uuid.New()
			sm := newSessionStateMachine(recorderID, sessionID, tt.initial, func(ctx context.Context, rID, sID uuid.UUID, trigger sessionTrigger, source, destination SessionState) {
				callbackCalled = true
				cbSource = source
				cbDest = destination
				if rID != recorderID {
					t.Errorf("callback recorderID = %v, want %v", rID, recorderID)
				}
				if sID != sessionID {
					t.Errorf("callback sessionID = %v, want %v", sID, sessionID)
				}
			})

			err := sm.FireCtx(context.Background(), tt.trigger)
			if err != nil {
				t.Fatalf("FireCtx(%s) returned error: %v", tt.trigger, err)
			}

			state, _ := sm.State(context.Background())
			if state != tt.wantState {
				t.Errorf("state after trigger = %v, want %v", state, tt.wantState)
			}

			if callbackCalled != tt.wantCallback {
				t.Errorf("callback called = %v, want %v", callbackCalled, tt.wantCallback)
			}

			if tt.wantCallback {
				if cbSource != tt.initial {
					t.Errorf("callback source = %v, want %v", cbSource, tt.initial)
				}
				if cbDest != tt.wantState {
					t.Errorf("callback destination = %v, want %v", cbDest, tt.wantState)
				}
			}
		})
	}
}

func TestSessionStateMachine_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		initial SessionState
		trigger sessionTrigger
	}{
		{"UNKNOWN cannot close", SessionStateUnknown, triggerCloseRecording},
		{"UNKNOWN cannot succeed", SessionStateUnknown, triggerRenderSuccess},
		{"RECORDING cannot succeed", SessionStateRecording, triggerRenderSuccess},
		{"RECORDING cannot fail", SessionStateRecording, triggerRenderFailure},
		{"RECORDING cannot retry", SessionStateRecording, triggerRetryRender},
		{"PROCESSING cannot start", SessionStateProcessing, triggerStartRecording},
		{"PROCESSING cannot close", SessionStateProcessing, triggerCloseRecording},
		{"PROCESSING cannot retry", SessionStateProcessing, triggerRetryRender},
		{"ERROR cannot start", SessionStateError, triggerStartRecording},
		{"ERROR cannot close", SessionStateError, triggerCloseRecording},
		{"ERROR cannot succeed", SessionStateError, triggerRenderSuccess},
		{"ERROR cannot fail", SessionStateError, triggerRenderFailure},
		{"FINISHED cannot start", SessionStateFinished, triggerStartRecording},
		{"FINISHED cannot close", SessionStateFinished, triggerCloseRecording},
		{"FINISHED cannot succeed", SessionStateFinished, triggerRenderSuccess},
		{"FINISHED cannot fail", SessionStateFinished, triggerRenderFailure},
		{"FINISHED cannot retry", SessionStateFinished, triggerRetryRender},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newSessionStateMachine(uuid.New(), uuid.New(), tt.initial, nil)

			err := sm.FireCtx(context.Background(), tt.trigger)
			if err == nil {
				t.Errorf("FireCtx(%s) from %s should have returned error", tt.trigger, tt.initial)
			}

			// State should remain unchanged
			state, _ := sm.State(context.Background())
			if state != tt.initial {
				t.Errorf("state changed to %v after invalid trigger, want %v", state, tt.initial)
			}
		})
	}
}

func TestSessionStateMachine_FullLifecycle(t *testing.T) {
	var transitions []string
	sm := newSessionStateMachine(uuid.New(), uuid.New(), SessionStateUnknown,
		func(ctx context.Context, rID, sID uuid.UUID, trigger sessionTrigger, source, destination SessionState) {
			transitions = append(transitions, string(trigger))
		})

	ctx := context.Background()

	// Full happy path
	for _, trigger := range []sessionTrigger{
		triggerStartRecording,
		triggerCloseRecording,
		triggerRenderSuccess,
	} {
		if err := sm.FireCtx(ctx, trigger); err != nil {
			t.Fatalf("FireCtx(%s) failed: %v", trigger, err)
		}
	}

	state, _ := sm.State(ctx)
	if state != SessionStateFinished {
		t.Errorf("final state = %v, want FINISHED", state)
	}

	if len(transitions) != 3 {
		t.Errorf("got %d transitions, want 3", len(transitions))
	}
}

func TestSessionStateMachine_ErrorAndRetryLifecycle(t *testing.T) {
	sm := newSessionStateMachine(uuid.New(), uuid.New(), SessionStateUnknown, nil)
	ctx := context.Background()

	// Record -> Process -> Fail -> Retry -> Succeed
	triggers := []sessionTrigger{
		triggerStartRecording,
		triggerCloseRecording,
		triggerRenderFailure,
		triggerRetryRender,
		triggerRenderSuccess,
	}
	for _, trigger := range triggers {
		if err := sm.FireCtx(ctx, trigger); err != nil {
			t.Fatalf("FireCtx(%s) failed: %v", trigger, err)
		}
	}

	state, _ := sm.State(ctx)
	if state != SessionStateFinished {
		t.Errorf("final state = %v, want FINISHED", state)
	}
}

func TestSessionStateMachine_ConcurrentFire(t *testing.T) {
	// The stateless library serializes Fire calls. Verify no panics/corruption.
	var callCount atomic.Int32
	sm := newSessionStateMachine(uuid.New(), uuid.New(), SessionStateRecording,
		func(ctx context.Context, rID, sID uuid.UUID, trigger sessionTrigger, source, destination SessionState) {
			callCount.Add(1)
		})

	ctx := context.Background()
	var wg sync.WaitGroup
	successCount := atomic.Int32{}

	// 10 goroutines all try to close the same session
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := sm.FireCtx(ctx, triggerCloseRecording); err == nil {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Final state must be PROCESSING, exactly 1 callback should have fired
	state, _ := sm.State(ctx)
	if state != SessionStateProcessing {
		t.Errorf("state = %v, want PROCESSING", state)
	}
	if callCount.Load() != 1 {
		t.Errorf("callback called %d times, want 1", callCount.Load())
	}
	// At least 1 must succeed (may be more if queued mode retries from same state)
	if successCount.Load() < 1 {
		t.Error("expected at least 1 success")
	}
}

// Segment transition validation tests

func TestValidateSegmentTransition_Valid(t *testing.T) {
	tests := []struct {
		from, to SegmentState
	}{
		{SegmentStateUnknown, SegmentStateQueued},
		{SegmentStateQueued, SegmentStateRendering},
		{SegmentStateRendering, SegmentStateFinished},
		{SegmentStateRendering, SegmentStateError},
		{SegmentStateFinished, SegmentStateQueued},
		{SegmentStateError, SegmentStateQueued},
	}

	for _, tt := range tests {
		t.Run(tt.from.String()+"->"+tt.to.String(), func(t *testing.T) {
			if err := validateSegmentTransition(tt.from, tt.to); err != nil {
				t.Errorf("expected valid transition %s -> %s, got error: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestValidateSegmentTransition_Invalid(t *testing.T) {
	tests := []struct {
		from, to SegmentState
	}{
		{SegmentStateUnknown, SegmentStateRendering},
		{SegmentStateUnknown, SegmentStateFinished},
		{SegmentStateUnknown, SegmentStateError},
		{SegmentStateQueued, SegmentStateFinished},
		{SegmentStateQueued, SegmentStateError},
		{SegmentStateRendering, SegmentStateQueued},
		{SegmentStateFinished, SegmentStateRendering},
		{SegmentStateFinished, SegmentStateError},
		{SegmentStateError, SegmentStateRendering},
		{SegmentStateError, SegmentStateFinished},
	}

	for _, tt := range tests {
		t.Run(tt.from.String()+"->"+tt.to.String(), func(t *testing.T) {
			if err := validateSegmentTransition(tt.from, tt.to); err == nil {
				t.Errorf("expected invalid transition %s -> %s to return error", tt.from, tt.to)
			}
		})
	}
}
