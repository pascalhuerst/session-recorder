import { type PeaksInstance } from "peaks.js";
import { inject, provide, type Ref } from "vue";
import { installZoomControls } from "./installZoomControls";
import { installAmplitudeControls } from "./installAmplitudeControls";
import { installPlayerControls } from "./installPlayerControls";
import type { CreatePeaksCanvasProps } from "./createPeaksModule";
import { createPeaksModule } from "./createPeaksModule";
import type { AudioSourceUrl } from "../types";
import { type Permissions } from "../types";
import { createSegmentControls, type CreateSegmentControlsProps } from "./createSegmentControls";
import { createEventEmitter } from "../lib/app/createEventEmitter";
import type { createCommandEmitter } from "../lib/app/createCommandEmitter";
import { createContextStore, defaultState } from "../lib/app/createContextStore";
import { merge } from "lodash.merge";
import { defineContext } from "../lib/app/defineContext";

export type CreatePeaksProps = CreatePeaksCanvasProps &
  CreateSegmentControlsProps & {
  audioUrls: Ref<Array<AudioSourceUrl>>;
};

export type PeaksContext = {
  state: ReturnType<typeof createContextStore>;
  peaks: Ref<PeaksInstance | undefined>;
  layout: {
    canvasElement: Ref<HTMLElement | undefined>;
    overviewElement: Ref<HTMLElement | undefined>;
    zoomviewElement: Ref<HTMLElement | undefined>;
    audioElement: Ref<HTMLAudioElement | undefined>;
  };
  player: ReturnType<typeof installPlayerControls>;
  segments: ReturnType<typeof createSegmentControls>;
  eventEmitter: ReturnType<typeof createEventEmitter>;
  commandEmitter: ReturnType<typeof createCommandEmitter>;
  permissions: Ref<Permissions>;
};

const PeaksInjectionKey = Symbol();

export const createPeaksContext = (props: CreatePeaksProps): PeaksContext => {
  const module = createPeaksModule({
    ...props,
    initialState: merge({}, defaultState, props.initialState)
  });

  installZoomControls(module);
  installAmplitudeControls(module);
  installPlayerControls(module);
  createSegmentControls(module);

  const context = {
    ...module,
    segments:,
    permissions: props.permissions
  } satisfies PeaksContext;

  provide(PeaksInjectionKey, context);
  return context;
};

export const usePeaksContext = () => {
  const context = inject(PeaksInjectionKey);
  if (!context) {
    throw new Error("You must create a peaks context first");
  }
  return context as PeaksContext;
};
