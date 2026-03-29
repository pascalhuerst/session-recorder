package broadcast

import (
	"sync"

	"github.com/google/uuid"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
)

// SessionUpdate wraps a session proto message with its recorder ID
// so subscribers can filter by recorder.
type SessionUpdate struct {
	RecorderID uuid.UUID
	Session    *sspb.Session
}

// SessionBroadcaster is a specialized broadcaster for session updates.
// Each update includes the recorder ID so subscribers can filter.
type SessionBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan SessionUpdate]struct{}
	bufferSize  int
}

// NewSessionBroadcaster creates a new SessionBroadcaster.
func NewSessionBroadcaster(bufferSize int) *SessionBroadcaster {
	return &SessionBroadcaster{
		subscribers: make(map[chan SessionUpdate]struct{}),
		bufferSize:  bufferSize,
	}
}

// Subscribe creates a new subscription and returns the channel and unsubscribe function.
func (b *SessionBroadcaster) Subscribe() (ch <-chan SessionUpdate, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan SessionUpdate, b.bufferSize)
	b.subscribers[subCh] = struct{}{}

	log.Debug().Int("subscribers", len(b.subscribers)).Msg("New session subscriber added")

	return subCh, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[subCh]; ok {
			delete(b.subscribers, subCh)
			close(subCh)
			log.Debug().Int("subscribers", len(b.subscribers)).Msg("Session subscriber removed")
		}
	}
}

// Broadcast sends a session update to all subscribers.
func (b *SessionBroadcaster) Broadcast(update SessionUpdate) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- update:
		default:
			log.Warn().Str("session-id", update.Session.ID).Msg("Subscriber buffer full, dropping session update")
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *SessionBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
