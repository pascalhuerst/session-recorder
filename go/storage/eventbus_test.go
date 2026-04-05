package storage

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

// testListener records events for assertion.
type testListener struct {
	mu              sync.Mutex
	sessionEvents   []SessionStateChangedEvent
	segmentEvents   []SegmentStateChangedEvent
}

func (l *testListener) OnSessionStateChanged(event SessionStateChangedEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sessionEvents = append(l.sessionEvents, event)
}

func (l *testListener) OnSegmentStateChanged(event SegmentStateChangedEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.segmentEvents = append(l.segmentEvents, event)
}

func (l *testListener) getSessionEvents() []SessionStateChangedEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := make([]SessionStateChangedEvent, len(l.sessionEvents))
	copy(cp, l.sessionEvents)
	return cp
}

func (l *testListener) getSegmentEvents() []SegmentStateChangedEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := make([]SegmentStateChangedEvent, len(l.segmentEvents))
	copy(cp, l.segmentEvents)
	return cp
}

func makeSessionEvent() SessionStateChangedEvent {
	return SessionStateChangedEvent{
		RecorderID:    uuid.New(),
		SessionID:     uuid.New(),
		PreviousState: SessionStateUnknown,
		NewState:      SessionStateRecording,
		Trigger:       "StartRecording",
	}
}

func makeSegmentEvent() SegmentStateChangedEvent {
	return SegmentStateChangedEvent{
		RecorderID:    uuid.New(),
		SessionID:     uuid.New(),
		SegmentID:     uuid.New(),
		PreviousState: SegmentStateQueued,
		NewState:      SegmentStateRendering,
	}
}

func TestEventBus_EmitSessionStateChanged_NoListeners(t *testing.T) {
	bus := NewEventBus()
	// Should not panic
	bus.EmitSessionStateChanged(makeSessionEvent())
}

func TestEventBus_EmitSegmentStateChanged_NoListeners(t *testing.T) {
	bus := NewEventBus()
	bus.EmitSegmentStateChanged(makeSegmentEvent())
}

func TestEventBus_EmitSessionStateChanged_SingleListener(t *testing.T) {
	bus := NewEventBus()
	listener := &testListener{}
	bus.AddListener(listener)

	event := makeSessionEvent()
	bus.EmitSessionStateChanged(event)

	events := listener.getSessionEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].RecorderID != event.RecorderID {
		t.Errorf("RecorderID mismatch: got %v, want %v", events[0].RecorderID, event.RecorderID)
	}
	if events[0].SessionID != event.SessionID {
		t.Errorf("SessionID mismatch: got %v, want %v", events[0].SessionID, event.SessionID)
	}
	if events[0].PreviousState != event.PreviousState {
		t.Errorf("PreviousState mismatch: got %v, want %v", events[0].PreviousState, event.PreviousState)
	}
	if events[0].NewState != event.NewState {
		t.Errorf("NewState mismatch: got %v, want %v", events[0].NewState, event.NewState)
	}
	if events[0].Trigger != event.Trigger {
		t.Errorf("Trigger mismatch: got %v, want %v", events[0].Trigger, event.Trigger)
	}
}

func TestEventBus_EmitSegmentStateChanged_SingleListener(t *testing.T) {
	bus := NewEventBus()
	listener := &testListener{}
	bus.AddListener(listener)

	event := makeSegmentEvent()
	bus.EmitSegmentStateChanged(event)

	events := listener.getSegmentEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].SegmentID != event.SegmentID {
		t.Errorf("SegmentID mismatch: got %v, want %v", events[0].SegmentID, event.SegmentID)
	}
	if events[0].PreviousState != event.PreviousState {
		t.Errorf("PreviousState mismatch: got %v, want %v", events[0].PreviousState, event.PreviousState)
	}
	if events[0].NewState != event.NewState {
		t.Errorf("NewState mismatch: got %v, want %v", events[0].NewState, event.NewState)
	}
}

func TestEventBus_MultipleListeners(t *testing.T) {
	bus := NewEventBus()
	l1 := &testListener{}
	l2 := &testListener{}
	l3 := &testListener{}
	bus.AddListener(l1)
	bus.AddListener(l2)
	bus.AddListener(l3)

	bus.EmitSessionStateChanged(makeSessionEvent())
	bus.EmitSegmentStateChanged(makeSegmentEvent())

	for i, l := range []*testListener{l1, l2, l3} {
		if len(l.getSessionEvents()) != 1 {
			t.Errorf("listener %d: expected 1 session event, got %d", i, len(l.getSessionEvents()))
		}
		if len(l.getSegmentEvents()) != 1 {
			t.Errorf("listener %d: expected 1 segment event, got %d", i, len(l.getSegmentEvents()))
		}
	}
}

func TestEventBus_RemoveListener(t *testing.T) {
	bus := NewEventBus()
	l1 := &testListener{}
	l2 := &testListener{}
	bus.AddListener(l1)
	bus.AddListener(l2)

	if bus.ListenerCount() != 2 {
		t.Fatalf("expected 2 listeners, got %d", bus.ListenerCount())
	}

	bus.RemoveListener(l1)

	if bus.ListenerCount() != 1 {
		t.Fatalf("expected 1 listener after removal, got %d", bus.ListenerCount())
	}

	bus.EmitSessionStateChanged(makeSessionEvent())

	if len(l1.getSessionEvents()) != 0 {
		t.Error("removed listener should not receive events")
	}
	if len(l2.getSessionEvents()) != 1 {
		t.Error("remaining listener should receive events")
	}
}

func TestEventBus_RemoveListener_NotFound(t *testing.T) {
	bus := NewEventBus()
	l := &testListener{}
	// Should not panic
	bus.RemoveListener(l)
}

func TestEventBus_ConcurrentEmit(t *testing.T) {
	bus := NewEventBus()
	var count atomic.Int32
	listener := &countingListener{count: &count}
	bus.AddListener(listener)

	const goroutines = 10
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				bus.EmitSessionStateChanged(makeSessionEvent())
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				bus.EmitSegmentStateChanged(makeSegmentEvent())
			}
		}()
	}

	wg.Wait()

	expected := int32(goroutines * eventsPerGoroutine * 2)
	if got := count.Load(); got != expected {
		t.Errorf("expected %d total events, got %d", expected, got)
	}
}

func TestEventBus_ListenerCount(t *testing.T) {
	bus := NewEventBus()
	if bus.ListenerCount() != 0 {
		t.Fatalf("expected 0, got %d", bus.ListenerCount())
	}

	l := &testListener{}
	bus.AddListener(l)
	if bus.ListenerCount() != 1 {
		t.Fatalf("expected 1, got %d", bus.ListenerCount())
	}

	bus.RemoveListener(l)
	if bus.ListenerCount() != 0 {
		t.Fatalf("expected 0, got %d", bus.ListenerCount())
	}
}

func TestEventBus_SessionEventCarriesErrorMessage(t *testing.T) {
	bus := NewEventBus()
	listener := &testListener{}
	bus.AddListener(listener)

	event := SessionStateChangedEvent{
		RecorderID:    uuid.New(),
		SessionID:     uuid.New(),
		PreviousState: SessionStateProcessing,
		NewState:      SessionStateError,
		Trigger:       "RenderFailure",
		ErrorMessage:  "encoding failed: ffmpeg exited with code 1",
	}
	bus.EmitSessionStateChanged(event)

	events := listener.getSessionEvents()
	if events[0].ErrorMessage != event.ErrorMessage {
		t.Errorf("ErrorMessage mismatch: got %q, want %q", events[0].ErrorMessage, event.ErrorMessage)
	}
}

func TestEventBus_SegmentEventCarriesErrorMessage(t *testing.T) {
	bus := NewEventBus()
	listener := &testListener{}
	bus.AddListener(listener)

	event := SegmentStateChangedEvent{
		RecorderID:    uuid.New(),
		SessionID:     uuid.New(),
		SegmentID:     uuid.New(),
		PreviousState: SegmentStateRendering,
		NewState:      SegmentStateError,
		ErrorMessage:  "invalid segment range",
	}
	bus.EmitSegmentStateChanged(event)

	events := listener.getSegmentEvents()
	if events[0].ErrorMessage != event.ErrorMessage {
		t.Errorf("ErrorMessage mismatch: got %q, want %q", events[0].ErrorMessage, event.ErrorMessage)
	}
}

// countingListener just counts events for concurrency tests.
type countingListener struct {
	count *atomic.Int32
}

func (l *countingListener) OnSessionStateChanged(_ SessionStateChangedEvent) {
	l.count.Add(1)
}

func (l *countingListener) OnSegmentStateChanged(_ SegmentStateChangedEvent) {
	l.count.Add(1)
}

