package broadcast

import (
	"sync"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
)

// PeakBroadcaster is a specialized broadcaster for waveform peak streaming.
// Subscribers receive peak data filtered by session ID.
type PeakBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan *sspb.WaveformPeakData]struct{}
	bufferSize  int
}

// NewPeakBroadcaster creates a new PeakBroadcaster.
func NewPeakBroadcaster(bufferSize int) *PeakBroadcaster {
	return &PeakBroadcaster{
		subscribers: make(map[chan *sspb.WaveformPeakData]struct{}),
		bufferSize:  bufferSize,
	}
}

// Subscribe creates a new subscription and returns the channel and unsubscribe function.
func (b *PeakBroadcaster) Subscribe() (ch <-chan *sspb.WaveformPeakData, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan *sspb.WaveformPeakData, b.bufferSize)
	b.subscribers[subCh] = struct{}{}

	log.Debug().Int("subscribers", len(b.subscribers)).Msg("New peak subscriber added")

	return subCh, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[subCh]; ok {
			delete(b.subscribers, subCh)
			close(subCh)
			log.Debug().Int("subscribers", len(b.subscribers)).Msg("Peak subscriber removed")
		}
	}
}

// Broadcast sends waveform peak data to all subscribers.
func (b *PeakBroadcaster) Broadcast(data *sspb.WaveformPeakData) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- data:
		default:
			log.Warn().Str("session-id", data.SessionID).Msg("Subscriber buffer full, dropping peak data")
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *PeakBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
