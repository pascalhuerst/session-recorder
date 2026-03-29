import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { reconnectingStream } from './reconnectingStream';

describe('reconnectingStream', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('calls connect on creation', () => {
    const connect = vi.fn(() => vi.fn());
    const { stop } = reconnectingStream({ name: 'test', connect });
    expect(connect).toHaveBeenCalledTimes(1);
    stop();
  });

  it('reconnects on stream end', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const stopFn = vi.fn();
    const connect = vi.fn((h) => {
      handlers = h;
      return stopFn;
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
    });

    expect(connect).toHaveBeenCalledTimes(1);

    // Simulate stream ending
    handlers!.onEnd();

    // Should not reconnect immediately
    expect(connect).toHaveBeenCalledTimes(1);

    // Advance past the initial delay + jitter
    await vi.advanceTimersByTimeAsync(200);
    expect(connect).toHaveBeenCalledTimes(2);

    stop();
  });

  it('reconnects on stream error', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const connect = vi.fn((h) => {
      handlers = h;
      return vi.fn();
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
    });

    // Simulate error
    handlers!.onError(new Error('connection lost'));

    await vi.advanceTimersByTimeAsync(200);
    expect(connect).toHaveBeenCalledTimes(2);

    stop();
  });

  it('uses exponential backoff', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const connect = vi.fn((h) => {
      handlers = h;
      return vi.fn();
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
      maxDelay: 10000,
    });

    // First failure: ~100ms delay
    handlers!.onEnd();
    await vi.advanceTimersByTimeAsync(50);
    expect(connect).toHaveBeenCalledTimes(1); // not yet
    await vi.advanceTimersByTimeAsync(100);
    expect(connect).toHaveBeenCalledTimes(2);

    // Second failure: ~200ms delay
    handlers!.onEnd();
    await vi.advanceTimersByTimeAsync(150);
    expect(connect).toHaveBeenCalledTimes(2); // not yet
    await vi.advanceTimersByTimeAsync(200);
    expect(connect).toHaveBeenCalledTimes(3);

    stop();
  });

  it('caps backoff at maxDelay', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const connect = vi.fn((h) => {
      handlers = h;
      return vi.fn();
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 1000,
      maxDelay: 2000,
    });

    // Fail several times
    for (let i = 0; i < 5; i++) {
      handlers!.onEnd();
      // Max delay is 2000ms + 20% jitter = 2400ms max
      await vi.advanceTimersByTimeAsync(2500);
    }

    // Should have reconnected each time (6 total: initial + 5 reconnects)
    expect(connect).toHaveBeenCalledTimes(6);

    stop();
  });

  it('resets backoff after successful message', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const connect = vi.fn((h) => {
      handlers = h;
      return vi.fn();
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
    });

    // Fail twice to increase backoff
    handlers!.onEnd();
    await vi.advanceTimersByTimeAsync(200);
    handlers!.onEnd();
    await vi.advanceTimersByTimeAsync(400);
    expect(connect).toHaveBeenCalledTimes(3);

    // Receive a message (resets backoff)
    handlers!.onMessage();

    // Fail again - should use initial delay, not accumulated
    handlers!.onEnd();
    await vi.advanceTimersByTimeAsync(50);
    expect(connect).toHaveBeenCalledTimes(3); // not yet
    await vi.advanceTimersByTimeAsync(100);
    expect(connect).toHaveBeenCalledTimes(4); // reconnected with short delay

    stop();
  });

  it('does not reconnect after stop()', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const connect = vi.fn((h) => {
      handlers = h;
      return vi.fn();
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
    });

    // Trigger reconnect
    handlers!.onEnd();

    // Stop before reconnect fires
    stop();

    await vi.advanceTimersByTimeAsync(500);
    // Should not have reconnected
    expect(connect).toHaveBeenCalledTimes(1);
  });

  it('calls onReconnecting and onReconnected callbacks', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const connect = vi.fn((h) => {
      handlers = h;
      return vi.fn();
    });

    const onReconnecting = vi.fn();
    const onReconnected = vi.fn();

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      onReconnecting,
      onReconnected,
      initialDelay: 100,
    });

    // Should not be called on initial connect
    expect(onReconnecting).not.toHaveBeenCalled();
    expect(onReconnected).not.toHaveBeenCalled();

    // Trigger disconnect
    handlers!.onEnd();
    expect(onReconnecting).toHaveBeenCalledWith(1);

    // Reconnect
    await vi.advanceTimersByTimeAsync(200);
    expect(connect).toHaveBeenCalledTimes(2);

    // Receive first message after reconnect
    handlers!.onMessage();
    expect(onReconnected).toHaveBeenCalledTimes(1);

    stop();
  });

  it('calls the stop function from connect on cleanup', () => {
    const innerStop = vi.fn();
    const connect = vi.fn(() => innerStop);

    const { stop } = reconnectingStream({ name: 'test', connect });

    stop();
    expect(innerStop).toHaveBeenCalledTimes(1);
  });

  it('calls the stop function from previous connect before reconnecting', async () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const innerStop1 = vi.fn();
    const innerStop2 = vi.fn();
    let callCount = 0;

    const connect = vi.fn((h) => {
      handlers = h;
      callCount++;
      return callCount === 1 ? innerStop1 : innerStop2;
    });

    const { stop } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
    });

    // Trigger reconnect
    handlers!.onEnd();
    await vi.advanceTimersByTimeAsync(200);

    // innerStop1 was NOT called (the stream ended naturally, no need to abort)
    // The new stream (innerStop2) is now active
    expect(connect).toHaveBeenCalledTimes(2);

    // Stopping the wrapper should stop the current stream
    stop();
    expect(innerStop2).toHaveBeenCalledTimes(1);
  });

  it('reconnect() stops current stream and reconnects immediately', () => {
    let handlers: {
      onEnd: () => void;
      onError: (error: Error) => void;
      onMessage: () => void;
    } | null = null;

    const innerStop = vi.fn();
    const connect = vi.fn((h) => {
      handlers = h;
      return innerStop;
    });

    const { stop, reconnect } = reconnectingStream({
      name: 'test',
      connect,
      initialDelay: 100,
    });

    expect(connect).toHaveBeenCalledTimes(1);

    // Force reconnect
    reconnect();

    // Should have stopped the old stream and connected again immediately
    expect(innerStop).toHaveBeenCalledTimes(1);
    expect(connect).toHaveBeenCalledTimes(2);

    stop();
  });

  it('reconnect() does nothing after stop()', () => {
    const connect = vi.fn(() => vi.fn());

    const { stop, reconnect } = reconnectingStream({
      name: 'test',
      connect,
    });

    stop();
    reconnect();

    // Should not have reconnected
    expect(connect).toHaveBeenCalledTimes(1);
  });
});
