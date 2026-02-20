<script setup lang="ts">
import { Recorder } from '@session-recorder/protocols/ts/sessionsource';
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
  return (
    props.recorder.info.oneofKind === 'status' &&
    props.recorder.info.status.signalStatus === SignalStatus.SIGNAL
  );
});
</script>

<template>
  <div role="button" :class="['device-card', { 'is-selected': isSelected }]">
    <div class="meta">
      <font-awesome-icon icon="fa-solid fa-microchip" class="icon" />
      <div class="name">{{ recorder.recorderName }}</div>
    </div>
    <div class="indicators">
      <StatusIndicator :isRecording="isRecording" />
      <RmsIndicator
        v-if="isRecording && recorder.info.oneofKind === 'status'"
        :value="recorder.info.status.rmsPercent"
      />
    </div>
  </div>
</template>

<style scoped>
.device-card {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
  color: var(--text-primary);
  background: transparent;
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  padding: var(--size-2) var(--size-2);
  border-radius: var(--radius-sm);
  text-decoration: none;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.15s ease;
  width: 100%;
  overflow: hidden;
}

.device-card:hover {
  background: var(--bg-hover);
}

.device-card.is-selected {
  background: var(--bg-selected);
  border-color: var(--accent);
}

.device-card.is-selected:hover {
  background: var(--bg-selected-hover);
}

.meta {
  display: flex;
  align-items: center;
  gap: var(--size-2);
  min-width: 0;
}

.icon {
  flex-shrink: 0;
  color: var(--text-muted);
}

.name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-secondary);
}

.device-card.is-selected .name {
  color: var(--text-primary);
}

.indicators {
  display: flex;
  align-items: center;
  gap: var(--size-2);
  padding-left: calc(var(--size-4) + var(--size-2));
  min-width: 0;
}
</style>
