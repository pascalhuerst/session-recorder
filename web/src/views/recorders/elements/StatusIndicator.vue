<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  isRecording?: boolean;
  isProcessing?: boolean;
}>();

const label = computed(() => {
  if (props.isRecording) return 'rec';
  if (props.isProcessing) return 'processing';
  return 'off';
});

const stateClass = computed(() => {
  if (props.isRecording) return 'is-recording';
  if (props.isProcessing) return 'is-processing';
  return '';
});
</script>

<template>
  <div class="status">
    <span :class="['indicator', stateClass]" />
    <span :class="['text', stateClass]">
      {{ label }}
    </span>
  </div>
</template>

<style scoped>
.status {
  display: flex;
  gap: var(--size-1);
  align-items: center;
  padding: 0 4px;
}

.indicator {
  background: var(--text-muted);
  border-radius: 50%;
  width: var(--size-2);
  height: var(--size-2);
  display: block;
}

.indicator.is-recording {
  animation: pulse 6s infinite;
  background: var(--color-red-500);
}

.text {
  font-size: var(--size-3);
  font-weight: bold;
  text-transform: uppercase;
  color: var(--text-muted);
}

.text.is-recording {
  color: var(--color-red-500);
}

.text.is-processing {
  color: var(--text-secondary);
}

.indicator.is-processing {
  animation: pulse-grey 6s infinite;
  background: var(--text-secondary);
}

@keyframes pulse {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(213, 63, 140, 0.4);
  }

  25% {
    transform: scale(1);
    box-shadow: 0 0 0 4px rgba(213, 63, 140, 0.2);
  }

  50% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(213, 63, 140, 0.4);
  }

  75% {
    transform: scale(1);
    box-shadow: 0 0 0 4px rgba(213, 63, 140, 0.2);
  }
}

@keyframes pulse-grey {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(107, 114, 128, 0.4);
  }

  25% {
    transform: scale(1);
    box-shadow: 0 0 0 4px rgba(107, 114, 128, 0.2);
  }

  50% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(107, 114, 128, 0.4);
  }

  75% {
    transform: scale(1);
    box-shadow: 0 0 0 4px rgba(107, 114, 128, 0.2);
  }
}
</style>
