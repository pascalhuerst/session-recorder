import Peaks, {
  type PeaksInstance,
  type PeaksOptions,
  type SegmentMarker,
} from 'peaks.js';
import {
  computed,
  inject,
  provide,
  type Ref,
  ref,
  shallowRef,
  toValue,
  watch,
} from 'vue';
import type { MaybeReactive } from '../types';
import type { OverviewTheme, ZoomviewTheme } from './theme';
import { CustomSegmentMarker } from '../waveform/CustomSegmentMarker';

export type CreatePeaksProps = {
  overviewTheme: MaybeReactive<OverviewTheme>;
  zoomviewTheme: MaybeReactive<ZoomviewTheme>;
  waveformUrl?: MaybeReactive<string | undefined>;
};

export type PeaksContext = {
  peaks: Ref<PeaksInstance | undefined>;
  canvasElement: Ref<HTMLElement | undefined>;
  overviewElement: Ref<HTMLElement | undefined>;
  zoomviewElement: Ref<HTMLElement | undefined>;
  audioElement: Ref<HTMLAudioElement | undefined>;
};
const PeaksInjectionKey = Symbol();

export const createPeaksContext = (props: CreatePeaksProps): PeaksContext => {
  const peaks = shallowRef<PeaksInstance>();

  const canvasElement = ref<HTMLElement>();
  const overviewElement = ref<HTMLElement>();
  const zoomviewElement = ref<HTMLElement>();
  const audioElement = ref<HTMLAudioElement>();

  const options = computed(() => {
    if (!overviewElement.value || !audioElement.value) {
      return;
    }

    const waveformUrl = toValue(props.waveformUrl);

    return {
      overview: {
        container: overviewElement.value,
        ...toValue(props.overviewTheme),
      },
      zoomview: {
        container: zoomviewElement.value,
        ...toValue(props.zoomviewTheme),
      },
      mediaElement: audioElement.value,
      ...(waveformUrl
        ? {
            dataUri: {
              arraybuffer: waveformUrl,
            },
          }
        : {
            webAudio: {
              audioContext: new AudioContext(),
              multiChannel: false,
            },
          }),
      createSegmentMarker: (options) => {
        return new CustomSegmentMarker(options) as SegmentMarker;
      },
    } satisfies PeaksOptions;
  });

  watch(
    options,
    () => {
      if (options.value) {
        Peaks.init(options.value, function (err, peaksInstance) {
          if (err || !peaksInstance) {
            console.warn(err.message);
            return null;
          }

          peaks.value = peaksInstance;
        });
      }
    },
    { immediate: true }
  );

  provide(PeaksInjectionKey, {
    peaks,
    canvasElement,
    overviewElement,
    zoomviewElement,
    audioElement,
  });

  return {
    peaks,
    canvasElement,
    overviewElement,
    zoomviewElement,
    audioElement,
  };
};

export const usePeaksContext = () => {
  const context = inject(PeaksInjectionKey);
  if (!context) {
    throw new Error('You must create a peaks instance first');
  }
  return context as PeaksContext;
};
