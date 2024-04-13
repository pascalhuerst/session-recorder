<script setup lang="ts">
import { ref, watchEffect } from 'vue';
import VirtualizedItem from '../lib/disclosure/VirtualizedItem.vue';
import { usePeaksContext } from '../context/usePeaksContext';
import Overview from '../elements/Overview/Overview.vue';
import Zoomview from '../elements/Zoomview/Zoomview.vue';
import Audio from '../elements/Overview/Audio.vue';
import Segments from '../elements/Segments/Segments.vue';
import { onClickOutside } from '@vueuse/core';
import { useWaverformLayoutProvider } from './useWaverformLayoutProvider';

const props = withDefaults(
  defineProps<{
    height?: number;
  }>(),
  {
    height: 200,
  }
);

const context = usePeaksContext();
const { provide } = useWaverformLayoutProvider();

const overviewRef = ref<HTMLElement>();
const zoomviewRef = ref<HTMLElement>();
const audioRef = ref<HTMLElement>();

provide({
  overviewRef,
  zoomviewRef,
  audioRef,
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
  <slot />
</template>

<style scoped>
.canvas {
  position: relative;
  width: 100%;
  border-top: 1px solid var(--color-grey-300);
  border-bottom: 1px solid var(--color-grey-300);
}
</style>
