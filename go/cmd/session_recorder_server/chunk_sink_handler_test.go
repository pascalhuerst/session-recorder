package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	"github.com/pascalhuerst/session-recorder/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/**
 * Test Plan: one close per session, after its last chunk
 *
 * Scenario: The recorder's last chunk arrives after its "no signal" status
 *   Given a recording session
 *   When the status flips to NO_SIGNAL and a chunk for the session follows
 *   Then the chunk is stored
 *   And the session is closed exactly once, after that chunk
 *
 * Scenario: Silence without a trailing chunk
 *   When the status flips to NO_SIGNAL and nothing else arrives
 *   Then the session is closed once the close delay has passed
 *
 * Scenario: Chunks keep arriving after the status (send buffer draining)
 *   When chunks for the session keep coming at shorter intervals than the delay
 *   Then the session stays open until they stop
 *
 * Scenario: A chunk for an already closed session
 *   Given the storage refuses the chunk with ErrSessionClosed
 *   When further NO_SIGNAL statuses arrive
 *   Then the session is not closed a second time
 *
 * Scenario: The recorder moves on to a new session before the delay is over
 *   When a chunk for a new session arrives
 *   Then the pending close is dropped (the storage closes the old session
 *     itself when it switches) and the new session is not closed
 */

type storedCall struct {
	kind    string // "chunk" or "close"
	session uuid.UUID
}

type fakeChunkStorage struct {
	storage.Storage
	mu     sync.Mutex
	calls  []storedCall
	closed map[uuid.UUID]bool // sessions the fake refuses chunks for
}

func (f *fakeChunkStorage) SafeChunks(_ context.Context, _, sessionID uuid.UUID, _ string, _ time.Time, _ []int16) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed[sessionID] {
		return storage.ErrSessionClosed
	}
	f.calls = append(f.calls, storedCall{"chunk", sessionID})
	return nil
}

func (f *fakeChunkStorage) CloseRecordingSession(_ context.Context, _, sessionID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, storedCall{"close", sessionID})
	if f.closed == nil {
		f.closed = make(map[uuid.UUID]bool)
	}
	f.closed[sessionID] = true
	return nil
}

func (f *fakeChunkStorage) EnsureRecorderExists(context.Context, uuid.UUID, string) {}

func (f *fakeChunkStorage) snapshot() []storedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]storedCall(nil), f.calls...)
}

func (f *fakeChunkStorage) closes(sessionID uuid.UUID) int {
	n := 0
	for _, call := range f.snapshot() {
		if call.kind == "close" && call.session == sessionID {
			n++
		}
	}
	return n
}

const testCloseDelay = 60 * time.Millisecond

type handlerFixture struct {
	t        *testing.T
	handler  *ChunkSinkHandler
	storage  *fakeChunkStorage
	recorder uuid.UUID
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	fake := &fakeChunkStorage{}
	handler := NewChunkSinkHandler(fake, broadcast.NewRecorderBroadcaster(8))
	handler.closeDelay = testCloseDelay
	return &handlerFixture{t: t, handler: handler, storage: fake, recorder: uuid.New()}
}

func (x *handlerFixture) status(signal cmpb.SignalStatus) {
	x.t.Helper()
	err := x.handler.setRecorderStatus(context.Background(), &cmpb.RecorderStatus{
		RecorderID: x.recorder.String(), RecorderName: "test", SignalStatus: signal,
	})
	if err != nil {
		x.t.Fatalf("setRecorderStatus: %v", err)
	}
}

func (x *handlerFixture) chunk(sessionID uuid.UUID) {
	x.t.Helper()
	err := x.handler.setChunks(context.Background(), &cspb.Chunks{
		RecorderID: x.recorder.String(), SessionID: sessionID.String(),
		TimeCreated: timestamppb.Now(), Data: make([]byte, 64),
	})
	if err != nil {
		x.t.Fatalf("setChunks: %v", err)
	}
}

func TestTrailingChunkIsStoredBeforeTheSingleClose(t *testing.T) {
	x := newHandlerFixture(t)
	session := uuid.New()

	x.status(cmpb.SignalStatus_SIGNAL)
	x.chunk(session)
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	x.chunk(session) // the tail, overtaken by the status
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	x.status(cmpb.SignalStatus_NO_SIGNAL)

	if n := x.storage.closes(session); n != 0 {
		t.Fatalf("session closed %d times before the close delay passed", n)
	}
	time.Sleep(4 * testCloseDelay)
	x.status(cmpb.SignalStatus_NO_SIGNAL)

	want := []storedCall{{"chunk", session}, {"chunk", session}, {"close", session}}
	got := x.storage.snapshot()
	if len(got) != len(want) {
		t.Fatalf("storage calls = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("storage calls = %v, want %v", got, want)
		}
	}
}

func TestSilenceWithoutTrailingChunkClosesAfterTheDelay(t *testing.T) {
	x := newHandlerFixture(t)
	session := uuid.New()

	x.status(cmpb.SignalStatus_SIGNAL)
	x.chunk(session)
	x.status(cmpb.SignalStatus_NO_SIGNAL)

	time.Sleep(testCloseDelay / 3)
	if n := x.storage.closes(session); n != 0 {
		t.Fatalf("session closed %d times right after the status", n)
	}
	time.Sleep(4 * testCloseDelay)
	if n := x.storage.closes(session); n != 1 {
		t.Fatalf("session closed %d times, want once", n)
	}
}

func TestChunksAfterTheStatusKeepTheSessionOpen(t *testing.T) {
	x := newHandlerFixture(t)
	session := uuid.New()

	x.status(cmpb.SignalStatus_SIGNAL)
	x.chunk(session)
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	for range 6 { // 3 close delays' worth of chunks, each well inside the delay
		time.Sleep(testCloseDelay / 2)
		x.chunk(session)
	}
	if n := x.storage.closes(session); n != 0 {
		t.Fatalf("session closed %d times while chunks were still arriving", n)
	}
	time.Sleep(4 * testCloseDelay)
	if n := x.storage.closes(session); n != 1 {
		t.Fatalf("session closed %d times, want once after the chunks stopped", n)
	}
	if got := len(x.storage.snapshot()); got != 8 { // 7 chunks + 1 close
		t.Errorf("%d storage calls, want 7 chunks and 1 close", got)
	}
}

func TestChunkForAClosedSessionDoesNotCloseItAgain(t *testing.T) {
	x := newHandlerFixture(t)
	session := uuid.New()

	x.status(cmpb.SignalStatus_SIGNAL)
	x.chunk(session)
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	time.Sleep(4 * testCloseDelay)

	x.chunk(session) // refused by the storage
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	x.status(cmpb.SignalStatus_SIGNAL) // and even a full status cycle
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	time.Sleep(4 * testCloseDelay)

	if n := x.storage.closes(session); n != 1 {
		t.Fatalf("session closed %d times, want once", n)
	}
}

func TestNewSessionDropsThePendingClose(t *testing.T) {
	x := newHandlerFixture(t)
	first, second := uuid.New(), uuid.New()

	x.status(cmpb.SignalStatus_SIGNAL)
	x.chunk(first)
	x.status(cmpb.SignalStatus_NO_SIGNAL)
	x.status(cmpb.SignalStatus_SIGNAL)
	x.chunk(second) // inside the delay
	time.Sleep(4 * testCloseDelay)

	if n := x.storage.closes(second); n != 0 {
		t.Errorf("the new session was closed %d times", n)
	}
	// The handler still closes the previous session on a switch (the storage
	// does too, and ignores the second request) - but only that once.
	if n := x.storage.closes(first); n != 1 {
		t.Errorf("the previous session was closed %d times by the handler, want once", n)
	}
}
