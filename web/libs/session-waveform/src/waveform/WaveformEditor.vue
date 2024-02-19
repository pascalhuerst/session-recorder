<script setup lang="ts">
import { defineModel, ref, watchEffect } from 'vue';
import VirtualizedItem from '../lib/disclosure/VirtualizedItem.vue';
import {
  createPeaksContext,
  usePeaksContext,
} from '../context/usePeaksContext';
import Overview from '../elements/Overview/Overview.vue';
import Zoomview from '../elements/Zoomview/Zoomview.vue';
import Audio from '../elements/Overview/Audio.vue';
import Segments from '../elements/Segments/Segments.vue';
import { onClickOutside } from '@vueuse/core';
import type {
  AudioSourceUrl,
  Permissions,
  Segment,
} from '../context/models/state';
import { useWaverformLayoutProvider } from './useWaverformLayoutProvider';

const segments = defineModel<Segment[]>('segments', { required: true });

const props = withDefaults(
  defineProps<{
    audioUrls: [AudioSourceUrl, ...AudioSourceUrl[]];
    waveformUrl?: string;
    permissions: Permissions;
    height?: number;
  }>(),
  {
    height: 200,
  }
);

const { provide } = useWaverformLayoutProvider();

const overviewRef = ref<HTMLElement>();
const zoomviewRef = ref<HTMLElement>();
const audioRef = ref<HTMLElement>();

provide({
  overviewRef,
  zoomviewRef,
  audioRef,
});

const context = createPeaksContext({
  initialState: {
    waveformUrl: props.waveformUrl,
    audioUrls: props.audioUrls,
    permissions: props.permissions,
    segments: segments.value,
  },
});

watchEffect(() => {
  if (overviewRef.value && zoomviewRef.value && audioRef.value) {
    context.commandEmitter.emit('mount', {
      overview: overviewRef.value,
      zoomview: zoomviewRef.value,
      audio: audioRef.value,
    });
  }
});

const canvasElement = ref<HTMLElement>();
onClickOutside(canvasElement.value, () => {
  const { eventEmitter } = usePeaksContext();
  eventEmitter.emit('clickOutsideCanvas');
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
