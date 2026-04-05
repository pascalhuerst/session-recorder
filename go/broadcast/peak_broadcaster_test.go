package broadcast

import (
	"sync"
	"testing"
	"time"

	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

func TestPeakBroadcaster_SubscribeAndBroadcast(t *testing.T) {
	b := NewPeakBroadcaster(10)

	ch, unsub := b.Subscribe()
	defer unsub()

	data := &sspb.WaveformPeakData{SessionID: "s1", Peaks: []int32{-10, 20}}
	b.Broadcast(data)

	select {
	case got := <-ch:
		if got.SessionID != "s1" {
			t.Errorf("SessionID = %s, want s1", got.SessionID)
		}
		if len(got.Peaks) != 2 {
			t.Errorf("Peaks len = %d, want 2", len(got.Peaks))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestPeakBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewPeakBroadcaster(10)

	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	if b.SubscriberCount() != 2 {
		t.Fatalf("SubscriberCount = %d, want 2", b.SubscriberCount())
	}

	data := &sspb.WaveformPeakData{SessionID: "s1"}
	b.Broadcast(data)

	for i, ch := range []<-chan *sspb.WaveformPeakData{ch1, ch2} {
		select {
		case got := <-ch:
			if got.SessionID != "s1" {
				t.Errorf("subscriber %d: SessionID = %s, want s1", i, got.SessionID)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestPeakBroadcaster_Unsubscribe(t *testing.T) {
	b := NewPeakBroadcaster(10)

	_, unsub := b.Subscribe()

	if b.SubscriberCount() != 1 {
		t.Fatalf("SubscriberCount = %d, want 1", b.SubscriberCount())
	}

	unsub()

	if b.SubscriberCount() != 0 {
		t.Fatalf("SubscriberCount after unsub = %d, want 0", b.SubscriberCount())
	}
}

func TestPeakBroadcaster_DoubleUnsubscribe(t *testing.T) {
	b := NewPeakBroadcaster(10)

	_, unsub := b.Subscribe()
	unsub()
	unsub() // should not panic
}

func TestPeakBroadcaster_BufferFullDropsMessage(t *testing.T) {
	b := NewPeakBroadcaster(1) // buffer of 1

	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the buffer
	b.Broadcast(&sspb.WaveformPeakData{SessionID: "s1"})
	// This should be dropped (buffer full)
	b.Broadcast(&sspb.WaveformPeakData{SessionID: "s2"})

	got := <-ch
	if got.SessionID != "s1" {
		t.Errorf("expected first message s1, got %s", got.SessionID)
	}

	// Channel should be empty now
	select {
	case msg := <-ch:
		t.Errorf("expected empty channel, got message for %s", msg.SessionID)
	default:
		// expected
	}
}

func TestPeakBroadcaster_ConcurrentAccess(t *testing.T) {
	b := NewPeakBroadcaster(100)

	var wg sync.WaitGroup

	// Concurrent subscribers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, unsub := b.Subscribe()
			time.Sleep(10 * time.Millisecond)
			unsub()
		}()
	}

	// Concurrent broadcasts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Broadcast(&sspb.WaveformPeakData{SessionID: "s1"})
		}()
	}

	wg.Wait()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount after all unsub = %d, want 0", b.SubscriberCount())
	}
}

func TestPeakBroadcaster_NoSubscribers(t *testing.T) {
	b := NewPeakBroadcaster(10)

	// Should not panic
	b.Broadcast(&sspb.WaveformPeakData{SessionID: "s1"})
}
