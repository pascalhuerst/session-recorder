<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { Button } from '@session-recorder/session-waveform';
import type { Session } from '@/types';

defineProps<{
  session: Session;
}>();

// Canvas-based static waveform
const canvasRef = ref<HTMLCanvasElement | null>(null);

// Generate test waveform data (simulated audio samples)
const sampleCount = 200;
const waveformData = Array.from({ length: sampleCount }, (_, i) => {
  // Create varied base amplitudes using multiple sine waves
  const base = 0.3 + Math.sin(i * 0.1) * 0.15 + Math.sin(i * 0.05) * 0.1;
  return base;
});

const drawWaveform = () => {
  const canvas = canvasRef.value;
  if (!canvas) return;

  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const dpr = window.devicePixelRatio || 1;
  const width = canvas.clientWidth;
  const height = canvas.clientHeight;

  // Set canvas resolution for sharp rendering
  canvas.width = width * dpr;
  canvas.height = height * dpr;
  ctx.scale(dpr, dpr);

  const barWidth = 3;
  const gap = 2;
  const totalBarWidth = barWidth + gap;
  const barCount = Math.floor(width / totalBarWidth);
  const centerY = height / 2;
  const maxBarHeight = height * 0.8;

  // Create gradient for bars (muted/grey for processing state)
  const gradient = ctx.createLinearGradient(0, 0, 0, height);
  gradient.addColorStop(0, 'rgba(156, 163, 175, 0.7)');   // grey-400
  gradient.addColorStop(0.5, 'rgba(107, 114, 128, 0.8)'); // grey-500
  gradient.addColorStop(1, 'rgba(156, 163, 175, 0.7)');   // grey-400

  ctx.fillStyle = gradient;

  for (let i = 0; i < barCount; i++) {
    const dataIndex = Math.floor((i / barCount) * sampleCount);
    const amplitude = waveformData[dataIndex];
    const barHeight = Math.max(4, amplitude * maxBarHeight);
    const x = i * totalBarWidth;
    const y = centerY - barHeight / 2;

    // Draw rounded bar
    ctx.beginPath();
    ctx.roundRect(x, y, barWidth, barHeight, barWidth / 2);
    ctx.fill();
  }
};

onMounted(() => {
  drawWaveform();
});
</script>

<template>
  <div class="processing-overview">
    <canvas ref="canvasRef" class="waveform-canvas" />
    <div class="controls">
      <Button
        shape="square"
        size="lg"
        variant="ghost"
        color="primary"
        disabled
        title="Processing..."
      >
        <font-awesome-icon icon="fa-solid fa-stop" />
      </Button>
    </div>
  </div>
</template>

<style scoped>
.processing-overview {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: stretch;
  width: 100%;
  height: 80px;
  border-top: 1px solid var(--color-grey-300);
  border-bottom: 1px solid var(--color-grey-300);
}

.processing-overview > * {
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
