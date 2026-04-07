package broadcast

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// Broadcaster fans out messages to multiple subscribers.
// It's safe for concurrent use.
type Broadcaster[T any] struct {
	mu          sync.RWMutex
	subscribers map[chan T]struct{}
	bufferSize  int
	lastValue   *T // Cache last value for new subscribers
}

// New creates a new Broadcaster with the specified buffer size for subscriber channels.
// A larger buffer reduces the chance of dropped messages for slow consumers.
func New[T any](bufferSize int) *Broadcaster[T] {
	return &Broadcaster[T]{
		subscribers: make(map[chan T]struct{}),
		bufferSize:  bufferSize,
	}
}

// Subscribe creates a new subscription channel and returns it along with an unsubscribe function.
// The returned channel will receive all broadcasted messages.
// Call the unsubscribe function when done to prevent resource leaks.
func (b *Broadcaster[T]) Subscribe() (ch <-chan T, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan T, b.bufferSize)
	b.subscribers[subCh] = struct{}{}

	log.Debug().Int("subscribers", len(b.subscribers)).Msg("New subscriber added")

	return subCh, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[subCh]; ok {
			delete(b.subscribers, subCh)
			close(subCh)
			log.Debug().Int("subscribers", len(b.subscribers)).Msg("Subscriber removed")
		}
	}
}

// Broadcast sends a message to all subscribers.
// Messages are sent non-blocking - if a subscriber's buffer is full, the message is dropped for that subscriber.
func (b *Broadcaster[T]) Broadcast(msg T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Cache the last value
	b.lastValue = &msg

	for ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			log.Warn().Msg("Subscriber buffer full, dropping message")
		}
	}
}

// GetLastValue returns the last broadcasted value, or nil if none.
func (b *Broadcaster[T]) GetLastValue() *T {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastValue
}

// SubscriberCount returns the current number of subscribers.
func (b *Broadcaster[T]) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Close closes all subscriber channels and clears the subscriber map.
// After Close, subscribers reading from their channels will receive the
// zero value and ok=false, unblocking any goroutines waiting on them.
func (b *Broadcaster[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}
