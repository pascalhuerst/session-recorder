/**
 * Wraps a streaming gRPC subscription with automatic reconnection on error/end.
 *
 * When the stream drops (network hiccup, backend restart, Envoy timeout),
 * it retries with exponential backoff + jitter up to maxDelay.
 *
 * Usage:
 *   const { stop, reconnect } = reconnectingStream({
 *     name: 'recorders',
 *     connect: () => streamRecorders({ onMessage, onError, onEnd }),
 *   });
 *
 * The `connect` function must return a stop/abort function (as all stream
 * procedures already do). The reconnecting wrapper calls `connect` again
 * whenever the previous stream ends or errors.
 */

export type ReconnectingStreamOptions = {
  /** Human-readable name for logging */
  name: string;
  /** Factory that starts the stream. Must return a stop function. */
  connect: (handlers: {
    onEnd: () => void;
    onError: (error: Error) => void;
    /** Call this on each message to reset backoff after a successful reconnect */
    onMessage: () => void;
  }) => () => void;
  /** Called when scheduling a reconnect (for UI indicators) */
  onReconnecting?: (attempt: number) => void;
  /** Called when a reconnect succeeds (first message received after reconnect) */
  onReconnected?: () => void;
  /** Initial delay before first retry in ms (default: 1000) */
  initialDelay?: number;
  /** Maximum delay between retries in ms (default: 30000) */
  maxDelay?: number;
};

const DEFAULT_INITIAL_DELAY = 1000;
const DEFAULT_MAX_DELAY = 30000;

export type ReconnectingStreamHandle = {
  stop: () => void;
  reconnect: () => void;
};

export const reconnectingStream = (
  options: ReconnectingStreamOptions
): ReconnectingStreamHandle => {
  const {
    name,
    connect,
    onReconnecting,
    onReconnected,
    initialDelay = DEFAULT_INITIAL_DELAY,
    maxDelay = DEFAULT_MAX_DELAY,
  } = options;

  let stopped = false;
  let currentStop: (() => void) | null = null;
  let retryTimeout: ReturnType<typeof setTimeout> | null = null;
  let consecutiveFailures = 0;
  let receivedMessage = false;

  const scheduleReconnect = () => {
    if (stopped) return;

    consecutiveFailures++;

    const baseDelay = Math.min(
      initialDelay * Math.pow(2, consecutiveFailures - 1),
      maxDelay
    );
    // Add 0-20% jitter to prevent thundering herd
    const jitter = baseDelay * 0.2 * Math.random();
    const delay = baseDelay + jitter;

    if (import.meta.env.DEV) {
      console.log(
        `[${name}] Stream ended, reconnecting in ${Math.round(delay)}ms (attempt ${consecutiveFailures})`
      );
    }

    onReconnecting?.(consecutiveFailures);

    retryTimeout = setTimeout(() => {
      if (stopped) return;
      startStream();
    }, delay);
  };

  const handlers = {
    onEnd: () => {
      if (!stopped) scheduleReconnect();
    },
    onError: (error: Error) => {
      console.error(`[${name}] Stream error:`, error);
      if (!stopped) scheduleReconnect();
    },
    onMessage: () => {
      if (!receivedMessage) {
        receivedMessage = true;
        if (consecutiveFailures > 0) {
          if (import.meta.env.DEV) console.log(`[${name}] Reconnected successfully`);
          consecutiveFailures = 0;
          onReconnected?.();
        }
      }
    },
  };

  const startStream = () => {
    if (stopped) return;
    receivedMessage = false;
    currentStop = connect(handlers);
  };

  const reconnect = () => {
    if (stopped) return;

    // Cancel any pending retry
    if (retryTimeout !== null) {
      clearTimeout(retryTimeout);
      retryTimeout = null;
    }

    // Stop current stream
    if (currentStop) {
      currentStop();
      currentStop = null;
    }

    // Reset backoff and reconnect immediately
    consecutiveFailures = 0;
    startStream();
  };

  // Start initial connection
  startStream();

  return {
    stop: () => {
      stopped = true;
      if (retryTimeout !== null) {
        clearTimeout(retryTimeout);
        retryTimeout = null;
      }
      if (currentStop) {
        currentStop();
        currentStop = null;
      }
    },
    reconnect,
  };
};
