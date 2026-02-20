<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue';
import { useDateFormat } from '@vueuse/core';
import SessionCardRecording from './SessionCardRecording.vue';
import StatusIndicator from './StatusIndicator.vue';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const now = ref(new Date());
let interval: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  interval = setInterval(() => {
    now.value = new Date();
  }, 1000);
});

onUnmounted(() => {
  if (interval) {
    clearInterval(interval);
  }
});

const elapsedTime = computed(() => {
  const diff = now.value.getTime() - props.session.startedAt.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  const pad = (n: number) => n.toString().padStart(2, '0');

  if (hours > 0) {
    return `${pad(hours)}:${pad(minutes % 60)}:${pad(seconds % 60)}`;
  }
  return `${pad(minutes)}:${pad(seconds % 60)}`;
});

const startTime = computed(() => {
  return useDateFormat(props.session.startedAt, 'HH:mm').value;
});
</script>

<template>
  <div class="recording-card">
    <div class="recording-header">
      <StatusIndicator :is-recording="true" />
      <span class="duration">{{ elapsedTime }}</span>
      <span class="since">since {{ startTime }}</span>
    </div>
    <SessionCardRecording
      :session="session"
      :recorder-id="recorderId"
    />
  </div>
</template>

<style scoped>
.recording-card {
  background: var(--neutral-emphasis);
  margin: calc(-1 * var(--size-4));
  margin-bottom: var(--size-4);
}

.recording-header {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: var(--size-2);
  padding: var(--size-2) var(--size-3);
  white-space: nowrap;
}

.duration {
  font-family: monospace;
  font-size: var(--scale-1);
  font-weight: var(--weight-semibold);
  color: var(--color-red-500);
}

.since {
  font-size: var(--scale-0);
  color: var(--text-muted);
}
</style>
