import type { Segment } from '../types';
import { createNanoEvents } from 'nanoevents';

export interface Events {
  'ui.segments.add': (segment: Segment) => void;
  'ui.segment.selectById': (segmentId: string) => void;
  'peaks.segment.updateById': (
    segmentId: string,
    patch: Partial<Segment>
  ) => void;
  'peaks.segment.removeById': (segmentId: string) => void;
}

export const createEventEmitter = () => {
  return createNanoEvents<Events>();
};

export type EventEmitter = ReturnType<typeof createEventEmitter>;
