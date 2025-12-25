<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { Button } from '@session-recorder/session-waveform';
import { cutSession } from '../../../grpc/procedures/cutSession';
import { toastService } from '../../../services/Toaster/ToastService';
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

// Canvas-based animated waveform
const canvasRef = ref<HTMLCanvasElement | null>(null);
let animationId: number | null = null;

// Generate test waveform data (simulated audio samples)
const sampleCount = 200;
const waveformData = Array.from({ length: sampleCount }, (_, i) => {
  // Create varied base amplitudes using multiple sine waves
  const base = 0.3 + Math.sin(i * 0.1) * 0.15 + Math.sin(i * 0.05) * 0.1;
  return base;
});

const drawWaveform = (time: number) => {
  const canvas = canvasRef.value;
  if (!canvas) return;

  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const dpr = window.devicePixelRatio || 1;
  const width = canvas.clientWidth;
  const height = canvas.clientHeight;

  // Set canvas resolution for sharp rendering
  if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);
  }

  // Clear canvas
  ctx.clearRect(0, 0, width, height);

  const barWidth = 3;
  const gap = 2;
  const totalBarWidth = barWidth + gap;
  const barCount = Math.floor(width / totalBarWidth);
  const centerY = height / 2;
  const maxBarHeight = height * 0.8;

  // Create gradient for bars
  const gradient = ctx.createLinearGradient(0, 0, 0, height);
  gradient.addColorStop(0, 'rgba(239, 68, 68, 0.9)');   // red-500
  gradient.addColorStop(0.5, 'rgba(220, 38, 38, 0.95)'); // red-600
  gradient.addColorStop(1, 'rgba(239, 68, 68, 0.9)');   // red-500

  ctx.fillStyle = gradient;

  for (let i = 0; i < barCount; i++) {
    const dataIndex = Math.floor((i / barCount) * sampleCount);
    const baseAmplitude = waveformData[dataIndex];

    // Animate amplitude with time-based sine waves
    const animatedAmplitude = baseAmplitude * (
      0.6 +
      Math.sin(time * 0.002 + i * 0.15) * 0.25 +
      Math.sin(time * 0.003 + i * 0.08) * 0.15
    );

    const barHeight = Math.max(4, animatedAmplitude * maxBarHeight);
    const x = i * totalBarWidth;
    const y = centerY - barHeight / 2;

    // Draw rounded bar
    ctx.beginPath();
    ctx.roundRect(x, y, barWidth, barHeight, barWidth / 2);
    ctx.fill();
  }

  animationId = requestAnimationFrame(drawWaveform);
};

onMounted(() => {
  animationId = requestAnimationFrame(drawWaveform);
});

onUnmounted(() => {
  if (animationId !== null) {
    cancelAnimationFrame(animationId);
  }
});
</script>

<template>
  <div class="recording-overview">
    <canvas ref="canvasRef" class="waveform-canvas" />
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
  border-top: 1px solid var(--color-grey-300);
  border-bottom: 1px solid var(--color-grey-300);
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
  height: 80px;
}
</style>
