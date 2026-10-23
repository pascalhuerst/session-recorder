<script setup lang="ts">
import {
  LevelMeter,
  Recorder,
} from '@session-recorder/protocols/ts/sessionsource';
import { SignalStatus } from '@session-recorder/protocols/ts/common';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import StatusIndicator from './StatusIndicator.vue';
import { computed } from 'vue';
import LevelIndicator from './LevelIndicator.vue';

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

const status = computed(() =>
  props.recorder.info.oneofKind === 'status'
    ? props.recorder.info.status
    : undefined,
);

// The backend's --level-meter option decides which meters are shown.
const showRms = computed(() => props.recorder.levelMeter !== LevelMeter.PEAK);
const showPeak = computed(() => props.recorder.levelMeter !== LevelMeter.RMS);
</script>

<template>
  <div role="button" :class="['device-card', { 'is-selected': isSelected }]">
    <div class="meta">
      <font-awesome-icon icon="fa-solid fa-microchip" class="icon" />
      <div class="name">{{ recorder.recorderName }}</div>
    </div>
    <div class="indicators">
      <StatusIndicator :isRecording="isRecording" />
      <div v-if="isRecording && status" class="meters">
        <LevelIndicator
          v-if="showRms"
          label="RMS"
          :value="status.rmsPercent"
        />
        <LevelIndicator
          v-if="showPeak"
          label="Peak"
          :value="status.peakPercent"
          :warn-above="80"
        />
      </div>
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

.meters {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
  flex: 1;
  min-width: 0;
}
</style>
