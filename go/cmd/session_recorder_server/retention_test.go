package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	"github.com/pascalhuerst/session-recorder/storage"
)

// fakeRetentionStorage implements just the two Storage methods the retention
// sweeper touches; the embedded nil interface makes any other call panic (which
// would surface an unintended dependency).
type fakeRetentionStorage struct {
	storage.Storage
	sessions []storage.SessionRef
	deleted  []uuid.UUID
}

func (f *fakeRetentionStorage) SnapshotSessions() []storage.SessionRef {
	return f.sessions
}

func (f *fakeRetentionStorage) DeleteSession(_ context.Context, _, sessionID uuid.UUID) error {
	f.deleted = append(f.deleted, sessionID)
	return nil
}

func TestSweepExpiredSessions(t *testing.T) {
	const retention = 4 * 24 * time.Hour
	now := time.Now()
	recID := uuid.New()

	ref := func(state storage.SessionState, keep bool, end time.Time) storage.SessionRef {
		return storage.SessionRef{
			RecorderID: recID,
			SessionID:  uuid.New(),
			Session:    storage.Session{State: state, Keep: keep, EndTime: end},
		}
	}

	old := now.Add(-5 * 24 * time.Hour)
	recent := now.Add(-1 * time.Hour)

	expired := ref(storage.SessionStateFinished, false, old)   // deleted
	kept := ref(storage.SessionStateFinished, true, old)       // Keep set -> retained
	young := ref(storage.SessionStateFinished, false, recent)  // too new -> retained
	recording := ref(storage.SessionStateRecording, false, time.Time{})
	processing := ref(storage.SessionStateProcessing, false, time.Time{})
	errored := ref(storage.SessionStateError, false, old) // ERROR is never reaped

	// Expired + non-kept, but has a rendered segment -> the whole session is
	// retained (rendered segments are recordings we keep for now).
	withRendered := ref(storage.SessionStateFinished, false, old)
	withRendered.Session.Segments = map[uuid.UUID]storage.Segment{
		uuid.New(): {State: storage.SegmentStateFinished},
	}

	// Expired + non-kept with only a non-rendered segment -> still deleted.
	withQueuedSeg := ref(storage.SessionStateFinished, false, old)
	withQueuedSeg.Session.Segments = map[uuid.UUID]storage.Segment{
		uuid.New(): {State: storage.SegmentStateQueued},
	}

	fake := &fakeRetentionStorage{sessions: []storage.SessionRef{
		expired, kept, young, recording, processing, errored, withRendered, withQueuedSeg,
	}}

	bc := broadcast.NewSessionBroadcaster(10)
	updates, unsubscribe := bc.Subscribe()
	defer unsubscribe()

	h := &SessionSourceHandler{sessionStorage: fake, sessionBroadcaster: bc}

	h.sweepExpiredSessions(context.Background(), retention)

	// Both the plain expired session and the one whose only segment is not
	// rendered are deleted; the session with a rendered segment is retained.
	wantDeleted := map[uuid.UUID]bool{
		expired.SessionID:       true,
		withQueuedSeg.SessionID: true,
	}
	gotDeleted := map[uuid.UUID]bool{}
	for _, id := range fake.deleted {
		gotDeleted[id] = true
	}
	if len(fake.deleted) != len(wantDeleted) {
		t.Fatalf("expected %d deletions, got %d: %v", len(wantDeleted), len(fake.deleted), fake.deleted)
	}
	for id := range wantDeleted {
		if !gotDeleted[id] {
			t.Fatalf("expected session %s to be deleted, deleted set was %v", id, fake.deleted)
		}
	}
	if gotDeleted[withRendered.SessionID] {
		t.Fatal("session with a rendered segment must be retained")
	}

	// Each deletion must be broadcast as a removal so the UI drops the card.
	removed := map[string]bool{}
	for {
		select {
		case msg := <-updates:
			if msg.GetRemoved() == nil {
				t.Fatalf("expected Session_Removed, got %T", msg.Info)
			}
			removed[msg.ID] = true
			continue
		default:
		}
		break
	}
	for id := range wantDeleted {
		if !removed[id.String()] {
			t.Fatalf("expected a removal broadcast for %s, got %v", id, removed)
		}
	}
}
