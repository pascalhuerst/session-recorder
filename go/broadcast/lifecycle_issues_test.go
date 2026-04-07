package broadcast

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// Issue: Broadcaster has no shutdown mechanism
//
// When Stop() is called on the system, broadcasters have no way to signal
// subscribers that they should disconnect. Subscriber channels remain open
// and goroutines reading from them block indefinitely.
// =============================================================================

func TestFix_BroadcasterCloseUnblocksSubscribers(t *testing.T) {
	b := New[string](5)

	ch, _ := b.Subscribe()

	// Simulate a gRPC stream handler waiting for messages
	unblocked := make(chan struct{})
	go func() {
		// This is what StreamRecorders/StreamSessions do:
		// block on channel read until closed
		_, ok := <-ch
		if !ok {
			close(unblocked)
		}
	}()

	// Close() should unblock all subscribers
	b.Close()

	select {
	case <-unblocked:
		// Good — subscriber was unblocked by Close()
	case <-time.After(time.Second):
		t.Error("Broadcaster.Close() did not unblock subscriber goroutine")
	}

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d after Close(), want 0", b.SubscriberCount())
	}
}

// =============================================================================
// Issue: Slow subscriber blocks all other subscribers
//
// Broadcaster uses a fixed buffer (10 in production). When one subscriber's
// buffer fills up, Broadcast drops the message for that subscriber (non-blocking).
// However, if broadcasts arrive faster than any subscriber processes them,
// ALL subscribers lose messages — there's no per-subscriber backpressure or
// eviction.
// =============================================================================

func TestKnown_SlowSubscriberMessageDrop(t *testing.T) {
	b := New[int](2) // Small buffer to trigger easily

	// Fast subscriber
	fastCh, fastUnsub := b.Subscribe()
	defer fastUnsub()

	// Slow subscriber — never reads
	_, slowUnsub := b.Subscribe()
	defer slowUnsub()

	// Broadcast more messages than buffer can hold
	for i := 0; i < 10; i++ {
		b.Broadcast(i)
	}

	// Fast subscriber only gets first 2 (buffer size), rest dropped because
	// the broadcast loop uses select/default which drops for ALL full subscribers
	var received []int
	for {
		select {
		case msg := <-fastCh:
			received = append(received, msg)
		default:
			goto done
		}
	}
done:

	// This is expected behavior with the non-blocking broadcast design.
	// Production uses buffer=10, which is sufficient for typical usage.
	// A fully fair per-subscriber backpressure system would be more complex
	// and isn't needed for this use case.
	t.Logf("Fast subscriber received %d/10 messages with buffer=2 "+
		"(expected: messages dropped when buffer full)", len(received))
}

// =============================================================================
// Issue: Concurrent broadcast + unsubscribe is safe (VERIFIED)
//
// Both Broadcast and Unsubscribe acquire the same write lock (mu.Lock()),
// so they're mutually exclusive. This test verifies no panic occurs.
// =============================================================================

func TestVerified_ConcurrentBroadcastUnsubscribeIsSafe(t *testing.T) {
	b := New[int](5)

	var wg sync.WaitGroup
	var panicked atomic.Bool

	for round := 0; round < 100; round++ {
		_, unsub := b.Subscribe()

		wg.Add(2)

		// Broadcast in one goroutine
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked.Store(true)
				}
			}()
			b.Broadcast(42)
		}()

		// Unsubscribe in another goroutine
		go func() {
			defer wg.Done()
			unsub()
		}()
	}

	wg.Wait()

	if panicked.Load() {
		t.Error("Panic during concurrent broadcast + unsubscribe — " +
			"send to closed channel")
	}
	// If we reach here without panic, the mutex protection is working
}
