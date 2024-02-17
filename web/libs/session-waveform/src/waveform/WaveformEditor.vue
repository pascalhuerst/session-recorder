<script setup lang="ts">
import { computed, defineModel, ref } from 'vue';
import VirtualizedItem from '../lib/disclosure/VirtualizedItem.vue';
import { overviewTheme, zoomviewTheme } from '../context/theme';
import { createPeaksContext } from '../context/usePeaksContext';
import Overview from '../elements/Overview/Overview.vue';
import Zoomview from '../elements/Zoomview/Zoomview.vue';
import type { AudioSourceUrl, Segment } from '../types';
import { type Permissions } from '../types';
import Audio from '../elements/Overview/Audio.vue';
import Segments from '../elements/Segments/Segments.vue';

const segments = defineModel<Segment[]>('segments', { required: true });

const props = withDefaults(
  defineProps<{
    audioUrls: AudioSourceUrl[];
    waveformUrl?: string;
    permissions: Permissions;
    height?: number;
  }>(),
  {
    height: 200,
  }
);

const {
  layout: { canvasElement },
} = createPeaksContext({
  overviewTheme: ref(overviewTheme),
  zoomviewTheme: ref(zoomviewTheme),
  waveformUrl: computed(() => props.waveformUrl),
  audioUrls: computed(() => props.audioUrls),
  permissions: computed(() => props.permissions),
  segments,
});
</script>

<template>
  <VirtualizedItem
    :min-height="props.height"
    :preload-margin="400"
    class="canvas"
    ref="canvasElement"
  >
    <Overview />
    <Zoomview />
    <Segments />
    <Audio />
  </VirtualizedItem>
</template>

<style scoped>
.canvas {
  position: relative;
  width: 100%;
  border-top: 1px solid var(--color-grey-300);
  border-bottom: 1px solid var(--color-grey-300);
}
</style>
