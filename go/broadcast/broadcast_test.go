package broadcast

import (
	"sync"
	"testing"
	"time"
)

/**
 * Test Plan:
 * - Subscribe: Creates subscription channel and returns unsubscribe function
 * - Broadcast: Sends message to all subscribers
 * - Unsubscribe: Removes subscriber and closes channel
 * - Concurrent access: Thread-safe with multiple goroutines
 * - Buffer overflow: Drops messages when buffer is full
 * - GetLastValue: Returns last broadcasted value
 * - SubscriberCount: Returns correct count
 */

func TestBroadcaster_Subscribe(t *testing.T) {
	b := New[string](5)

	ch, unsub := b.Subscribe()
	defer unsub()

	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}

	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() = %d, want 1", b.SubscriberCount())
	}
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := New[string](5)

	_, unsub := b.Subscribe()

	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() before unsubscribe = %d, want 1", b.SubscriberCount())
	}

	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() after unsubscribe = %d, want 0", b.SubscriberCount())
	}
}

func TestBroadcaster_Unsubscribe_Idempotent(t *testing.T) {
	b := New[string](5)

	_, unsub := b.Subscribe()

	// Unsubscribe multiple times should not panic
	unsub()
	unsub()
	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", b.SubscriberCount())
	}
}

func TestBroadcaster_Broadcast_SingleSubscriber(t *testing.T) {
	b := New[string](5)

	ch, unsub := b.Subscribe()
	defer unsub()

	b.Broadcast("hello")

	select {
	case msg := <-ch:
		if msg != "hello" {
			t.Errorf("received message = %q, want %q", msg, "hello")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message")
	}
}

func TestBroadcaster_Broadcast_MultipleSubscribers(t *testing.T) {
	b := New[string](5)

	ch1, unsub1 := b.Subscribe()
	defer unsub1()

	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	ch3, unsub3 := b.Subscribe()
	defer unsub3()

	b.Broadcast("test message")

	// All subscribers should receive the message
	for i, ch := range []<-chan string{ch1, ch2, ch3} {
		select {
		case msg := <-ch:
			if msg != "test message" {
				t.Errorf("subscriber %d: received = %q, want %q", i, msg, "test message")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for message", i)
		}
	}
}

func TestBroadcaster_Broadcast_NoSubscribers(t *testing.T) {
	b := New[string](5)

	// Should not panic when broadcasting with no subscribers
	b.Broadcast("hello")
	b.Broadcast("world")
}

func TestBroadcaster_BufferOverflow(t *testing.T) {
	b := New[string](2) // Small buffer

	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the buffer
	b.Broadcast("msg1")
	b.Broadcast("msg2")

	// This should be dropped (buffer full, non-blocking)
	b.Broadcast("msg3")

	// Read the first two messages
	msg1 := <-ch
	msg2 := <-ch

	if msg1 != "msg1" || msg2 != "msg2" {
		t.Errorf("expected msg1 and msg2, got %q and %q", msg1, msg2)
	}

	// Channel should be empty now (msg3 was dropped)
	select {
	case msg := <-ch:
		t.Errorf("unexpected message in channel: %q", msg)
	default:
		// Expected - channel should be empty
	}
}

func TestBroadcaster_GetLastValue(t *testing.T) {
	b := New[string](5)

	// No value yet
	if b.GetLastValue() != nil {
		t.Error("GetLastValue() should be nil before any broadcast")
	}

	b.Broadcast("first")

	lastVal := b.GetLastValue()
	if lastVal == nil || *lastVal != "first" {
		t.Errorf("GetLastValue() = %v, want 'first'", lastVal)
	}

	b.Broadcast("second")

	lastVal = b.GetLastValue()
	if lastVal == nil || *lastVal != "second" {
		t.Errorf("GetLastValue() = %v, want 'second'", lastVal)
	}
}

func TestBroadcaster_Concurrent(t *testing.T) {
	b := New[int](100)

	var wg sync.WaitGroup
	numSubscribers := 10
	numMessages := 100

	// Start subscribers
	subscribers := make([]<-chan int, numSubscribers)
	unsubscribers := make([]func(), numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		ch, unsub := b.Subscribe()
		subscribers[i] = ch
		unsubscribers[i] = unsub
	}

	defer func() {
		for _, unsub := range unsubscribers {
			unsub()
		}
	}()

	// Consume messages in goroutines
	receivedCounts := make([]int, numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(idx int, ch <-chan int) {
			defer wg.Done()
			for range ch {
				receivedCounts[idx]++
			}
		}(i, subscribers[i])
	}

	// Broadcast messages
	for i := 0; i < numMessages; i++ {
		b.Broadcast(i)
	}

	// Unsubscribe all (closes channels)
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

func TestBroadcaster_SubscribeUnsubscribeConcurrent(t *testing.T) {
	b := New[string](10)

	var wg sync.WaitGroup

	// Concurrent subscribe/unsubscribe
	for i := 0; i < 100; i++ {
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
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.Broadcast("message")
		}(i)
	}

	wg.Wait()

	// Should end with 0 subscribers
	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", b.SubscriberCount())
	}
}
