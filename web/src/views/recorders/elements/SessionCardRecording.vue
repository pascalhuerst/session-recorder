<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { Button } from '@session-recorder/session-waveform';
import { cutSession } from '../../../grpc/procedures/cutSession';
import { toastService } from '../../../services/Toaster/ToastService';
import {
  streamSessionAudio,
  type AudioChunkMessage,
} from '../../../grpc/procedures/streamSessionAudio';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

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

// Canvas-based waveform with real audio data
const canvasRef = ref<HTMLCanvasElement | null>(null);
let animationId: number | null = null;
let unsubscribeAudio: (() => void) | null = null;

// Store computed peaks for rendering
// Each peak is a normalized value 0-1 representing max amplitude in a time window
const peaks = ref<number[]>([]);

// Compute peaks from int16 samples
// windowSize: number of samples per peak
const computePeaks = (samples: Int16Array, windowSize: number): number[] => {
  const result: number[] = [];
  for (let i = 0; i < samples.length; i += windowSize) {
    let max = 0;
    const end = Math.min(i + windowSize, samples.length);
    for (let j = i; j < end; j++) {
      max = Math.max(max, Math.abs(samples[j]));
    }
    // Normalize to 0-1 (int16 max is 32767)
    result.push(max / 32767);
  }
  return result;
};

// Track canvas width for peak trimming
let lastCanvasWidth = 0;

const onAudioChunk = (chunk: AudioChunkMessage) => {
  // Compute peaks from samples
  // With 48kHz stereo, each chunk is about 48000 samples per 500ms
  // We want ~10 peaks per chunk for smooth scrolling
  const windowSize = Math.max(1, Math.floor(chunk.samples.length / 10));
  const newPeaks = computePeaks(chunk.samples, windowSize);

  // Append new peaks, trim to canvas width once full
  const maxPeaks = lastCanvasWidth || 400;
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

  // Update canvas width for peak trimming
  lastCanvasWidth = Math.floor(width);

  // Set canvas resolution for sharp rendering
  if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);
  }

  // Clear canvas
  ctx.clearRect(0, 0, width, height);

  const centerY = height / 2;
  const maxAmplitudeHeight = height * 0.4; // Half of 80% since we draw both up and down

  // Use same slate grey as the finished session waveforms
  ctx.fillStyle = '#94a3b8';

  const peaksData = peaks.value;
  const peaksLength = peaksData.length;

  // If we have no peaks yet, nothing to draw
  if (peaksLength === 0) {
    animationId = requestAnimationFrame(drawWaveform);
    return;
  }

  // Draw from left side, 1 pixel per peak
  // Peaks grow from left to right, scroll right when full
  const startX = Math.max(0, peaksLength - width);

  // Draw filled waveform path (Peaks.js style)
  ctx.beginPath();

  // Forward pass: draw top edge (positive amplitude) from left to right
  for (let i = startX; i < peaksLength; i++) {
    const amplitude = peaksData[i] || 0;
    const x = i - startX + 0.5;
    const y = centerY - amplitude * maxAmplitudeHeight + 0.5;

    if (i === startX) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  }

  // Reverse pass: draw bottom edge (negative amplitude) from right to left
  for (let i = peaksLength - 1; i >= startX; i--) {
    const amplitude = peaksData[i] || 0;
    const x = i - startX + 0.5;
    const y = centerY + amplitude * maxAmplitudeHeight + 0.5;

    ctx.lineTo(x, y);
  }

  ctx.closePath();
  ctx.fill();

  animationId = requestAnimationFrame(drawWaveform);
};

onMounted(() => {
  // Start animation loop
  animationId = requestAnimationFrame(drawWaveform);

  // Subscribe to audio stream for this session
  unsubscribeAudio = streamSessionAudio({
    sessionID: props.session.id,
    onChunk: onAudioChunk,
    onError: (error) => {
      console.error('Audio stream error:', error);
    },
    onEnd: () => {
      console.log('Audio stream ended');
    },
  });
});

onUnmounted(() => {
  if (animationId !== null) {
    cancelAnimationFrame(animationId);
  }
  if (unsubscribeAudio) {
    unsubscribeAudio();
  }
});

// Re-subscribe if session changes
watch(
  () => props.session.id,
  (newId, oldId) => {
    if (newId !== oldId) {
      // Clear existing peaks
      peaks.value = [];

      // Unsubscribe from old session
      if (unsubscribeAudio) {
        unsubscribeAudio();
      }

      // Subscribe to new session
      unsubscribeAudio = streamSessionAudio({
        sessionID: newId,
        onChunk: onAudioChunk,
        onError: (error) => {
          console.error('Audio stream error:', error);
        },
        onEnd: () => {
          console.log('Audio stream ended');
        },
      });
    }
  }
);
</script>

<template>
  <div class="recording-overview">
    <div class="controls">
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
.recording-overview {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: stretch;
  width: 100%;
  height: 80px;
}

.recording-overview > * {
  height: 80px;
}

.controls {
  flex: none;
  width: 80px;
  height: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.waveform-canvas {
  flex: 1;
  min-width: 0; /* Prevent Firefox overflow */
  height: 80px;
}
</style>
