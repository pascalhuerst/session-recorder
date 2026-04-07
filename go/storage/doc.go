// Package storage manages session recording state and S3 persistence.
//
// # Session State Machine
//
// Sessions follow a finite state machine with these transitions:
//
//	UNKNOWN ──► RECORDING ──► PROCESSING ──► FINISHED (terminal)
//	                ▲              │
//	                │ resume       │ RenderFailure
//	                │              ▼
//	                └────────── ERROR
//	                  RetryRender    │
//	                                ▼
//	                           PROCESSING (retry)
//
//   - UNKNOWN → RECORDING: First SafeChunks call creates the session and starts streaming encoders.
//   - RECORDING → PROCESSING: Triggered by explicit close, signal loss, session switch, or timeout.
//   - PROCESSING → FINISHED: Streaming or fallback render completes successfully.
//   - PROCESSING → ERROR: Render fails. Error message stored in session metadata.
//   - ERROR → PROCESSING: Manual retry via RetryRenderSession.
//   - PROCESSING → RECORDING: Resume — when chunks arrive for a session in PROCESSING,
//     the FSM is replaced and the session returns to RECORDING.
//   - FINISHED is terminal — no transitions out.
//
// Each session has a dedicated [stateless.StateMachine] created via newSessionStateMachine.
// Machines are stored in Minio.sessionMachines, keyed by sessionID.
//
// # Segment State Machine
//
//	UNKNOWN → QUEUED → RENDERING → FINISHED → QUEUED (re-render)
//	                      │
//	                      ▼
//	                    ERROR → QUEUED (retry)
//
// Segment transitions are validated by validateSegmentTransition (no FSM object).
//
// # Recording Flow
//
//  1. SafeChunks receives audio samples and buffers them in a bytes.Buffer.
//  2. Every ~1 second the buffer is flushed to 4 streaming encode pipelines
//     (raw PCM, FLAC, WAV, waveform) running as goroutines connected via io.Pipe.
//  3. Each pipeline uploads directly to S3 via PutObject.
//  4. On session close, the remaining buffer is flushed, encoders are closed,
//     and the FSM transitions to PROCESSING → FINISHED.
//
// # Shutdown
//
// Minio.Stop performs a graceful shutdown:
//  1. Closes stopTimeout channel — stops the session timeout checker.
//  2. Calls renderQueue.stopAndWait — waits for all in-flight render jobs to complete.
//     This prevents sessions from getting stuck in PROCESSING permanently.
//  3. Cancels shutdownCtx — signals streaming upload goroutines to abort.
//
// The work queue context (wq.ctx) is only cancelled after the pool has fully drained,
// so in-flight jobs can complete their S3 uploads before the context is invalidated.
//
// # Concurrency
//
// Two mutexes protect shared state:
//
//   - dataLock: protects system, chunks, lastChunkTime, deletedSessions.
//   - machineLock: protects sessionMachines map lookups.
//
// Key rules:
//   - Event emission (EmitSessionStateChanged, EmitSegmentStateChanged) always
//     occurs OUTSIDE dataLock to prevent deadlocks with listeners that call
//     GetSession/GetSessions.
//   - machineLock is released before calling sm.FireCtx, so the onSessionTransition
//     callback can acquire dataLock without creating a lock cycle.
//   - onSessionTransition validates that the in-memory session state matches the
//     FSM's source state to guard against stale SM references (e.g., after resume).
//
// # Tombstone Cleanup
//
// Deleted sessions are tracked in deletedSessions with timestamps to prevent
// resurrection by stale render callbacks. Entries are cleaned up:
//   - Amortized during DeleteSession calls.
//   - Periodically by sweepDeletedSessions in the session timeout checker.
//
// Entries older than 1 minute are removed.
package storage
