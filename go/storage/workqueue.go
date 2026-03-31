package storage

import (
	"context"

	"github.com/alitto/pond/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	// DefaultMaxRenderWorkers is the default number of concurrent render workers.
	DefaultMaxRenderWorkers = 3
)

// workQueue manages background rendering work using a bounded worker pool.
type workQueue struct {
	pool   pond.Pool
	cancel context.CancelFunc
	ctx    context.Context
}

// newWorkQueue creates a new work queue with the given max concurrency.
func newWorkQueue(maxWorkers int) *workQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &workQueue{
		pool:   pond.NewPool(maxWorkers),
		cancel: cancel,
		ctx:    ctx,
	}
}

// submitSessionRender enqueues a session render job.
// The chunk parameter is optional (nil when re-rendering from existing raw data).
// Delegates to closeSessionAsync which handles flush, isSessionClosed check, and rendering.
func (wq *workQueue) submitSessionRender(
	ctx context.Context,
	m *Minio,
	recorderID, sessionID uuid.UUID,
	chunk *minioChunk,
) {
	wq.pool.Submit(func() {
		// Use the work queue's context so renders abort on shutdown
		jobCtx := wq.ctx

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Work queue: starting session render")

		if err := m.closeSessionAsync(jobCtx, recorderID, sessionID, chunk); err != nil {
			log.Err(err).
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Msg("Work queue: session render failed")
			return
		}

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Msg("Work queue: session render completed")
	})
}

// submitSegmentRender enqueues a segment render job.
func (wq *workQueue) submitSegmentRender(
	ctx context.Context,
	m *Minio,
	recorderID, sessionID, segmentID uuid.UUID,
) {
	wq.pool.Submit(func() {
		jobCtx := wq.ctx

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Stringer("segment-id", segmentID).
			Msg("Work queue: starting segment render")

		if err := m.renderSegmentInternal(jobCtx, recorderID, sessionID, segmentID); err != nil {
			log.Err(err).
				Stringer("recorder-id", recorderID).
				Stringer("session-id", sessionID).
				Stringer("segment-id", segmentID).
				Msg("Work queue: segment render failed")
			return
		}

		log.Info().
			Stringer("recorder-id", recorderID).
			Stringer("session-id", sessionID).
			Stringer("segment-id", segmentID).
			Msg("Work queue: segment render completed")
	})
}

// stop cancels pending work and shuts down the work queue without blocking.
func (wq *workQueue) stop() {
	wq.cancel()
	wq.pool.Stop()
}

// stopAndWait gracefully shuts down, waiting for in-flight jobs to complete.
func (wq *workQueue) stopAndWait() {
	wq.pool.StopAndWait()
}
