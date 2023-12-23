<script setup lang="ts">
import { ref, watch } from "vue";
import VirtualizedItem from "../disclosure/VirtualizedItem.vue";
import Peaks, { type PeaksOptions } from "peaks.js";

const props = withDefaults(defineProps<{
  waveformUrl: string
  audioUrls: Array<{ src: string, type: string }>
  height?: number
}>(), {
  height: 200
});

const containerEl = ref<HTMLElement>();
const mediaEl = ref<HTMLElement>();

watch([containerEl, mediaEl, props], () => {
  if (containerEl.value && mediaEl.value) {
    const options = {
      overview: {
        container: containerEl.value,
        enablePoints: false,
        enableSegments: false,
        playheadPadding: 0,
        playheadColor: 'transparent'
      },
      mediaElement: mediaEl.value,
      dataUri: {
        arraybuffer: props.waveformUrl
      },
    } satisfies PeaksOptions;

    Peaks.init(options, function(err, peaksInstance) {
      if (err || !peaksInstance) {
        console.warn(err.message);
        return null;
      }

      const overview = peaksInstance.views.getView("overview");
      const zoomview = peaksInstance.views.getView("zoomview");

      // @todo: any better way or do we want to allow playing?
      overview?.enableSeek(false);
      zoomview?.enableSeek(false);
    });
  }
});
</script>

<template>
  <VirtualizedItem :min-height="props.height" :preload-margin="400">
    <div ref="containerEl" :style="{ height: `${props.height}px` }" class="overview"></div>
    <audio ref="mediaEl">
      <source v-for="url in audioUrls" :key="url.type" v-bind="url">
    </audio>
  </VirtualizedItem>
</template>

<style scoped>
.overview {
  display: block;
  width: 100%;
}
</style>