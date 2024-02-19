<script setup lang="ts">
import SeekInput from './controls/SeekInput.vue';
import ZoomInput from './controls/ZoomInput.vue';
import AmplitudeInput from './controls/AmplitudeInput.vue';
import AddSegmentButton from '../Segments/controls/AddSegmentButton.vue';
import { usePeaksContext } from '../../context/usePeaksContext';
import { useWaverformLayoutProvider } from '../../waveform/useWaverformLayoutProvider';

const { inject } = useWaverformLayoutProvider();
const { zoomviewRef } = inject();
</script>

<template>
  <div class="zoomview">
    <div ref="zoomviewRef" class="zoomview__waveform"></div>
    <div class="zoomview__controls">
      <SeekInput />
      <ZoomInput />
      <AmplitudeInput />
      <AddSegmentButton />
    </div>
  </div>
</template>

<style scoped>
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

.zoomview__controls > * {
  width: 100%;
}
</style>

<i18n locale="en" lang="json"></i18n>
