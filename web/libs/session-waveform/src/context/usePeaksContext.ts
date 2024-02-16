import { type PeaksInstance } from 'peaks.js';
import { inject, provide, type Ref } from 'vue';
import { createZoomControls } from './createZoomControls';
import { createAmplitudeControls } from './createAmplitudeControls';
import { createPlayerControls } from './createPlayerControls';
import type { CreatePeaksCanvasProps } from './createPeaksCanvas';
import { createPeaksCanvas } from './createPeaksCanvas';
import type { AudioSourceUrl } from '../types';
import { type Permissions } from '../types';
import {
  createSegmentControls,
  type CreateSegmentControlsProps,
} from './createSegmentControls';
import { createEventEmitter } from './createEventEmitter';
import type { createCommandEmitter } from './createCommandEmitter';

export type CreatePeaksProps = CreatePeaksCanvasProps &
  CreateSegmentControlsProps & {
    audioUrls: Ref<Array<AudioSourceUrl>>;
  };

export type PeaksContext = {
  peaks: Ref<PeaksInstance | undefined>;
  audioUrls: Ref<Array<AudioSourceUrl>>;
  waveformUrl?: Ref<string | undefined>;
  layout: {
    canvasElement: Ref<HTMLElement | undefined>;
    overviewElement: Ref<HTMLElement | undefined>;
    zoomviewElement: Ref<HTMLElement | undefined>;
    audioElement: Ref<HTMLAudioElement | undefined>;
  };
  zoom: ReturnType<typeof createZoomControls>;
  amplitude: ReturnType<typeof createAmplitudeControls>;
  player: ReturnType<typeof createPlayerControls>;
  segments: ReturnType<typeof createSegmentControls>;
  eventEmitter: ReturnType<typeof createEventEmitter>;
  commandEmitter: ReturnType<typeof createCommandEmitter>;
  permissions: Ref<Permissions>;
};

const PeaksInjectionKey = Symbol();

export const createPeaksContext = (props: CreatePeaksProps): PeaksContext => {
  const canvas = createPeaksCanvas(props);

  const context = {
    ...canvas,
    audioUrls: props.audioUrls,
    waveformUrl: props.waveformUrl,
    zoom: createZoomControls(canvas),
    amplitude: createAmplitudeControls(canvas),
    player: createPlayerControls(canvas),
    segments: createSegmentControls({
      ...canvas,
      segments: props.segments,
      permissions: props.permissions,
    }),
    permissions: props.permissions,
  } satisfies PeaksContext;

  provide(PeaksInjectionKey, context);
  return context;
};

export const usePeaksContext = () => {
  const context = inject(PeaksInjectionKey);
  if (!context) {
    throw new Error('You must create a peaks context first');
  }
  return context as PeaksContext;
};
