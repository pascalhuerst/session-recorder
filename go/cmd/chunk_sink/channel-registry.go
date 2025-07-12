package main

import (
	"fmt"

	"github.com/google/uuid"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
)

type RecorderChannelRegistry interface {
	GetRecorderChannel(uuid.UUID) (chan *sspb.Session, error)
	RegisterRecorder(uuid.UUID) (chan *sspb.Session, error)
	UnregisterRecorder(uuid.UUID) error
	Reset() error
}

type SessionUpdateBroadcaster struct {
	channels map[uuid.UUID]chan *sspb.Session
}

func NewSessionUpdateBroadcaster() *SessionUpdateBroadcaster {
	return &SessionUpdateBroadcaster{
		channels: make(map[uuid.UUID]chan *sspb.Session),
	}
}

func (r *SessionUpdateBroadcaster) GetRecorderChannel(id uuid.UUID) (chan *sspb.Session, error) {
	if _, exists := r.channels[id]; !exists {
		return nil, fmt.Errorf("recorder not found")
	}

	return r.channels[id], nil
}

func (r *SessionUpdateBroadcaster) RegisterRecorder(id uuid.UUID) (chan *sspb.Session, error) {
	if _, exists := r.channels[id]; !exists {
		r.channels[id] = make(chan *sspb.Session)
	}

	return r.channels[id], nil
}

func (r *SessionUpdateBroadcaster) UnregisterRecorder(id uuid.UUID) error {
	if _, exists := r.channels[id]; !exists {
		return fmt.Errorf("recorder not found")
	}

	close(r.channels[id])
	delete(r.channels, id)
	return nil
}

func (r *SessionUpdateBroadcaster) Reset() error {
	for id := range r.channels {
		if err := r.UnregisterRecorder(id); err != nil {
			return err
		}
	}

	return nil
}
