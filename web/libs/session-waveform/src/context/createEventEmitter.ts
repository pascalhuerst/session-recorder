import type { Segment } from '../types';
import { createNanoEvents } from 'nanoevents';
import type { PeaksInstance } from 'peaks.js';

export interface Events {
  ready: (peaks: PeaksInstance) => void;
  playbackStarted: () => void;
  playbackPaused: () => void;
  playbackEnded: () => void;
  segmentAdded: (segment: Segment) => void;
  segmentUpdated: (segmentId: string, patch: Partial<Segment>) => void;
  segmentRemoved: (segmentId: string) => void;
  amplitudeScaleChanged: (scale: number) => void;
}

export const createEventEmitter = () => {
  return createNanoEvents<Events>();
};

export type EventEmitter = ReturnType<typeof createEventEmitter>;
