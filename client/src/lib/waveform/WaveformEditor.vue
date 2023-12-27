<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import VirtualizedItem from "../disclosure/VirtualizedItem.vue";
import Peaks, { type PeaksInstance, type PeaksOptions } from "peaks.js";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { onClickOutside } from "@vueuse/core";
import Button from "../controls/Button.vue";
import TextInput from "../forms/TextInput.vue";
import uuid from "uuidv4";
import { CustomSegmentMarker } from "./CustomeSegmentMarker.ts";
import Marker from "../controls/Marker.vue";
import { parsePlayTime } from "../../utils/parsePlayTime.ts";

type Segment = {
  startTime: number;
  endTime: number;
  editable?: boolean;
  color?: string;
  labelText?: string;
  id?: string;
  [key: string]: any
}

const props = withDefaults(defineProps<{
  waveformUrl?: string
  audioUrls: Array<{ src: string, type: string }>
  height?: number,
  segments?: Map<string, Segment & { id: string }>
}>(), {
  height: 200
});

const canvasEl = ref<HTMLElement>();
const containerEl = ref<HTMLElement>();
const zoomViewEl = ref<HTMLElement>();
const mediaEl = ref<HTMLAudioElement>();
const peaksInstanceRef = shallowRef<PeaksInstance>();

const zoomLevel = ref(300);
const zoomviewAmpZoom = ref(0.6);
const zoomStep = ref(100);
const zoomAmpStep = ref(0.1);

watch([containerEl, zoomViewEl, mediaEl, props], () => {
  if (containerEl.value && mediaEl.value) {
    const options = {
      overview: {
        container: containerEl.value,
        enablePoints: false,
        enableSegments: true,
        playheadPadding: 0,
        playheadColor: "red",
        playedWaveformColor: "#6b46c1",
        showPlayheadTime: true,
        playheadTextColor: "#6b46c1",
        playheadBackgroundColor: "rgb(107,70,193,0.1)",
        waveformColor: "#767c89"
      },
      zoomview: {
        container: zoomViewEl.value,
        enablePoints: false,
        enableSegments: true,
        playheadPadding: 16,
        playheadColor: "#6b46c1",
        waveformColor: "#a5aab4",
        showPlayheadTime: true,
        playedWaveformColor: "#bdafe3",
        playheadTextColor: "#6b46c1",
        segmentOptions: {
          startMarkerColor: "#d53f8c",
          endMarkerColor: "#d53f8c"
        }
      },
      mediaElement: mediaEl.value,
      ...props.waveformUrl ? {
        dataUri: {
          arraybuffer: props.waveformUrl
        }
      } : {
        webAudio: {
          audioContext: new AudioContext(),
          multiChannel: false
        }
      },
      createSegmentMarker(options) {
        return new CustomSegmentMarker(options);
      }
    } satisfies PeaksOptions;

    Peaks.init(options, function(err, peaksInstance) {
      if (err || !peaksInstance) {
        console.warn(err.message);
        return null;
      }

      peaksInstanceRef.value = peaksInstance;

    });
  }
});

const segments = ref(new Map(props.segments));

watch([segments, peaksInstanceRef], () => {
  if (peaksInstanceRef.value && segments.value.size) {
    try {
      segments.value.forEach((segment) => {
        peaksInstanceRef.value?.segments.removeById(segment.id);
        peaksInstanceRef.value?.segments.add(
          segment
        );
      });

    } catch (err) {
      console.log(err);
    }
    // const overview = peaksInstanceRef.value?.views.getView("overview");
    // overview?.setPlayedWaveformColor("#767c89");
    //
    // const zoomview = peaksInstanceRef.value?.views.getView("zoomview");
    // zoomview?.setPlayedWaveformColor("#767c89");
  }
}, {
  immediate: true,
  deep: true
});

const intToChar = (int: number) => {
  const start = "a".charCodeAt(0);
  return String.fromCharCode(start + int).toUpperCase();
};
const addSegment = () => {
  const segmentId = uuid();
  const size = segments.value.size * 2;
  segments.value.set(
    segmentId,
    {
      id: segmentId,
      startTime: Number(peaksInstanceRef.value?.player.getCurrentTime()),
      endTime: Number(peaksInstanceRef.value?.player.getCurrentTime()) + 5,
      color: "#ed64a6",
      labelText: "0 to 10.5 seconds non-editable demo segment",
      editable: true,
      startIndex: intToChar(size),
      endIndex: intToChar(size + 1)
    }
  );
};

const isPlaying = ref(false);

const onPlay = () => {
  const player = peaksInstanceRef.value?.player as PeaksOptions["player"];
  if (!player) {
    return;
  }

  if (isPlaying.value) {
    player.pause();
    isPlaying.value = false;
  } else {
    player.play();
    isPlaying.value = true;
  }

  onClickOutside(canvasEl.value, () => {
    if (isPlaying.value) {
      player.pause();
      isPlaying.value = false;
    }
  });
};

const onZoomIn = async () => {
  zoomLevel.value += zoomStep.value;
};

const onZoomOut = async () => {
  zoomLevel.value -= zoomStep.value;
};

watch(zoomLevel, () => {
  const zoomview = peaksInstanceRef.value?.views.getView("zoomview");
  zoomview?.setZoom({ seconds: zoomLevel.value });
}, { immediate: true });

const onAmpPlus = async () => {
  zoomviewAmpZoom.value += zoomAmpStep.value;
};

const onAmpMinus = async () => {
  zoomviewAmpZoom.value -= zoomAmpStep.value;
};

watch(zoomviewAmpZoom, () => {
  zoomviewAmpZoom.value = Math.ceil(zoomviewAmpZoom.value * 100) / 100;
  const overview = peaksInstanceRef.value?.views.getView("overview");
  overview?.setAmplitudeScale(zoomviewAmpZoom.value);

  const zoomview = peaksInstanceRef.value?.views.getView("zoomview");
  zoomview?.setAmplitudeScale(zoomviewAmpZoom.value);
}, { immediate: true });

const chooseSegment = (segmentId: string) => {
  const segment = peaksInstanceRef.value?.segments.getSegment(segmentId);
  if (segment) {
    peaksInstanceRef.value?.player.seek(segment.startTime);
  }
};

const currentTime = ref(peaksInstanceRef.value?.player.getCurrentTime());
let interval = shallowRef();

onMounted(() => {
  interval.value = setInterval(() => {
    const zoomview = peaksInstanceRef.value?.views.getView("zoomview");

    zoomview?.setAmplitudeScale(zoomviewAmpZoom.value);

    currentTime.value = peaksInstanceRef.value?.player.getCurrentTime();
  }, 100);
});

const playTime = computed(() => {
  return currentTime.value ? parsePlayTime(currentTime.value) : null;
});

onUnmounted(() => {
  if (interval.value) {
    clearInterval(interval.value);
  }
});
</script>

<template>
  <VirtualizedItem :min-height="props.height" :preload-margin="400" class="canvas" ref="canvasEl">
    <div class="overview">
      <div class="overview__waveform" ref="containerEl"></div>
      <div class="overview__controls">
        <Button v-if="peaksInstanceRef" shape="square" size="lg" variant="ghost" color="primary" @click="onPlay">
          <font-awesome-icon v-if="isPlaying" icon="fa-solid fa-pause" />
          <font-awesome-icon v-else icon="fa-solid fa-play" />
        </Button>
      </div>
    </div>

    <div class="zoomview">
      <div ref="zoomViewEl" class="zoomview__waveform"></div>
      <div class="zoomview__controls">
        <TextInput type="number" v-model="zoomLevel" :step="zoomStep" size="md" variant="outlined" color="neutral">
          <template #prepend>
            <font-awesome-icon icon="fa-solid fa-stopwatch" />
          </template>

          <template #actions>
            <div class="zoomview__input__controls">
              <Button
                shape="square" size="xs"
                color="primary"
                variant="ghost"
                @click="onZoomOut">
                <font-awesome-icon icon="fa-solid fa-minus" />
              </Button>

              <Button
                shape="square" size="xs"
                color="primary"
                variant="ghost"
                @click="onZoomIn">
                <font-awesome-icon icon="fa-solid fa-plus" />
              </Button>
            </div>
          </template>
        </TextInput>

        <TextInput type="number" v-model="zoomviewAmpZoom" :step="zoomAmpStep" size="md" variant="outlined"
                   color="neutral">
          <template #prepend>
            <font-awesome-icon icon="fa-solid fa-arrow-up-right-dots" />
          </template>

          <template #actions>
            <div class="zoomview__input__controls">
              <Button
                shape="square" size="xs"
                color="primary"
                variant="ghost"
                @click="onAmpMinus">
                <font-awesome-icon icon="fa-solid fa-minus" />
              </Button>

              <Button
                shape="square" size="xs"
                color="primary"
                variant="ghost"
                @click="onAmpPlus">
                <font-awesome-icon icon="fa-solid fa-plus" />
              </Button>
            </div>
          </template>
        </TextInput>

        <div class="playtime">
          <template v-if="playTime">
            {{ `${playTime.hours}:${playTime.minutes}:${playTime.seconds}` }}<small>.{{ playTime.milliseconds }}</small>
          </template>
          <template v-else>
            00:00:00.<small>000</small>
          </template>
        </div>

        <Button size="xs" @click="addSegment" class="add-segment-btn">
          Add Segment
        </Button>
      </div>
    </div>

    <div class="segments">
      <table>
        <tr v-for="segment in segments.values()" :key="segment.id" @click="() => chooseSegment(segment.id)">
          <td>
            <Marker :index="segment.startIndex" :time="segment.startTime" />
          </td>
          <td>
            <Marker :index="segment.endIndex" :time="segment.endTime" />
          </td>
          <td>{{ segment.labelText }}</td>
          <td></td>
        </tr>
        <tr>
          <td></td>
          <td>

          </td>
          <td>

          </td>
          <td>

          </td>

        </tr>
      </table>
    </div>
    <audio ref="mediaEl">
      <source v-for="url in audioUrls" :key="url.type" v-bind="url">
    </audio>
  </VirtualizedItem>
</template>

<style scoped>
.canvas {
  position: relative;
  width: 100%;
  border-top: 1px solid var(--color-grey-300);
  border-bottom: 1px solid var(--color-grey-300);
}

.zoomview {
  position: relative;
  border-top: 1px solid var(--color-grey-300);
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: stretch;
  width: 100%;
  height: 260px;
}

.zoomview__waveform {
  flex: 1;
  height: 260px;
  position: relative;
}

.zoomview__controls {
  width: 200px;
  padding: var(--size-2);
  flex: none;
  flex-wrap: nowrap;
  gap: var(--size-2);
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  flex-direction: column;
  top: 0;
  right: 0;
  margin: var(--size-1);
}

.overview {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: stretch;
  width: 100%;
  height: 80px;
}

.overview > * {
  height: 80px;
}

.overview__controls {
  flex: none;
  width: 80px;
  height: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.overview__waveform {
  flex: 1;
  height: 80px;
}

.zoomview__input__controls {
  display: flex;
  height: 100%;
  gap: var(--input-padding);
  margin: var(--input-padding);
}

.segments {
  margin-top: var(--size-10);
  padding: var(--size-6);
}

.segments table {
  width: 100%;

  td, th {
    padding: var(--size-2);
    border-bottom: 1px solid var(--color-grey-300);
  }
}

.playtime {
  width: 100%;
  padding: var(--size-2);
  border-radius: var(--radius-sm);
  border: 2px solid var(--color-grey-200);
  display: flex;
  align-items: baseline;
  justify-content: center;
  background-color: var(--color-grey-50);
  font-size: var(--scale-4);

  small {
    font-size: var(--scale-2);
    color: var(--color-grey-700);
  }
}

.add-segment-btn {
  width: 100%;
}
</style>