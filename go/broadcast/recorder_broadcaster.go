package broadcast

import (
	"sync"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
)

// RecorderBroadcaster is a specialized broadcaster for recorder status updates.
// It caches the last known status per recorder so new clients can receive
// the current state immediately upon subscription.
type RecorderBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan *sspb.Recorder]struct{}
	bufferSize  int
	// Cache last status per recorder ID
	lastStatus map[string]*sspb.Recorder
}

// NewRecorderBroadcaster creates a new RecorderBroadcaster.
func NewRecorderBroadcaster(bufferSize int) *RecorderBroadcaster {
	return &RecorderBroadcaster{
		subscribers: make(map[chan *sspb.Recorder]struct{}),
		bufferSize:  bufferSize,
		lastStatus:  make(map[string]*sspb.Recorder),
	}
}

// Subscribe creates a new subscription and returns the channel and unsubscribe function.
func (b *RecorderBroadcaster) Subscribe() (ch <-chan *sspb.Recorder, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan *sspb.Recorder, b.bufferSize)
	b.subscribers[subCh] = struct{}{}

	log.Debug().Int("subscribers", len(b.subscribers)).Msg("New recorder subscriber added")

	return subCh, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[subCh]; ok {
			delete(b.subscribers, subCh)
			close(subCh)
			log.Debug().Int("subscribers", len(b.subscribers)).Msg("Recorder subscriber removed")
		}
	}
}

// Broadcast sends a recorder update to all subscribers and caches the status.
func (b *RecorderBroadcaster) Broadcast(recorder *sspb.Recorder) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Cache the status by recorder ID
	b.lastStatus[recorder.RecorderID] = recorder

	for ch := range b.subscribers {
		select {
		case ch <- recorder:
		default:
			log.Warn().Str("recorder-id", recorder.RecorderID).Msg("Subscriber buffer full, dropping recorder update")
		}
	}
}

// GetCachedStatus returns the last known status for a specific recorder.
func (b *RecorderBroadcaster) GetCachedStatus(recorderID string) *sspb.Recorder {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastStatus[recorderID]
}

// GetAllCachedStatuses returns all cached recorder statuses.
func (b *RecorderBroadcaster) GetAllCachedStatuses() []*sspb.Recorder {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*sspb.Recorder, 0, len(b.lastStatus))
	for _, recorder := range b.lastStatus {
		result = append(result, recorder)
	}
	return result
}

// SubscriberCount returns the current number of subscribers.
func (b *RecorderBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
