package broadcast

import (
	"sync"
	"testing"
	"time"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

/**
 * Test Plan:
 * - Subscribe: Creates subscription and returns unsubscribe function
 * - Broadcast: Sends session updates to all subscribers
 * - Multiple subscribers: All receive the same updates
 * - Unsubscribe: Removes subscriber correctly
 * - Concurrent access: Thread-safe operations
 */

func makeSession(id string) *sspb.Session {
	return &sspb.Session{
		ID: id,
		Info: &sspb.Session_Updated{
			Updated: &sspb.SessionInfo{
				Name: "Test Session",
			},
		},
	}
}

func makeSessionRemoved(id string) *sspb.Session {
	return &sspb.Session{
		ID: id,
		Info: &sspb.Session_Removed{
			Removed: &sspb.SessionRemoved{},
		},
	}
}

func TestSessionBroadcaster_Subscribe(t *testing.T) {
	b := NewSessionBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}

	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() = %d, want 1", b.SubscriberCount())
	}
}

func TestSessionBroadcaster_Unsubscribe(t *testing.T) {
	b := NewSessionBroadcaster(5)

	_, unsub := b.Subscribe()
	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() after unsubscribe = %d, want 0", b.SubscriberCount())
	}
}

func TestSessionBroadcaster_Unsubscribe_Idempotent(t *testing.T) {
	b := NewSessionBroadcaster(5)

	_, unsub := b.Subscribe()

	// Multiple unsubscribes should not panic
	unsub()
	unsub()
	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", b.SubscriberCount())
	}
}

func TestSessionBroadcaster_Broadcast_SingleSubscriber(t *testing.T) {
	b := NewSessionBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	session := makeSession("session-1")
	b.Broadcast(session)

	select {
	case msg := <-ch:
		if msg.ID != "session-1" {
			t.Errorf("Session.ID = %q, want %q", msg.ID, "session-1")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message")
	}
}

func TestSessionBroadcaster_Broadcast_MultipleSubscribers(t *testing.T) {
	b := NewSessionBroadcaster(5)

	ch1, unsub1 := b.Subscribe()
	defer unsub1()

	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	ch3, unsub3 := b.Subscribe()
	defer unsub3()

	session := makeSession("session-1")
	b.Broadcast(session)

	// All subscribers should receive
	for i, ch := range []<-chan *sspb.Session{ch1, ch2, ch3} {
		select {
		case msg := <-ch:
			if msg.ID != "session-1" {
				t.Errorf("subscriber %d: Session.ID = %q, want %q", i, msg.ID, "session-1")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for message", i)
		}
	}
}

func TestSessionBroadcaster_Broadcast_NoSubscribers(t *testing.T) {
	b := NewSessionBroadcaster(5)

	// Should not panic
	b.Broadcast(makeSession("session-1"))
	b.Broadcast(makeSessionRemoved("session-2"))
}

func TestSessionBroadcaster_Broadcast_SessionUpdated(t *testing.T) {
	b := NewSessionBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	session := makeSession("session-1")
	b.Broadcast(session)

	msg := <-ch

	// Check it's an Updated message
	updated, ok := msg.Info.(*sspb.Session_Updated)
	if !ok {
		t.Fatal("expected Session_Updated message")
	}

	if updated.Updated.Name != "Test Session" {
		t.Errorf("Name = %q, want %q", updated.Updated.Name, "Test Session")
	}
}

func TestSessionBroadcaster_Broadcast_SessionRemoved(t *testing.T) {
	b := NewSessionBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	session := makeSessionRemoved("session-1")
	b.Broadcast(session)

	msg := <-ch

	// Check it's a Removed message
	_, ok := msg.Info.(*sspb.Session_Removed)
	if !ok {
		t.Fatal("expected Session_Removed message")
	}

	if msg.ID != "session-1" {
		t.Errorf("ID = %q, want %q", msg.ID, "session-1")
	}
}

func TestSessionBroadcaster_BufferOverflow(t *testing.T) {
	b := NewSessionBroadcaster(2) // Small buffer

	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the buffer
	b.Broadcast(makeSession("session-1"))
	b.Broadcast(makeSession("session-2"))

	// This should be dropped (buffer full)
	b.Broadcast(makeSession("session-3"))

	// Read the first two messages
	msg1 := <-ch
	msg2 := <-ch

	if msg1.ID != "session-1" || msg2.ID != "session-2" {
		t.Errorf("expected session-1 and session-2, got %q and %q", msg1.ID, msg2.ID)
	}

	// Channel should be empty (session-3 was dropped)
	select {
	case msg := <-ch:
		t.Errorf("unexpected message in channel: %q", msg.ID)
	default:
		// Expected
	}
}

func TestSessionBroadcaster_Concurrent(t *testing.T) {
	b := NewSessionBroadcaster(100)

	var wg sync.WaitGroup
	numSubscribers := 5
	numMessages := 50

	// Start subscribers
	subscribers := make([]<-chan *sspb.Session, numSubscribers)
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
		go func(idx int, ch <-chan *sspb.Session) {
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
			b.Broadcast(makeSession("session-1"))
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

func TestSessionBroadcaster_SubscribeUnsubscribeConcurrent(t *testing.T) {
	b := NewSessionBroadcaster(10)

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
			b.Broadcast(makeSession("session-1"))
		}()
	}

	wg.Wait()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", b.SubscriberCount())
	}
}
