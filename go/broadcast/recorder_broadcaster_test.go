package broadcast

import (
	"sync"
	"testing"
	"time"

	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

/**
 * Test Plan:
 * - Subscribe: Creates subscription and returns unsubscribe function
 * - Broadcast: Sends recorder updates to all subscribers
 * - GetCachedStatus: Returns last status for specific recorder ID
 * - GetAllCachedStatuses: Returns all cached statuses
 * - Multiple recorders: Each recorder's status is cached separately
 * - Concurrent access: Thread-safe operations
 */

func makeRecorder(id, name string, signalStatus cmpb.SignalStatus) *sspb.Recorder {
	return &sspb.Recorder{
		RecorderID:   id,
		RecorderName: name,
		Info: &sspb.Recorder_Status{
			Status: &cmpb.RecorderStatus{
				RecorderID:   id,
				RecorderName: name,
				SignalStatus: signalStatus,
				RmsPercent:   50.0,
				Clipping:     false,
			},
		},
	}
}

func TestRecorderBroadcaster_Subscribe(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}

	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() = %d, want 1", b.SubscriberCount())
	}
}

func TestRecorderBroadcaster_Unsubscribe(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	_, unsub := b.Subscribe()
	unsub()

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() after unsubscribe = %d, want 0", b.SubscriberCount())
	}
}

func TestRecorderBroadcaster_Broadcast_SingleSubscriber(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	ch, unsub := b.Subscribe()
	defer unsub()

	recorder := makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_SIGNAL)
	b.Broadcast(recorder)

	select {
	case msg := <-ch:
		if msg.RecorderID != "rec-1" {
			t.Errorf("RecorderID = %q, want %q", msg.RecorderID, "rec-1")
		}
		if msg.RecorderName != "Recorder 1" {
			t.Errorf("RecorderName = %q, want %q", msg.RecorderName, "Recorder 1")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message")
	}
}

func TestRecorderBroadcaster_Broadcast_MultipleSubscribers(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	ch1, unsub1 := b.Subscribe()
	defer unsub1()

	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	recorder := makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_SIGNAL)
	b.Broadcast(recorder)

	// Both subscribers should receive
	for i, ch := range []<-chan *sspb.Recorder{ch1, ch2} {
		select {
		case msg := <-ch:
			if msg.RecorderID != "rec-1" {
				t.Errorf("subscriber %d: RecorderID = %q, want %q", i, msg.RecorderID, "rec-1")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for message", i)
		}
	}
}

func TestRecorderBroadcaster_GetCachedStatus(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	// No cached status initially
	if b.GetCachedStatus("rec-1") != nil {
		t.Error("GetCachedStatus() should return nil for unknown recorder")
	}

	recorder := makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_SIGNAL)
	b.Broadcast(recorder)

	cached := b.GetCachedStatus("rec-1")
	if cached == nil {
		t.Fatal("GetCachedStatus() returned nil after broadcast")
	}

	if cached.RecorderID != "rec-1" {
		t.Errorf("cached.RecorderID = %q, want %q", cached.RecorderID, "rec-1")
	}
}

func TestRecorderBroadcaster_GetCachedStatus_UpdatesOnNewBroadcast(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	// First broadcast
	recorder1 := makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_NO_SIGNAL)
	b.Broadcast(recorder1)

	cached := b.GetCachedStatus("rec-1")
	if cached.Info.(*sspb.Recorder_Status).Status.SignalStatus != cmpb.SignalStatus_NO_SIGNAL {
		t.Error("expected NO_SIGNAL status")
	}

	// Second broadcast with updated status
	recorder2 := makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_SIGNAL)
	b.Broadcast(recorder2)

	cached = b.GetCachedStatus("rec-1")
	if cached.Info.(*sspb.Recorder_Status).Status.SignalStatus != cmpb.SignalStatus_SIGNAL {
		t.Error("expected SIGNAL status after update")
	}
}

func TestRecorderBroadcaster_GetAllCachedStatuses(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	// No statuses initially
	statuses := b.GetAllCachedStatuses()
	if len(statuses) != 0 {
		t.Errorf("GetAllCachedStatuses() = %d items, want 0", len(statuses))
	}

	// Broadcast for multiple recorders
	b.Broadcast(makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_SIGNAL))
	b.Broadcast(makeRecorder("rec-2", "Recorder 2", cmpb.SignalStatus_NO_SIGNAL))
	b.Broadcast(makeRecorder("rec-3", "Recorder 3", cmpb.SignalStatus_UNKNOWN))

	statuses = b.GetAllCachedStatuses()
	if len(statuses) != 3 {
		t.Errorf("GetAllCachedStatuses() = %d items, want 3", len(statuses))
	}

	// Verify all recorders are present
	ids := make(map[string]bool)
	for _, s := range statuses {
		ids[s.RecorderID] = true
	}

	for _, id := range []string{"rec-1", "rec-2", "rec-3"} {
		if !ids[id] {
			t.Errorf("missing recorder %q in cached statuses", id)
		}
	}
}

func TestRecorderBroadcaster_MultipleRecorders_SeparateCache(t *testing.T) {
	b := NewRecorderBroadcaster(5)

	b.Broadcast(makeRecorder("rec-1", "Recorder 1", cmpb.SignalStatus_SIGNAL))
	b.Broadcast(makeRecorder("rec-2", "Recorder 2", cmpb.SignalStatus_NO_SIGNAL))

	cached1 := b.GetCachedStatus("rec-1")
	cached2 := b.GetCachedStatus("rec-2")

	if cached1.RecorderID != "rec-1" {
		t.Errorf("rec-1 cached status has wrong ID: %q", cached1.RecorderID)
	}

	if cached2.RecorderID != "rec-2" {
		t.Errorf("rec-2 cached status has wrong ID: %q", cached2.RecorderID)
	}

	// Statuses should be different
	status1 := cached1.Info.(*sspb.Recorder_Status).Status.SignalStatus
	status2 := cached2.Info.(*sspb.Recorder_Status).Status.SignalStatus

	if status1 == status2 {
		t.Error("recorders should have different signal statuses")
	}
}

func TestRecorderBroadcaster_Concurrent(t *testing.T) {
	b := NewRecorderBroadcaster(100)

	var wg sync.WaitGroup
	numSubscribers := 5
	numMessages := 50

	// Start subscribers
	subscribers := make([]<-chan *sspb.Recorder, numSubscribers)
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
		go func(idx int, ch <-chan *sspb.Recorder) {
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
			recorder := makeRecorder("rec-1", "Recorder", cmpb.SignalStatus_SIGNAL)
			b.Broadcast(recorder)
		}(i)
	}

	// Wait for all broadcasts to complete
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
