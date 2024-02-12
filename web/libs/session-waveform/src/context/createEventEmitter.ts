import type { Segment } from '../types';
import { createNanoEvents } from 'nanoevents';

export interface Events {
  segmentAdded: (segment: Segment) => void;
  segmentUpdated: (segmentId: string, patch: Partial<Segment>) => void;
  segmentRemoved: (segmentId: string) => void;
  segmentSelected: (segmentId: string) => void;
}

export const createEventEmitter = () => {
  return createNanoEvents<Events>();
};

export type EventEmitter = ReturnType<typeof createEventEmitter>;
