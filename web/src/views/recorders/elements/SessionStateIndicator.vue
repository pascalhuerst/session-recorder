<script setup lang="ts">
import type { SessionState } from '@/types';

defineProps<{
  state: SessionState;
}>();

const stateConfig = {
  recording: { text: 'Recording', class: 'is-recording' },
  processing: { text: 'Processing', class: 'is-processing' },
  finished: { text: 'Ready', class: 'is-finished' },
  error: { text: 'Error', class: 'is-error' },
};
</script>

<template>
  <div :class="['state-indicator', stateConfig[state].class]">
    <span class="indicator" />
    <span class="text">{{ stateConfig[state].text }}</span>
  </div>
</template>

<style scoped>
.state-indicator {
  display: flex;
  gap: var(--size-1);
  align-items: center;
}

.indicator {
  width: var(--size-2);
  height: var(--size-2);
  border-radius: 50%;
  flex-shrink: 0;
}

.text {
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Recording state */
.is-recording .indicator {
  background: var(--color-red-500);
  animation: pulse 2s ease-in-out infinite;
}

.is-recording .text {
  color: var(--color-red-500);
}

/* Processing state */
.is-processing .indicator {
  background: var(--color-yellow-500);
  animation: spin 1.5s linear infinite;
}

.is-processing .text {
  color: var(--color-yellow-600);
}

/* Finished state */
.is-finished .indicator {
  background: var(--color-green-500);
}

.is-finished .text {
  color: var(--color-green-600);
}

/* Error state */
.is-error .indicator {
  background: var(--color-red-700);
}

.is-error .text {
  color: var(--color-red-700);
}

@keyframes pulse {
  0%,
  100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.2);
    opacity: 0.7;
  }
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>
