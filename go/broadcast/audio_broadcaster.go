package broadcast

import (
	"sync"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
)

// AudioBroadcaster is a specialized broadcaster for audio chunk streaming.
// Subscribers receive audio chunks filtered by session ID.
type AudioBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan *sspb.AudioChunk]struct{}
	bufferSize  int
}

// NewAudioBroadcaster creates a new AudioBroadcaster.
func NewAudioBroadcaster(bufferSize int) *AudioBroadcaster {
	return &AudioBroadcaster{
		subscribers: make(map[chan *sspb.AudioChunk]struct{}),
		bufferSize:  bufferSize,
	}
}

// Subscribe creates a new subscription and returns the channel and unsubscribe function.
func (b *AudioBroadcaster) Subscribe() (ch <-chan *sspb.AudioChunk, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan *sspb.AudioChunk, b.bufferSize)
	b.subscribers[subCh] = struct{}{}

	log.Debug().Int("subscribers", len(b.subscribers)).Msg("New audio subscriber added")

	return subCh, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[subCh]; ok {
			delete(b.subscribers, subCh)
			close(subCh)
			log.Debug().Int("subscribers", len(b.subscribers)).Msg("Audio subscriber removed")
		}
	}
}

// Broadcast sends an audio chunk to all subscribers.
func (b *AudioBroadcaster) Broadcast(chunk *sspb.AudioChunk) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- chunk:
		default:
			log.Warn().Str("session-id", chunk.SessionID).Msg("Subscriber buffer full, dropping audio chunk")
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *AudioBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
