<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { Button } from '@session-recorder/session-waveform';
import { cutSession } from '../../../grpc/procedures/cutSession';
import { toastService } from '../../../services/Toaster/ToastService';
import {
  streamWaveformPeaks,
  type WaveformPeakMessage,
} from '../../../grpc/procedures/streamWaveformPeaks';
import { reconnectingStream } from '../../../grpc/reconnectingStream';
import { useSessionsStore } from '../../../store/useSessionsStore';
import PeakMeter from './PeakMeter.vue';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const sessionsStore = useSessionsStore();
const isCutting = ref(false);

const handleCutSession = async () => {
  if (isCutting.value) return;
  isCutting.value = true;

  try {
    await cutSession({ recorderID: props.recorderId });
    toastService.success('Session cut successfully');
    // Force-reconnect the sessions stream to pick up the state change.
    // The cut command is forwarded to the recorder asynchronously — the
    // recorder processes it on its next chunk cycle (~100-500ms), then the
    // backend transitions the session. Reconnecting fetches fresh initial
    // state, which avoids relying on a single streamed update that gRPC-Web
    // proxying may buffer.
    setTimeout(() => sessionsStore.reconnect(), 1000);
  } catch (error) {
    console.error('Failed to cut session:', error);
    toastService.error('Failed to cut session');
  } finally {
    isCutting.value = false;
  }
};

// Canvas-based waveform with server-computed peaks
const canvasRef = ref<HTMLCanvasElement | null>(null);
let animationId: number | null = null;
let peakStream: { stop: () => void } | null = null;

// Store min/max peaks from server (int8 values, -128 to 127)
const minPeaks = ref<Int8Array>(new Int8Array(0));
const maxPeaks = ref<Int8Array>(new Int8Array(0));

// Peak level meter state
const peakLevel = ref(0);
const clipping = ref(false);
let clippingTimeout: ReturnType<typeof setTimeout> | null = null;

// Track canvas width for peak trimming
let lastCanvasWidth = 0;

const onWaveformPeaks = (msg: WaveformPeakMessage) => {
  const newPairCount = msg.peaks.length / 2;

  if (msg.isInitial) {
    // Replace all data (reconnect backfill)
    const mins = new Int8Array(newPairCount);
    const maxs = new Int8Array(newPairCount);
    for (let i = 0; i < newPairCount; i++) {
      mins[i] = msg.peaks[i * 2];
      maxs[i] = msg.peaks[i * 2 + 1];
    }
    minPeaks.value = mins;
    maxPeaks.value = maxs;
  } else {
    // Append incremental peaks, trim to canvas width
    const maxPeakCount = lastCanvasWidth || 400;
    const oldLen = minPeaks.value.length;
    const totalLen = oldLen + newPairCount;
    const trimStart = Math.max(0, totalLen - maxPeakCount);

    const newMins = new Int8Array(totalLen - trimStart);
    const newMaxs = new Int8Array(totalLen - trimStart);

    // Copy old data (trimmed from the start)
    const oldStart = Math.max(0, trimStart);
    if (oldLen > oldStart) {
      newMins.set(minPeaks.value.subarray(oldStart), 0);
      newMaxs.set(maxPeaks.value.subarray(oldStart), 0);
    }

    // Append new data
    const writeOffset = Math.max(0, oldLen - oldStart);
    for (let i = 0; i < newPairCount; i++) {
      newMins[writeOffset + i] = msg.peaks[i * 2];
      newMaxs[writeOffset + i] = msg.peaks[i * 2 + 1];
    }

    minPeaks.value = newMins;
    maxPeaks.value = newMaxs;
  }

  // Update peak level meter
  peakLevel.value = msg.peakLevel;
  if (msg.clipping) {
    clipping.value = true;
    // Hold clipping indicator for 1 second
    if (clippingTimeout) clearTimeout(clippingTimeout);
    clippingTimeout = setTimeout(() => {
      clipping.value = false;
    }, 1000);
  }
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

  // Set canvas resolution for sharp rendering
  if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);
  }

  ctx.clearRect(0, 0, width, height);

  const centerY = height / 2;
  const halfHeight = height * 0.45; // Use 90% of height (45% each side)

  ctx.fillStyle = '#94a3b8';

  const mins = minPeaks.value;
  const maxs = maxPeaks.value;
  const peaksLength = mins.length;

  if (peaksLength === 0) {
    animationId = requestAnimationFrame(drawWaveform);
    return;
  }

  // Peaks grow from left to right, scroll when full
  const startX = Math.max(0, peaksLength - width);

  ctx.beginPath();

  // Forward pass: draw max values (positive peaks) from left to right
  for (let i = startX; i < peaksLength; i++) {
    const x = i - startX + 0.5;
    // max values are positive (0 to 127), draw upward from center
    const y = centerY - (maxs[i] / 127) * halfHeight;

    if (i === startX) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  }

  // Reverse pass: draw min values (negative peaks) from right to left
  for (let i = peaksLength - 1; i >= startX; i--) {
    const x = i - startX + 0.5;
    // min values are negative (-128 to 0), draw downward from center
    const y = centerY - (mins[i] / 127) * halfHeight;

    ctx.lineTo(x, y);
  }

  ctx.closePath();
  ctx.fill();

  animationId = requestAnimationFrame(drawWaveform);
};

const subscribePeaks = (sessionID: string) => {
  return reconnectingStream({
    name: `waveformPeaks(${sessionID})`,
    connect: (handlers) =>
      streamWaveformPeaks({
        sessionID,
        onPeaks: (msg) => {
          handlers.onMessage();
          onWaveformPeaks(msg);
        },
        onError: handlers.onError,
        onEnd: handlers.onEnd,
      }),
  });
};

onMounted(() => {
  animationId = requestAnimationFrame(drawWaveform);
  peakStream = subscribePeaks(props.session.id);
});

onUnmounted(() => {
  if (animationId !== null) {
    cancelAnimationFrame(animationId);
  }
  peakStream?.stop();
  if (clippingTimeout) clearTimeout(clippingTimeout);
});

// Re-subscribe if session changes
watch(
  () => props.session.id,
  (newId, oldId) => {
    if (newId !== oldId) {
      minPeaks.value = new Int8Array(0);
      maxPeaks.value = new Int8Array(0);
      peakLevel.value = 0;
      clipping.value = false;

      peakStream?.stop();
      peakStream = subscribePeaks(newId);
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
    <PeakMeter :level="peakLevel" :clipping="clipping" />

  </div>
</template>

<style scoped>
.recording-overview {
  position: relative;
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

@media (max-width: 768px) {
  .recording-overview,
  .recording-overview > * {
    height: 60px;
  }

  .controls {
    width: 60px;
    height: 60px;
  }

  .waveform-canvas {
    height: 60px;
  }
}
</style>
