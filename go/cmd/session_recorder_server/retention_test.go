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

	fake := &fakeRetentionStorage{sessions: []storage.SessionRef{
		expired, kept, young, recording, processing, errored,
	}}

	bc := broadcast.NewSessionBroadcaster(10)
	updates, unsubscribe := bc.Subscribe()
	defer unsubscribe()

	h := &SessionSourceHandler{sessionStorage: fake, sessionBroadcaster: bc}

	h.sweepExpiredSessions(context.Background(), retention)

	if len(fake.deleted) != 1 {
		t.Fatalf("expected exactly 1 deletion, got %d: %v", len(fake.deleted), fake.deleted)
	}
	if fake.deleted[0] != expired.SessionID {
		t.Fatalf("deleted wrong session: got %s, want %s", fake.deleted[0], expired.SessionID)
	}

	// The deletion must be broadcast as a removal so the UI drops the card.
	select {
	case msg := <-updates:
		if msg.ID != expired.SessionID.String() {
			t.Fatalf("removal broadcast for wrong session: got %s, want %s", msg.ID, expired.SessionID)
		}
		if msg.GetRemoved() == nil {
			t.Fatalf("expected Session_Removed, got %T", msg.Info)
		}
	default:
		t.Fatal("expected a removal broadcast, got none")
	}
}
