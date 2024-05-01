<script setup lang="ts">
import { type Recorder } from '@session-recorder/protocols/ts/sessionsource';
import { SignalStatus } from '@session-recorder/protocols/ts/common';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import StatusIndicator from './StatusIndicator.vue';
import { computed } from 'vue';

const props = defineProps<{
  recorder: Recorder;
  isSelected?: boolean;
}>();

const isRecording = computed(() => {
  return props.recorder.status.signalStatus === SignalStatus.SIGNAL;
});
</script>

<template>
  <div role="button" :class="['link', { 'is-selected': isSelected }]">
    <div class="name">
      <font-awesome-icon icon="fa-solid fa-microchip" />
      {{ recorder.recorderName }}
    </div>
    <div class="indicators">
      <StatusIndicator :isRecording="isRecording" />
      <v-gauge :value="recorder.status.rmsPercent" />
    </div>
  </div>
</template>

<style scoped>
.link {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
  color: var(--color-black);
  background: transparent;
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  padding: var(--size-4) var(--size-4);
  white-space: nowrap;
  border-radius: var(--radius-sm);
  text-decoration: none;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.25s;
  width: var(--size-32);
  overflow: hidden;
}

.link:hover,
.link.is-selected {
  background: #fff;
  border-color: var(--color-purple-700);
}

.name {
  display: flex;
  align-items: center;
  gap: var(--size-1);
  white-space: nowrap;
  overflow: hidden;
  color: var(--color-grey-800);
  font-size: var(--size-4);
}
</style>
