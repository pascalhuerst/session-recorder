// Package broadcast provides fan-out message distribution to multiple subscribers.
//
// # Broadcaster
//
// Broadcaster[T] sends messages to all subscribers via buffered channels.
// Production uses buffer size 10. Messages are sent non-blocking — if a
// subscriber's buffer is full, the message is dropped for that subscriber only.
//
// Thread safety:
//   - Both Broadcast and Unsubscribe acquire the same write lock, so there is
//     no risk of sending to a closed channel.
//   - Subscribe and Unsubscribe are safe to call from any goroutine.
//   - Unsubscribe is idempotent.
//
// # Shutdown
//
// Call Close to close all subscriber channels at once. Goroutines blocked on
// channel reads will receive the zero value and ok=false, allowing them to
// exit cleanly. After Close, the subscriber map is empty.
//
// # RecorderBroadcaster
//
// RecorderBroadcaster wraps Broadcaster with recorder-specific timeout logic.
// It periodically checks for stale recorder entries and removes them.
// Call Stop to signal the timeout checker to exit.
package broadcast
