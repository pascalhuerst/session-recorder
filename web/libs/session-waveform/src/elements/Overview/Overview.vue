<script setup lang="ts">
import { usePeaksContext } from '../../context/usePeaksContext';
import PlayButton from '../../elements/Player/PlayButton.vue';

const { overviewElement } = usePeaksContext();
</script>

<template>
  <div class="overview">
    <div class="overview__waveform" ref="overviewElement"></div>
    <div class="overview__controls">
      <PlayButton />
    </div>
  </div>
</template>

<style scoped>
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
</style>

<i18n locale="en" lang="json"></i18n>
