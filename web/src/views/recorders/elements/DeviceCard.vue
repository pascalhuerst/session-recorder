<script setup lang="ts">
import { type Recorder } from '@session-recorder/protocols/ts/sessionsource';
import { SignalStatus } from '@session-recorder/protocols/ts/common';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import StatusIndicator from './StatusIndicator.vue';
import { computed } from 'vue';
import RmsIndicator from './RmsIndicator.vue';

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
    <div class="meta">
      <font-awesome-icon icon="fa-solid fa-microchip" />
      <div class="name">{{ recorder.recorderName }}</div>
    </div>
    <div class="indicators">
      <StatusIndicator :isRecording="isRecording" />
      <RmsIndicator v-if="isRecording" :value="recorder.status.rmsPercent" />
    </div>
  </div>
</template>

<style scoped>
.link {
  display: flex;
  flex-direction: column;
  gap: var(--size-2);
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
  width: var(--size-40);
  overflow: hidden;
}

.link:hover,
.link.is-selected {
  background: #fff;
}

.meta {
  display: flex;
  align-items: center;
  gap: var(--size-1);
}

.indicators {
  display: grid;
  gap: var(--size-2);
  grid-template-columns: auto 1fr;
}

.name {
  margin-right: auto;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--color-grey-800);
}
</style>
