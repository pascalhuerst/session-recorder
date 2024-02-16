import { computed, type Ref, ref, shallowRef, toValue, watch } from 'vue';
import { CustomSegmentMarker } from '../elements/CustomSegmentMarker';
import type { PeaksInstance, PeaksOptions, SegmentMarker } from 'peaks.js';
import Peaks from 'peaks.js';
import type { OverviewTheme, ZoomviewTheme } from './theme';
import { createEventEmitter } from './createEventEmitter';
import { createCommandEmitter } from './createCommandEmitter';

export type CreatePeaksCanvasProps = {
  overviewTheme: Ref<OverviewTheme>;
  zoomviewTheme: Ref<ZoomviewTheme>;
  waveformUrl?: Ref<string | undefined>;
};

export const createPeaksCanvas = (props: CreatePeaksCanvasProps) => {
  const eventEmitter = createEventEmitter();
  const commandEmitter = createCommandEmitter();

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
        return new CustomSegmentMarker(options, {
          eventEmitter,
        }) as SegmentMarker;
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
          eventEmitter.emit('ready', peaksInstance);
        });
      }
    },
    { immediate: true }
  );

  return {
    peaks,
    layout: {
      canvasElement,
      overviewElement,
      zoomviewElement,
      audioElement,
    },
    eventEmitter,
    commandEmitter,
  };
};
