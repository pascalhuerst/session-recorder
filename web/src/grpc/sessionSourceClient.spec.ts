import { describe, it, expect } from 'vitest';
import { SessionSourceClient } from '@session-recorder/protocols/ts/sessionsource.client';

/**
 * Guards against stale protobuf stubs. If a new RPC is added to the .proto
 * file but `make ts` hasn't been re-run, the generated client will be missing
 * the method and this test will fail.
 */
describe('SessionSourceClient generated stubs', () => {
  const expectedMethods = [
    'streamRecorders',
    'streamSessions',
    'streamSessionAudio',
    'streamWaveformPeaks',
    'setKeepSession',
    'deleteSession',
    'setName',
    'createSegment',
    'deleteSegment',
    'renderSegment',
    'updateSegment',
    'cutSession',
    'retryRenderSession',
    'shareSession',
    'shareSegment',
  ] as const;

  it.each(expectedMethods)('has %s method', (method) => {
    expect(SessionSourceClient.prototype).toHaveProperty(method);
    expect(typeof SessionSourceClient.prototype[method as keyof SessionSourceClient]).toBe(
      'function'
    );
  });
});
