package broadcast

import (
	"context"
	"sync"
	"time"

	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/rs/zerolog/log"
)

const (
	// DefaultRecorderStatusTimeout is the default duration after which a recorder
	// is considered stale if no status updates are received.
	DefaultRecorderStatusTimeout = 10 * time.Second

	// recorderTimeoutCheckInterval is how often to check for stale recorders.
	recorderTimeoutCheckInterval = 3 * time.Second
)

// RecorderBroadcaster is a specialized broadcaster for recorder status updates.
// It caches the last known status per recorder so new clients can receive
// the current state immediately upon subscription.
// It also detects stale recorders that stop sending updates and broadcasts
// NO_SIGNAL status after a timeout.
type RecorderBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan *sspb.Recorder]struct{}
	bufferSize  int
	// Cache last status per recorder ID
	lastStatus map[string]*sspb.Recorder
	// Track when each recorder last sent an update
	lastUpdate map[string]time.Time
	// How long before marking a recorder as stale
	statusTimeout time.Duration
	// Signal to stop the timeout checker goroutine
	stopTimeout chan struct{}
}

// NewRecorderBroadcaster creates a new RecorderBroadcaster.
func NewRecorderBroadcaster(bufferSize int) *RecorderBroadcaster {
	return &RecorderBroadcaster{
		subscribers:   make(map[chan *sspb.Recorder]struct{}),
		bufferSize:    bufferSize,
		lastStatus:    make(map[string]*sspb.Recorder),
		lastUpdate:    make(map[string]time.Time),
		statusTimeout: DefaultRecorderStatusTimeout,
		stopTimeout:   make(chan struct{}),
	}
}

// SetStatusTimeout configures the recorder status timeout duration.
func (b *RecorderBroadcaster) SetStatusTimeout(timeout time.Duration) {
	b.statusTimeout = timeout
}

// Start begins the timeout checker goroutine that detects stale recorders.
func (b *RecorderBroadcaster) Start(ctx context.Context) {
	go b.runTimeoutChecker(ctx)
}

// Stop stops the timeout checker goroutine.
func (b *RecorderBroadcaster) Stop() {
	close(b.stopTimeout)
}

// Subscribe creates a new subscription and returns the channel and unsubscribe function.
func (b *RecorderBroadcaster) Subscribe() (ch <-chan *sspb.Recorder, unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subCh := make(chan *sspb.Recorder, b.bufferSize)
	b.subscribers[subCh] = struct{}{}

	log.Debug().Int("subscribers", len(b.subscribers)).Msg("New recorder subscriber added")

	return subCh, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[subCh]; ok {
			delete(b.subscribers, subCh)
			close(subCh)
			log.Debug().Int("subscribers", len(b.subscribers)).Msg("Recorder subscriber removed")
		}
	}
}

// Broadcast sends a recorder update to all subscribers and caches the status.
func (b *RecorderBroadcaster) Broadcast(recorder *sspb.Recorder) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Cache the status and timestamp by recorder ID
	b.lastStatus[recorder.RecorderID] = recorder
	b.lastUpdate[recorder.RecorderID] = time.Now()

	for ch := range b.subscribers {
		select {
		case ch <- recorder:
		default:
			log.Warn().Str("recorder-id", recorder.RecorderID).Msg("Subscriber buffer full, dropping recorder update")
		}
	}
}

// GetCachedStatus returns the last known status for a specific recorder.
func (b *RecorderBroadcaster) GetCachedStatus(recorderID string) *sspb.Recorder {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastStatus[recorderID]
}

// GetAllCachedStatuses returns all cached recorder statuses.
func (b *RecorderBroadcaster) GetAllCachedStatuses() []*sspb.Recorder {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*sspb.Recorder, 0, len(b.lastStatus))
	for _, recorder := range b.lastStatus {
		result = append(result, recorder)
	}
	return result
}

// SubscriberCount returns the current number of subscribers.
func (b *RecorderBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// runTimeoutChecker periodically checks for stale recorders and broadcasts NO_SIGNAL.
func (b *RecorderBroadcaster) runTimeoutChecker(ctx context.Context) {
	ticker := time.NewTicker(recorderTimeoutCheckInterval)
	defer ticker.Stop()

	log.Info().
		Dur("timeout", b.statusTimeout).
		Dur("check-interval", recorderTimeoutCheckInterval).
		Msg("Recorder status timeout checker started")

	for {
		select {
		case <-b.stopTimeout:
			log.Info().Msg("Recorder status timeout checker stopped")
			return
		case <-ctx.Done():
			log.Info().Msg("Recorder status timeout checker stopped (context cancelled)")
			return
		case <-ticker.C:
			b.checkStaleRecorders()
		}
	}
}

// checkStaleRecorders checks for recorders that haven't sent updates recently
// and broadcasts NO_SIGNAL status for them.
func (b *RecorderBroadcaster) checkStaleRecorders() {
	b.mu.Lock()

	now := time.Now()
	var staleRecorders []*sspb.Recorder

	for recorderID, lastTime := range b.lastUpdate {
		if now.Sub(lastTime) > b.statusTimeout {
			cachedRecorder := b.lastStatus[recorderID]
			if cachedRecorder == nil {
				continue
			}

			// Check if already marked as NO_SIGNAL
			if status, ok := cachedRecorder.Info.(*sspb.Recorder_Status); ok {
				if status.Status.SignalStatus == cmpb.SignalStatus_NO_SIGNAL {
					continue // Already at NO_SIGNAL, skip
				}
			}

			log.Warn().
				Str("recorder-id", recorderID).
				Dur("since-last-update", now.Sub(lastTime)).
				Msg("Recorder status timed out, setting to NO_SIGNAL")

			// Create updated status with NO_SIGNAL
			staleRecorder := &sspb.Recorder{
				RecorderID:   cachedRecorder.RecorderID,
				RecorderName: cachedRecorder.RecorderName,
				Info: &sspb.Recorder_Status{
					Status: &cmpb.RecorderStatus{
						RecorderID:   cachedRecorder.RecorderID,
						RecorderName: cachedRecorder.RecorderName,
						SignalStatus: cmpb.SignalStatus_NO_SIGNAL,
						RmsPercent:   0.0,
						Clipping:     false,
					},
				},
			}

			// Update cache (but don't update lastUpdate - keep it stale)
			b.lastStatus[recorderID] = staleRecorder
			staleRecorders = append(staleRecorders, staleRecorder)
		}
	}

	b.mu.Unlock()

	// Broadcast stale recorders outside the lock
	for _, recorder := range staleRecorders {
		b.broadcastToSubscribers(recorder)
	}
}

// broadcastToSubscribers sends to all subscribers without updating cache.
func (b *RecorderBroadcaster) broadcastToSubscribers(recorder *sspb.Recorder) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- recorder:
		default:
			log.Warn().Str("recorder-id", recorder.RecorderID).Msg("Subscriber buffer full, dropping recorder update")
		}
	}
}
