<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue';
import VirtualizedItem from '../disclosure/VirtualizedItem.vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import Button from '../controls/Button.vue';
import TextInput from '../forms/TextInput.vue';
import uuid from 'uuidv4';
import Marker from '../controls/Marker.vue';
import { parsePlayTime } from '../utils/parsePlayTime';
import { overviewTheme, zoomviewTheme } from '../context/theme';
import { createPeaksContext } from '../context/usePeaksContext';
import Overview from '../elements/Overview/Overview.vue';

type Segment = {
  startTime: number;
  endTime: number;
  editable?: boolean;
  color?: string;
  labelText?: string;
  id?: string;
  [key: string]: any;
};

const props = withDefaults(
  defineProps<{
    waveformUrl?: string;
    audioUrls: Array<{ src: string; type: string }>;
    height?: number;
    segments?: Map<string, Segment & { id: string }>;
  }>(),
  {
    height: 200,
  }
);

const zoomLevel = ref(300);
const zoomviewAmpZoom = ref(0.6);
const zoomStep = ref(100);
const zoomAmpStep = ref(0.1);

const { peaks, canvasElement, zoomviewElement, audioElement } =
  createPeaksContext({
    overviewTheme,
    zoomviewTheme,
    waveformUrl: computed(() => props.waveformUrl),
  });

const segments = ref(new Map(props.segments));

watch(
  [segments, peaks],
  () => {
    if (peaks.value && segments.value.size) {
      try {
        segments.value.forEach((segment) => {
          peaks.value?.segments.removeById(segment.id);
          peaks.value?.segments.add(segment);
        });
      } catch (err) {
        console.log(err);
      }
      // const overview = peaks.value?.views.getView("overview");
      // overview?.setPlayedWaveformColor("#767c89");
      //
      // const zoomview = peaks.value?.views.getView("zoomview");
      // zoomview?.setPlayedWaveformColor("#767c89");
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

const intToChar = (int: number) => {
  const start = 'a'.charCodeAt(0);
  return String.fromCharCode(start + int).toUpperCase();
};

const addSegment = () => {
  const segmentId = uuid();
  const size = segments.value.size * 2;
  segments.value.set(segmentId, {
    id: segmentId,
    startTime: Number(peaks.value?.player.getCurrentTime()),
    endTime: Number(peaks.value?.player.getCurrentTime()) + 5,
    color: '#ed64a6',
    labelText: '0 to 10.5 seconds non-editable demo segment',
    editable: true,
    startIndex: intToChar(size),
    endIndex: intToChar(size + 1),
  });
};

const onZoomIn = async () => {
  zoomLevel.value = Math.max(0, zoomLevel.value + zoomStep.value);
};

const onZoomOut = async () => {
  zoomLevel.value = Math.max(0, zoomLevel.value - zoomStep.value);
};

watch(
  zoomLevel,
  () => {
    const zoomview = peaks.value?.views.getView('zoomview');
    zoomview?.setZoom({ seconds: zoomLevel.value });
  },
  { immediate: true }
);

const onAmpPlus = async () => {
  zoomviewAmpZoom.value = Math.max(
    0,
    (zoomviewAmpZoom.value * 100 + zoomAmpStep.value * 100) / 100
  );
};

const onAmpMinus = async () => {
  zoomviewAmpZoom.value = Math.max(
    0,
    (zoomviewAmpZoom.value * 100 - zoomAmpStep.value * 100) / 100
  );
};

watch(
  zoomviewAmpZoom,
  () => {
    const overview = peaks.value?.views.getView('overview');
    overview?.setAmplitudeScale(zoomviewAmpZoom.value);

    const zoomview = peaks.value?.views.getView('zoomview');
    zoomview?.setAmplitudeScale(zoomviewAmpZoom.value);
  },
  { immediate: true }
);

const chooseSegment = (segmentId: string) => {
  const segment = peaks.value?.segments.getSegment(segmentId);
  if (segment) {
    peaks.value?.player.seek(segment.startTime);
  }
};

const currentTime = ref(peaks.value?.player.getCurrentTime());
let interval = shallowRef();

onMounted(() => {
  const zoomview = peaks.value?.views.getView('zoomview');
  zoomview?.setAmplitudeScale(zoomviewAmpZoom.value);

  interval.value = setInterval(() => {
    currentTime.value = peaks.value?.player.getCurrentTime();
  }, 100);
});

onUnmounted(() => {
  if (interval.value) {
    clearInterval(interval.value);
  }
});

const playTime = computed({
  get: () => {
    if (!currentTime.value) {
      return '00:00:00';
    }
    const parsed = parsePlayTime(currentTime.value);
    return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
  },
  set: (value: string) => {
    const [h, m, s, ms] =
      value
        .match(/(\d{2}):(\d{2}):(\d{2})\.(\d{3})/)
        ?.slice(1, 5)
        .map((val) => parseInt(val)) || [];
    const seconds = h * 3600 + m * 60 + s + ms / 1000;
    peaks.value?.player.seek(seconds);
  },
});

const maxPlayTime = computed(() => {
  if (!peaks.value?.player.getDuration()) {
    return '00:00:00';
  }

  const parsed = parsePlayTime(peaks.value?.player.getDuration());
  return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
});

const showPicker = (ref?: HTMLInputElement) => {
  return ref?.showPicker();
};
</script>

<template>
  <VirtualizedItem
    :min-height="props.height"
    :preload-margin="400"
    class="canvas"
    ref="canvasElement"
  >
    <Overview />
    <div class="zoomview">
      <div ref="zoomviewElement" class="zoomview__waveform"></div>
      <div class="zoomview__controls">
        <TextInput
          type="time"
          v-model="playTime"
          :step="0.01"
          min="00:00:00"
          :max="maxPlayTime"
          size="md"
          variant="outlined"
          color="neutral"
        >
          <template #prepend="{ inputRef }">
            <font-awesome-icon
              icon="fa-regular fa-clock"
              @click="showPicker(inputRef)"
            />
          </template>
        </TextInput>

        <TextInput
          type="number"
          v-model="zoomLevel"
          :step="zoomStep"
          :min="0"
          size="md"
          variant="outlined"
          color="neutral"
        >
          <template #prepend>
            <font-awesome-icon icon="fa-solid fa-arrows-left-right-to-line" />
          </template>

          <template #actions>
            <div class="zoomview__input__controls">
              <Button
                shape="square"
                size="xs"
                color="primary"
                variant="ghost"
                @click="onZoomOut"
                :disabled="zoomLevel <= 0"
              >
                <font-awesome-icon icon="fa-solid fa-minus" />
              </Button>

              <Button
                shape="square"
                size="xs"
                color="primary"
                variant="ghost"
                @click="onZoomIn"
              >
                <font-awesome-icon icon="fa-solid fa-plus" />
              </Button>
            </div>
          </template>
        </TextInput>

        <TextInput
          type="number"
          v-model="zoomviewAmpZoom"
          :step="zoomAmpStep"
          size="md"
          variant="outlined"
          color="neutral"
        >
          <template #prepend>
            <font-awesome-icon icon="fa-solid fa-arrow-up-right-dots" />
          </template>

          <template #actions>
            <div class="zoomview__input__controls">
              <Button
                shape="square"
                size="xs"
                color="primary"
                variant="ghost"
                @click="onAmpMinus"
                :disabled="zoomviewAmpZoom <= 0"
              >
                <font-awesome-icon icon="fa-solid fa-minus" />
              </Button>

              <Button
                shape="square"
                size="xs"
                color="primary"
                variant="ghost"
                @click="onAmpPlus"
              >
                <font-awesome-icon icon="fa-solid fa-plus" />
              </Button>
            </div>
          </template>
        </TextInput>

        <Button size="xs" @click="addSegment" class="add-segment-btn">
          Add Segment
        </Button>
      </div>
    </div>

    <div class="segments">
      <table>
        <tr
          v-for="segment in segments.values()"
          :key="segment.id"
          @click="() => chooseSegment(segment.id)"
        >
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
          <td></td>
          <td></td>
          <td></td>
        </tr>
      </table>
    </div>
    <audio ref="audioElement">
      <source v-for="url in audioUrls" :key="url.type" v-bind="url" />
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

.zoomview__input__controls {
  display: flex;
  height: 100%;
  gap: var(--input-padding);
  margin: var(--input-padding);
}

.zoomview__controls > * {
  width: 100%;
}

.segments {
  margin-top: var(--size-10);
  padding: var(--size-6);
}

.segments table {
  width: 100%;

  td,
  th {
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
