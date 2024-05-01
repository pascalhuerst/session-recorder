<script setup lang="ts">
defineProps<{
  isRecording?: boolean;
}>();
</script>

<template>
  <div class="status">
    <span :class="['text', { 'is-recording': isRecording }]">
      {{ isRecording ? 'rec' : 'off' }}
    </span>
    <span :class="['indicator', { 'is-recording': isRecording }]" />
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
  background: var(--color-grey-400);
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
  color: var(--color-grey-400);
}

.text.is-recording {
  color: var(--color-red-500);
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
</style>
