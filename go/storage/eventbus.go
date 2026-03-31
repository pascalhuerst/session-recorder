package storage

import "sync"

// EventListener receives lifecycle events from the storage layer.
// Implementations MUST be fast and non-blocking. If a listener needs to do
// slow work (e.g., network I/O), it should enqueue internally.
type EventListener interface {
	OnSessionStateChanged(event SessionStateChangedEvent)
	OnSegmentStateChanged(event SegmentStateChangedEvent)
}

// EventBus dispatches lifecycle events to registered listeners.
// All emission is synchronous — listeners are called inline under a read lock.
type EventBus struct {
	mu        sync.RWMutex
	listeners []EventListener
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// AddListener registers a listener for lifecycle events.
func (eb *EventBus) AddListener(l EventListener) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.listeners = append(eb.listeners, l)
}

// RemoveListener unregisters a listener. No-op if the listener is not found.
func (eb *EventBus) RemoveListener(l EventListener) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for i, existing := range eb.listeners {
		if existing == l {
			eb.listeners = append(eb.listeners[:i], eb.listeners[i+1:]...)
			return
		}
	}
}

// ListenerCount returns the current number of registered listeners.
func (eb *EventBus) ListenerCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.listeners)
}

// EmitSessionStateChanged dispatches a session state change event to all listeners.
func (eb *EventBus) EmitSessionStateChanged(event SessionStateChangedEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, l := range eb.listeners {
		l.OnSessionStateChanged(event)
	}
}

// EmitSegmentStateChanged dispatches a segment state change event to all listeners.
func (eb *EventBus) EmitSegmentStateChanged(event SegmentStateChangedEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, l := range eb.listeners {
		l.OnSegmentStateChanged(event)
	}
}
