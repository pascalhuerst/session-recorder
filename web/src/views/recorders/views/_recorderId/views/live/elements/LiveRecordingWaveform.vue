<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { useDateFormat } from '@vueuse/core';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { Button } from '@session-recorder/session-waveform';
import { cutSession } from '@/grpc/procedures/cutSession';
import { toastService } from '@/services/Toaster/ToastService';
import {
  streamSessionAudio,
  type AudioChunkMessage,
} from '@/grpc/procedures/streamSessionAudio';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const isCutting = ref(false);
const now = ref(new Date());
let clockInterval: ReturnType<typeof setInterval>;

onMounted(() => {
  clockInterval = setInterval(() => {
    now.value = new Date();
  }, 1000);
});

onUnmounted(() => {
  clearInterval(clockInterval);
});

const elapsedTime = computed(() => {
  const diff = now.value.getTime() - props.session.startedAt.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const pad = (n: number) => n.toString().padStart(2, '0');
  if (hours > 0) return `${pad(hours)}:${pad(minutes % 60)}:${pad(seconds % 60)}`;
  return `${pad(minutes)}:${pad(seconds % 60)}`;
});

const startTime = computed(() => {
  return useDateFormat(props.session.startedAt, 'HH:mm').value;
});

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

// Canvas-based waveform
const canvasRef = ref<HTMLCanvasElement | null>(null);
let animationId: number | null = null;
let unsubscribeAudio: (() => void) | null = null;

const peaks = ref<number[]>([]);

const computePeaks = (samples: Int16Array, windowSize: number): number[] => {
  const result: number[] = [];
  for (let i = 0; i < samples.length; i += windowSize) {
    let max = 0;
    const end = Math.min(i + windowSize, samples.length);
    for (let j = i; j < end; j++) {
      max = Math.max(max, Math.abs(samples[j]));
    }
    result.push(max / 32767);
  }
  return result;
};

let lastCanvasWidth = 0;

const onAudioChunk = (chunk: AudioChunkMessage) => {
  const windowSize = Math.max(1, Math.floor(chunk.samples.length / 6));
  const newPeaks = computePeaks(chunk.samples, windowSize);
  const maxPeaks = Math.floor((lastCanvasWidth || 800) / 6);
  peaks.value = [...peaks.value, ...newPeaks].slice(-maxPeaks);
};

const drawWaveform = () => {
  const canvas = canvasRef.value;
  if (!canvas) return;

  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const dpr = window.devicePixelRatio || 1;
  const width = canvas.clientWidth;
  const height = canvas.clientHeight;

  lastCanvasWidth = Math.floor(width);

  if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);
  }

  ctx.clearRect(0, 0, width, height);

  const centerY = height / 2;
  const maxAmplitudeHeight = height * 0.45;
  const barWidth = 4;
  const barGap = 2;
  const step = barWidth + barGap;
  const minBarHeight = 4;

  ctx.fillStyle = '#94a3b8';

  const peaksData = peaks.value;
  const peaksLength = peaksData.length;

  if (peaksLength === 0) {
    animationId = requestAnimationFrame(drawWaveform);
    return;
  }

  // How many bars fit on screen
  const maxBars = Math.floor(width / step);
  const startIdx = Math.max(0, peaksLength - maxBars);

  for (let i = startIdx; i < peaksLength; i++) {
    const raw = peaksData[i] || 0;
    // Boost quiet signals with sqrt curve and apply gain
    const boosted = Math.sqrt(raw) * 3;
    const amplitude = Math.min(boosted, 1);
    const barHeight = Math.max(amplitude * maxAmplitudeHeight * 2, minBarHeight);
    const barIndex = i - startIdx;
    const x = barIndex * step;
    const y = centerY - barHeight / 2;

    ctx.beginPath();
    ctx.roundRect(x, y, barWidth, barHeight, 2);
    ctx.fill();
  }

  animationId = requestAnimationFrame(drawWaveform);
};

let retryTimeout: ReturnType<typeof setTimeout> | null = null;
let mounted = true;

function subscribeAudio() {
  unsubscribeAudio = streamSessionAudio({
    sessionID: props.session.id,
    onChunk: onAudioChunk,
    onError: (error) => {
      console.error('Audio stream error:', error);
      if (mounted) {
        retryTimeout = setTimeout(() => subscribeAudio(), 2000);
      }
    },
    onEnd: () => console.log('Audio stream ended'),
  });
}

onMounted(() => {
  animationId = requestAnimationFrame(drawWaveform);
  subscribeAudio();
});

onUnmounted(() => {
  mounted = false;
  if (animationId !== null) cancelAnimationFrame(animationId);
  if (unsubscribeAudio) unsubscribeAudio();
  if (retryTimeout !== null) clearTimeout(retryTimeout);
});

watch(
  () => props.session.id,
  (newId, oldId) => {
    if (newId !== oldId) {
      peaks.value = [];
      if (unsubscribeAudio) unsubscribeAudio();
      if (retryTimeout !== null) clearTimeout(retryTimeout);
      subscribeAudio();
    }
  },
);
</script>

<template>
  <div class="live-waveform">
    <div class="waveform-header">
      <div class="recording-status">
        <span class="rec-dot" />
        <span class="rec-label">REC</span>
      </div>
      <span class="elapsed">{{ elapsedTime }}</span>
      <span class="since">since {{ startTime }}</span>
      <div class="spacer" />
      <Button
        shape="square"
        size="lg"
        variant="ghost"
        color="primary"
        :disabled="isCutting"
        title="Stop recording"
        @click="handleCutSession"
      >
        <font-awesome-icon icon="fa-solid fa-stop" />
      </Button>
    </div>
    <canvas ref="canvasRef" class="waveform-canvas" />
  </div>
</template>

<style scoped>
.live-waveform {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.waveform-header {
  display: flex;
  align-items: center;
  gap: var(--size-3);
  padding: var(--size-3) var(--size-4);
  flex-shrink: 0;
}

.recording-status {
  display: flex;
  align-items: center;
  gap: var(--size-1);
}

.rec-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #dc2626;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

.rec-label {
  font-size: var(--scale-1);
  font-weight: var(--weight-bold);
  color: #dc2626;
}

.elapsed {
  font-family: monospace;
  font-size: 2rem;
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}

.since {
  font-size: var(--scale-1);
  color: var(--text-muted);
}

.spacer {
  flex: 1;
}

.waveform-canvas {
  flex: 1;
  width: 100%;
  min-height: 0;
}
</style>
