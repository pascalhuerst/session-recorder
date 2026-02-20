<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import type { Session } from '@/types';

defineProps<{
  session: Session;
  recorderId: string;
}>();

const canvasRef = ref<HTMLCanvasElement | null>(null);

// Draw a static waveform placeholder
const drawPlaceholder = () => {
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

  const centerY = height / 2;
  const maxAmplitudeHeight = height * 0.4;

  // Subtle placeholder color so error message pops
  ctx.fillStyle = 'rgba(148, 163, 184, 0.15)';

  // Generate a static waveform pattern
  ctx.beginPath();

  // Forward pass: draw top edge
  for (let x = 0; x < width; x++) {
    // Create a pseudo-random but consistent pattern
    const noise = Math.sin(x * 0.05) * 0.3 + Math.sin(x * 0.12) * 0.2 + Math.sin(x * 0.03) * 0.5;
    const amplitude = Math.abs(noise) * 0.6 + 0.1;
    const y = centerY - amplitude * maxAmplitudeHeight;

    if (x === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  }

  // Reverse pass: draw bottom edge
  for (let x = width - 1; x >= 0; x--) {
    const noise = Math.sin(x * 0.05) * 0.3 + Math.sin(x * 0.12) * 0.2 + Math.sin(x * 0.03) * 0.5;
    const amplitude = Math.abs(noise) * 0.6 + 0.1;
    const y = centerY + amplitude * maxAmplitudeHeight;

    ctx.lineTo(x, y);
  }

  ctx.closePath();
  ctx.fill();
};

onMounted(() => {
  drawPlaceholder();
});
</script>

<template>
  <div class="error-overview">
    <div class="waveform-placeholder">
      <canvas ref="canvasRef" class="placeholder-canvas" />
      <div class="error-message">
        <font-awesome-icon icon="fa-solid fa-circle-exclamation" class="error-icon" />
        <span>{{ session.errorMessage || 'An error occurred while processing this session' }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.error-overview {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: stretch;
  width: 100%;
  height: 80px;
}

.waveform-placeholder {
  flex: 1;
  min-width: 0;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  border-top: 1px solid var(--border-primary);
  border-bottom: 1px solid var(--border-primary);
}

.placeholder-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.error-message {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--size-2);
  padding: var(--size-1) var(--size-2);
  font-size: var(--scale-0);
  color: var(--color-red-700);
  background: var(--color-red-50);
  border-radius: var(--radius-xs);
}

:global(.theme-dark) .error-message {
  color: var(--color-red-300);
  background: var(--color-red-950);
}

.error-icon {
  flex-shrink: 0;
}
</style>
