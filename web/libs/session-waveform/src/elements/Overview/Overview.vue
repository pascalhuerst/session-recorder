<script setup lang="ts">
import Button from '../../controls/Button.vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { useAudioControls } from '../../context/useAudioControls';
import { usePeaksContext } from '../../context/usePeaksContext';

const { overviewElement } = usePeaksContext();
const { player, onToggle, isPlaying } = useAudioControls();
</script>

<template>
  <div class="overview">
    <div class="overview__waveform" ref="overviewElement"></div>
    <div class="overview__controls">
      <Button
        shape="square"
        size="lg"
        variant="ghost"
        color="primary"
        :disabled="!player"
        @click="onToggle"
      >
        <font-awesome-icon v-if="isPlaying" icon="fa-solid fa-pause" />
        <font-awesome-icon v-else icon="fa-solid fa-play" />
      </Button>
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
