package broadcast

import (
	"sync"
	"testing"
	"time"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/**
 * Test Plan: AudioBroadcaster
 *
 * Scenario: Subscribe creates channel and unsubscribe function
 *   Given a new AudioBroadcaster
 *   When Subscribe is called
 *   Then a non-nil channel is returned
 *   And subscriber count is 1
 *
 * Scenario: Unsubscribe removes subscriber
 *   Given a subscribed client
 *   When unsubscribe is called
 *   Then subscriber count is 0
 *   And the channel is closed
 *
 * Scenario: Broadcast sends to all subscribers
 *   Given multiple subscribers
 *   When Broadcast is called with an audio chunk
 *   Then all subscribers receive the chunk
 *
 * Scenario: Buffer overflow drops chunks
 *   Given a subscriber with full buffer
 *   When Broadcast is called
 *   Then the chunk is dropped (non-blocking)
 *
 * Scenario: Concurrent access is thread-safe
 *   Given multiple goroutines subscribing and unsubscribing
 *   When broadcasts occur concurrently
 *   Then no race conditions occur
 */

func makeAudioChunk(sessionID string, samples []int32) *sspb.AudioChunk {
	return &sspb.AudioChunk{
		SessionID:   sessionID,
		Samples:     samples,
		ChunkNumber: 1,
		Timestamp:   timestamppb.Now(),
	}
}

func TestAudioBroadcaster_Subscribe(t *testing.T) {
	b := NewAudioBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}

	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() = %d, want 1", b.SubscriberCount())
	}
}

func TestAudioBroadcaster_Unsubscribe(t *testing.T) {
	b := NewAudioBroadcaster(5)

	_, unsub := b.Subscribe()

	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() before unsubscribe = %d, want 1", b.SubscriberCount())
	}

	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() after unsubscribe = %d, want 0", b.SubscriberCount())
	}
}

func TestAudioBroadcaster_Unsubscribe_Idempotent(t *testing.T) {
	b := NewAudioBroadcaster(5)

	_, unsub := b.Subscribe()

	// Multiple unsubscribes should not panic
	unsub()
	unsub()
	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", b.SubscriberCount())
	}
}

func TestAudioBroadcaster_Broadcast_SingleSubscriber(t *testing.T) {
	b := NewAudioBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	chunk := makeAudioChunk("session-1", []int32{1, 2, 3, 4, 5})
	b.Broadcast(chunk)

	select {
	case msg := <-ch:
		if msg.SessionID != "session-1" {
			t.Errorf("SessionID = %q, want %q", msg.SessionID, "session-1")
		}
		if len(msg.Samples) != 5 {
			t.Errorf("Samples length = %d, want 5", len(msg.Samples))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message")
	}
}

func TestAudioBroadcaster_Broadcast_MultipleSubscribers(t *testing.T) {
	b := NewAudioBroadcaster(5)

	ch1, unsub1 := b.Subscribe()
	defer unsub1()

	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	ch3, unsub3 := b.Subscribe()
	defer unsub3()

	chunk := makeAudioChunk("session-1", []int32{10, 20, 30})
	b.Broadcast(chunk)

	// All subscribers should receive
	for i, ch := range []<-chan *sspb.AudioChunk{ch1, ch2, ch3} {
		select {
		case msg := <-ch:
			if msg.SessionID != "session-1" {
				t.Errorf("subscriber %d: SessionID = %q, want %q", i, msg.SessionID, "session-1")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for message", i)
		}
	}
}

func TestAudioBroadcaster_Broadcast_NoSubscribers(t *testing.T) {
	b := NewAudioBroadcaster(5)

	// Should not panic
	chunk := makeAudioChunk("session-1", []int32{1, 2, 3})
	b.Broadcast(chunk)
	b.Broadcast(chunk)
}

func TestAudioBroadcaster_BufferOverflow(t *testing.T) {
	b := NewAudioBroadcaster(2) // Small buffer

	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the buffer
	b.Broadcast(makeAudioChunk("session-1", []int32{1}))
	b.Broadcast(makeAudioChunk("session-2", []int32{2}))

	// This should be dropped (buffer full)
	b.Broadcast(makeAudioChunk("session-3", []int32{3}))

	// Read the first two messages
	msg1 := <-ch
	msg2 := <-ch

	if msg1.SessionID != "session-1" || msg2.SessionID != "session-2" {
		t.Errorf("expected session-1 and session-2, got %q and %q", msg1.SessionID, msg2.SessionID)
	}

	// Channel should be empty (session-3 was dropped)
	select {
	case msg := <-ch:
		t.Errorf("unexpected message in channel: %q", msg.SessionID)
	default:
		// Expected
	}
}

func TestAudioBroadcaster_Concurrent(t *testing.T) {
	b := NewAudioBroadcaster(100)

	var wg sync.WaitGroup
	numSubscribers := 5
	numMessages := 50

	// Start subscribers
	subscribers := make([]<-chan *sspb.AudioChunk, numSubscribers)
	unsubscribers := make([]func(), numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		ch, unsub := b.Subscribe()
		subscribers[i] = ch
		unsubscribers[i] = unsub
	}

	// Consume messages
	receivedCounts := make([]int, numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(idx int, ch <-chan *sspb.AudioChunk) {
			defer wg.Done()
			for range ch {
				receivedCounts[idx]++
			}
		}(i, subscribers[i])
	}

	// Broadcast messages from multiple goroutines
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.Broadcast(makeAudioChunk("session-1", []int32{int32(n)}))
		}(i)
	}

	// Wait for broadcasts to complete
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe all
	for _, unsub := range unsubscribers {
		unsub()
	}

	wg.Wait()

	// Each subscriber should have received all messages
	for i, count := range receivedCounts {
		if count != numMessages {
			t.Errorf("subscriber %d received %d messages, want %d", i, count, numMessages)
		}
	}
}

func TestAudioBroadcaster_SubscribeUnsubscribeConcurrent(t *testing.T) {
	b := NewAudioBroadcaster(10)

	var wg sync.WaitGroup

	// Concurrent subscribe/unsubscribe
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsub := b.Subscribe()
			_ = ch
			time.Sleep(time.Millisecond)
			unsub()
		}()
	}

	// Concurrent broadcasts
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Broadcast(makeAudioChunk("session-1", []int32{1, 2, 3}))
		}()
	}

	wg.Wait()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", b.SubscriberCount())
	}
}
