import type { PeaksInstance } from 'peaks.js';
import { createCommandEmitter } from '../lib/app/createCommandEmitter';
import { createContextStore } from '../lib/app/createContextStore';
import { z } from 'zod';
import { createEventEmitter } from '../lib/app/createEventEmitter';
import { peaksModuleSchema, type Segment } from './models/state';

export interface Events {
  mounted: () => void;
  ready: (peaks: PeaksInstance) => void;
  clickOutsideCanvas: () => void;
  playbackStarted: () => void;
  playbackPaused: () => void;
  playbackEnded: () => void;
  segmentAdded: (segment: Segment) => void;
  segmentUpdated: (
    segmentId: string,
    patch: Partial<Segment>,
    segment: Segment
  ) => void;
  segmentRemoved: (segmentId: string) => void;
  amplitudeScaleChanged: (scale: number) => void;
  zoomLevelChanged: (seconds: number) => void;
}

export interface Commands {
  mount: (elements: {
    overview: HTMLElement;
    zoomview: HTMLElement;
    audio: HTMLElement;
  }) => void;
  play: () => void;
  pause: () => void;
  seek: (time: number) => void;
  addSegment: () => void;
  updateSegment: (segmentId: string, patch: Partial<Segment>) => void;
  removeSegment: (segmentId: string) => void;
  renderSegment: (segmentId: string) => void;
  setAmplitudeScale: (scale: number) => void;
  setZoomLevel: (seconds: number) => void;
}

export const createPeaksModule = (props: {
  initialState: z.input<typeof peaksModuleSchema>;
}) => {
  const state = createContextStore({
    schema: peaksModuleSchema,
    initialState: props.initialState,
  });
  const eventEmitter = createEventEmitter<Events>();
  const commandEmitter = createCommandEmitter<Commands>();

  return {
    state,
    eventEmitter,
    commandEmitter,
  };
};
