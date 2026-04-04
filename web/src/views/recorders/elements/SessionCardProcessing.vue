<script setup lang="ts">
import { ref, onMounted, onUnmounted, toRefs } from 'vue';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
}>();

const { session } = toRefs(props);

// Canvas-based animated waveform
const canvasRef = ref<HTMLCanvasElement | null>(null);
let animationId: number | null = null;

// Generate base waveform data (simulated audio samples)
const sampleCount = 10000;
const baseWaveformData = Array.from({ length: sampleCount }, (_, i) => {
  // Create varied base amplitudes with clearer peaks
  const base = 0.25;
  const peak1 = Math.sin(i * 0.08) * 0.2;
  const peak2 = Math.sin(i * 0.03) * 0.15;
  const peak3 = Math.sin(i * 0.15) * 0.1;
  const detail = Math.sin(i * 0.5) * 0.05;
  return Math.max(0.08, base + peak1 + peak2 + peak3 + detail);
});

const getAmplitude = (i: number, pointCount: number, time: number) => {
  const dataIndex = Math.floor((i / pointCount) * sampleCount);
  const baseAmplitude = baseWaveformData[dataIndex];
  const dance =
    Math.sin(time * 0.002 + i * 0.05) * 0.08 +
    Math.sin(time * 0.003 + i * 0.03) * 0.05 +
    Math.sin(time * 0.001 + i * 0.08) * 0.03;
  return Math.max(0.05, baseAmplitude + dance);
};

const drawWaveformSection = (
  ctx: CanvasRenderingContext2D,
  from: number,
  to: number,
  pointCount: number,
  time: number,
  centerY: number,
  maxAmplitudeHeight: number,
) => {
  if (from >= to) return;
  ctx.beginPath();

  for (let i = from; i < to; i++) {
    const amplitude = getAmplitude(i, pointCount, time);
    const x = i + 0.5;
    const y = centerY - amplitude * maxAmplitudeHeight + 0.5;
    if (i === from) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  }

  for (let i = to - 1; i >= from; i--) {
    const amplitude = getAmplitude(i, pointCount, time);
    const x = i + 0.5;
    const y = centerY + amplitude * maxAmplitudeHeight + 0.5;
    ctx.lineTo(x, y);
  }

  ctx.closePath();
  ctx.fill();
};

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

  ctx.clearRect(0, 0, width, height);

  const centerY = height / 2;
  const maxAmplitudeHeight = height * 0.4;
  const pointCount = Math.floor(width);
  const progress = session.value.renderProgress ?? 0;
  const splitPoint = Math.floor(progress * pointCount);

  // Completed portion in orange
  if (splitPoint > 0) {
    ctx.fillStyle = '#ed730c';
    drawWaveformSection(ctx, 0, splitPoint, pointCount, time, centerY, maxAmplitudeHeight);
  }

  // Remaining portion in grey
  if (splitPoint < pointCount) {
    ctx.fillStyle = '#c4c9d4';
    drawWaveformSection(ctx, splitPoint, pointCount, pointCount, time, centerY, maxAmplitudeHeight);
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
  <div class="processing-overview">
    <canvas ref="canvasRef" class="waveform-canvas" />
    <div v-if="session.renderProgress" class="progress-bar">
      <div class="progress-fill" :style="{ width: `${Math.round(session.renderProgress * 100)}%` }" />
    </div>
  </div>
</template>

<style scoped>
.processing-overview {
  position: relative;
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: stretch;
  width: 100%;
  height: 80px;
}

.waveform-canvas {
  flex: 1;
  height: 80px;
  border-top: 1px solid var(--border-primary);
  border-bottom: 1px solid var(--border-primary);
}

.progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--color-grey-400);
}

.progress-fill {
  height: 100%;
  background: linear-gradient(
    90deg,
    var(--color-primary, #ed730c) 0%,
    color-mix(in srgb, var(--color-primary, #ed730c), white 40%) 50%,
    var(--color-primary, #ed730c) 100%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  transition: width 0.3s ease;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
