import type { Segment } from '../types';
import { createNanoEvents } from 'nanoevents';

export interface Commands {
  play: () => void;
  pause: () => void;
  seek: (time: number) => void;
  addSegment: () => void;
  updateSegment: (segmentId: string, patch: Partial<Segment>) => void;
  removeSegment: (segmentId: string) => void;
  setAmplitudeScale: (scale: number) => void;
  setZoomLevel: (seconds: number) => void;
}

export const createCommandEmitter = () => {
  return createNanoEvents<Commands>();
};

export type CommandEmitter = ReturnType<typeof createCommandEmitter>;
