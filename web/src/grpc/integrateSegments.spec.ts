import { describe, it, expect, vi } from 'vitest';
import { createNanoEvents } from 'nanoevents';

/**
 * We test integrateSegments' cleanup pattern by mocking the emitters
 * and the gRPC procedure modules. The function subscribes to events
 * on ctx.eventEmitter and ctx.commandEmitter, and returns a cleanup
 * function that unbinds all listeners.
 */

// Mock all gRPC procedures so they don't make real calls
vi.mock('./procedures/createSegment', () => ({
  createSegment: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('./procedures/updateSegment', () => ({
  updateSegment: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('./procedures/deleteSegment', () => ({
  deleteSegment: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('./procedures/renderSegment', () => ({
  renderSegment: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('../services/Toaster/ToastService', () => ({
  toastService: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

import type { PeaksContext } from '@session-recorder/session-waveform';
import type { Session } from '../types';
import { integrateSegments } from './integrateSegments';
import { createSegment } from './procedures/createSegment';
import { deleteSegment } from './procedures/deleteSegment';
import { renderSegment } from './procedures/renderSegment';
import { toastService } from '../services/Toaster/ToastService';

type EventEmitter = {
  on: (event: string, cb: (...args: unknown[]) => void) => () => void;
  emit: (event: string, ...args: unknown[]) => void;
  events: Record<string, ((...args: unknown[]) => void)[]>;
};

function createMockCtx() {
  const eventEmitter = createNanoEvents() as unknown as EventEmitter;
  const commandEmitter = createNanoEvents() as unknown as EventEmitter;
  return { eventEmitter, commandEmitter } as unknown as PeaksContext;
}

function createMockSession() {
  return { id: 'session-1' } as unknown as Session;
}

describe('integrateSegments', () => {
  it('registers event listeners on both emitters', () => {
    const ctx = createMockCtx();
    const onSpy = vi.spyOn(ctx.eventEmitter, 'on');
    const cmdSpy = vi.spyOn(ctx.commandEmitter, 'on');

    const cleanup = integrateSegments(createMockSession(), 'rec-1', ctx);

    // 4 event listeners + 1 command listener
    expect(onSpy).toHaveBeenCalledTimes(4);
    expect(cmdSpy).toHaveBeenCalledTimes(1);

    expect(onSpy).toHaveBeenCalledWith('segmentAdded', expect.any(Function));
    expect(onSpy).toHaveBeenCalledWith('segmentUpdated', expect.any(Function));
    expect(onSpy).toHaveBeenCalledWith('segmentRemoved', expect.any(Function));
    expect(onSpy).toHaveBeenCalledWith('segmentsBulkDeleted', expect.any(Function));
    expect(cmdSpy).toHaveBeenCalledWith('renderSegment', expect.any(Function));

    cleanup();
  });

  it('returns a cleanup function that unbinds all listeners', () => {
    const ctx = createMockCtx();
    const cleanup = integrateSegments(createMockSession(), 'rec-1', ctx);

    // Before cleanup: listeners are registered
    expect(Object.keys(ctx.eventEmitter.events).length).toBeGreaterThan(0);

    cleanup();

    // After cleanup: all listener arrays should be empty
    for (const event of Object.keys(ctx.eventEmitter.events)) {
      expect(ctx.eventEmitter.events[event]).toHaveLength(0);
    }
    for (const event of Object.keys(ctx.commandEmitter.events)) {
      expect(ctx.commandEmitter.events[event]).toHaveLength(0);
    }
  });

  it('after cleanup, emitting events does not trigger handlers', async () => {
    const ctx = createMockCtx();
    const cleanup = integrateSegments(createMockSession(), 'rec-1', ctx);

    cleanup();

    // Reset mocks to track only post-cleanup calls
    vi.mocked(createSegment).mockClear();
    vi.mocked(deleteSegment).mockClear();
    vi.mocked(renderSegment).mockClear();

    // Emit events after cleanup
    ctx.eventEmitter.emit('segmentAdded', {
      id: 'seg-1',
      startTime: 0,
      endTime: 1,
      labelText: 'test',
    });
    ctx.eventEmitter.emit('segmentRemoved', 'seg-1');
    ctx.commandEmitter.emit('renderSegment', 'seg-1');

    // Allow any pending microtasks
    await Promise.resolve();

    expect(createSegment).not.toHaveBeenCalled();
    expect(deleteSegment).not.toHaveBeenCalled();
    expect(renderSegment).not.toHaveBeenCalled();
  });

  it('calls createSegment when segmentAdded is emitted', async () => {
    const ctx = createMockCtx();
    const cleanup = integrateSegments(createMockSession(), 'rec-1', ctx);

    ctx.eventEmitter.emit('segmentAdded', {
      id: 'seg-1',
      startTime: 1.5,
      endTime: 3.7,
      labelText: 'Test Segment',
    });

    await Promise.resolve();

    expect(createSegment).toHaveBeenCalledWith({
      recorderId: 'rec-1',
      sessionId: 'session-1',
      segmentId: 'seg-1',
      segment: expect.objectContaining({
        name: 'Test Segment',
      }),
    });

    cleanup();
  });

  it('shows toast on segmentsBulkDeleted', () => {
    const ctx = createMockCtx();
    const cleanup = integrateSegments(createMockSession(), 'rec-1', ctx);

    ctx.eventEmitter.emit('segmentsBulkDeleted', 3);

    expect(toastService.success).toHaveBeenCalledWith('3 segments deleted');

    cleanup();
  });
});
