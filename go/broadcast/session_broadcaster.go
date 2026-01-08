package broadcast

import (
	"sync"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
)

// SessionBroadcaster is a specialized broadcaster for session updates.
// All subscribers receive all session updates - filtering by recorder ID
// should be done at the subscription or handler level.
type SessionBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan *sspb.Session]struct{}
	bufferSize  int
}

// NewSessionBroadcaster creates a new SessionBroadcaster.
func NewSessionBroadcaster(bufferSize int) *SessionBroadcaster {
	return &SessionBroadcaster{
		subscribers: make(map[chan *sspb.Session]struct{}),
		bufferSize:  bufferSize,
	}
}

// Subscribe creates a new subscription and returns the channel and unsubscribe function.
func (b *SessionBroadcaster) Subscribe() (ch <-chan *sspb.Session, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan *sspb.Session, b.bufferSize)
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
func (b *SessionBroadcaster) Broadcast(session *sspb.Session) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- session:
		default:
			log.Warn().Str("session-id", session.ID).Msg("Subscriber buffer full, dropping session update")
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *SessionBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
