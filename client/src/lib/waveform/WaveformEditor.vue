<script setup lang="ts">
import { ref, shallowRef, watch } from "vue";
import VirtualizedItem from "../disclosure/VirtualizedItem.vue";
import Peaks, { type PeaksInstance, type PeaksOptions } from "peaks.js";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { onClickOutside } from "@vueuse/core";

const props = withDefaults(defineProps<{
  waveformUrl: string
  audioUrls: Array<{ src: string, type: string }>
  height?: number
}>(), {
  height: 200
});

const canvasEl = ref<HTMLElement>();
const containerEl = ref<HTMLElement>();
const mediaEl = ref<HTMLAudioElement>();
const peaksInstanceRef = shallowRef<PeaksInstance>();

watch([containerEl, mediaEl, props], () => {
  if (containerEl.value && mediaEl.value) {
    const options = {
      overview: {
        container: containerEl.value,
        enablePoints: false,
        enableSegments: false,
        playheadPadding: 16,
        playheadColor: "#6b46c1",
        playedWaveformColor: "#bdafe3",
        showPlayheadTime: true,
        playheadTextColor: "#6b46c1"
      },
      mediaElement: mediaEl.value,
      dataUri: {
        arraybuffer: props.waveformUrl
      }
    } satisfies PeaksOptions;

    Peaks.init(options, function(err, peaksInstance) {
      if (err || !peaksInstance) {
        console.warn(err.message);
        return null;
      }

      peaksInstanceRef.value = peaksInstance;
      // const overview = peaksInstance.views.getView("overview");
      // const zoomview = peaksInstance.views.getView("zoomview");
      //
      // overview?.enableSeek(false);
      // zoomview?.enableSeek(false);
    });
  }
});

const isPlaying = ref(false);

const onClick = () => {
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
</script>

<template>
  <VirtualizedItem :min-height="props.height" :preload-margin="400" class="canvas" ref="canvasEl">
    <div ref="containerEl" :style="{ height: `${props.height}px` }" class="overview"></div>
    <audio ref="mediaEl">
      <source v-for="url in audioUrls" :key="url.type" v-bind="url">
    </audio>
    <button v-if="peaksInstanceRef" class="play" @click="onClick">
      <font-awesome-icon v-if="isPlaying" icon="fa-solid fa-pause" />
      <font-awesome-icon v-else icon="fa-solid fa-play" />
    </button>
  </VirtualizedItem>
</template>

<style scoped>
.canvas {
  position: relative;
  border-radius: var(--radius-sm);
  box-shadow: rgba(0, 0, 0, 0.05) 0px 6px 24px 0px, rgba(0, 0, 0, 0.08) 0px 0px 0px 1px;
}

.overview {
  display: block;
  width: 100%;
}

.play {
  position: absolute;
  bottom: 0;
  left: 0;
  margin: var(--size-3);
  width: var(--size-12);
  height: var(--size-12);
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--size-6);
  border: 1px solid var(--color-grey-200);
  background: var(--color-grey-100);
  color: var(--color-purple-700);
  box-shadow: rgba(17, 17, 26, 0.1) 0px 4px 16px, rgba(17, 17, 26, 0.05) 0px 8px 32px;
}
</style>