<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { Button } from '@session-recorder/session-waveform';
import { cutSession } from '../../../grpc/procedures/cutSession';
import { toastService } from '../../../services/Toaster/ToastService';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

// Elapsed time counter
const now = ref(new Date());
let interval: ReturnType<typeof setInterval>;

onMounted(() => {
  interval = setInterval(() => {
    now.value = new Date();
  }, 1000);
});

onUnmounted(() => {
  clearInterval(interval);
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

const isCutting = ref(false);

const handleCutSession = async () => {
  if (isCutting.value) return;
  isCutting.value = true;

  try {
    await cutSession({ recorderID: props.recorderId });
    toastService.success('Session cut successfully');
  } catch (error) {
    console.error('Failed to cut session:', error);
    toastService.error('Failed to cut session');
  } finally {
    isCutting.value = false;
  }
};
</script>

<template>
  <div class="recording-content">
    <div class="waveform-placeholder">
      <div class="wave-animation">
        <span class="bar" v-for="i in 20" :key="i" :style="{ animationDelay: `${i * 0.1}s` }" />
      </div>
    </div>
    <div class="controls">
      <span class="elapsed-time">{{ elapsedTime }}</span>
      <Button
        variant="secondary"
        :disabled="isCutting"
        @click="handleCutSession"
      >
        {{ isCutting ? 'Cutting...' : 'Cut Session' }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.recording-content {
  display: flex;
  flex-direction: column;
  gap: var(--size-2);
  padding: var(--size-2);
}

.waveform-placeholder {
  height: 80px;
  background: var(--color-grey-100);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.wave-animation {
  display: flex;
  gap: 3px;
  align-items: center;
  height: 100%;
}

.bar {
  width: 4px;
  background: var(--color-red-400);
  border-radius: 2px;
  animation: wave 1.2s ease-in-out infinite;
}

@keyframes wave {
  0%,
  100% {
    height: 20%;
  }
  50% {
    height: 80%;
  }
}

.controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--size-2);
}

.elapsed-time {
  font-family: monospace;
  font-size: var(--scale-3);
  font-weight: var(--weight-semibold);
  color: var(--color-red-500);
}
</style>
