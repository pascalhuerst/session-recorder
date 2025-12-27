<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue';

const props = withDefaults(
  defineProps<{
    startTime: number;
    endTime: number;
    waveformUrl: string;
    duration: number;
    height?: number;
    color?: string;
  }>(),
  {
    height: 40,
  }
);

const canvasRef = ref<HTMLCanvasElement | null>(null);
const waveformData = ref<{ min: number[]; max: number[] } | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Parse Peaks.js binary waveform format
async function parseWaveformData(buffer: ArrayBuffer) {
  const view = new DataView(buffer);

  // Header format (20 bytes):
  // - version (int32): 1 or 2
  // - flags (uint32): bit 0 = 8-bit (0) or 16-bit (1)
  // - sample_rate (int32)
  // - samples_per_pixel (int32)
  // - length (uint32): number of min/max pairs

  // Header: version (int32), flags (uint32), sample_rate (int32), samples_per_pixel (int32), length (uint32)
  const flags = view.getUint32(4, true);
  const length = view.getUint32(16, true);

  const is16Bit = (flags & 1) === 1;
  const bytesPerSample = is16Bit ? 2 : 1;
  const headerSize = 20;

  const min: number[] = [];
  const max: number[] = [];

  for (let i = 0; i < length; i++) {
    const offset = headerSize + i * bytesPerSample * 2;

    if (is16Bit) {
      min.push(view.getInt16(offset, true) / 32768);
      max.push(view.getInt16(offset + 2, true) / 32768);
    } else {
      min.push(view.getInt8(offset) / 128);
      max.push(view.getInt8(offset + 1) / 128);
    }
  }

  return { min, max };
}

// Calculate which samples to display
const sampleRange = computed(() => {
  if (!waveformData.value || props.duration <= 0) {
    return { startSample: 0, endSample: 0 };
  }

  const totalSamples = waveformData.value.min.length;
  const startRatio = props.startTime / props.duration;
  const endRatio = props.endTime / props.duration;

  return {
    startSample: Math.floor(startRatio * totalSamples),
    endSample: Math.ceil(endRatio * totalSamples),
  };
});

function drawWaveform() {
  const canvas = canvasRef.value;
  if (!canvas || !waveformData.value) return;

  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const { startSample, endSample } = sampleRange.value;
  const segmentSamples = endSample - startSample;

  if (segmentSamples <= 0) return;

  // Set canvas size with device pixel ratio for sharp rendering
  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.getBoundingClientRect();
  canvas.width = rect.width * dpr;
  canvas.height = rect.height * dpr;
  ctx.scale(dpr, dpr);

  const width = rect.width;
  const height = rect.height;
  const centerY = height / 2;

  // Clear canvas
  ctx.clearRect(0, 0, width, height);

  // Draw background
  ctx.fillStyle = 'var(--color-grey-100, #f3f4f6)';
  ctx.fillRect(0, 0, width, height);

  // Draw waveform using segment color or fallback
  ctx.fillStyle = props.color || 'var(--color-grey-400, #9ca3af)';

  const samplesPerPixel = segmentSamples / width;

  for (let x = 0; x < width; x++) {
    const sampleStart = startSample + Math.floor(x * samplesPerPixel);
    const sampleEnd = Math.min(
      startSample + Math.ceil((x + 1) * samplesPerPixel),
      endSample
    );

    // Find min/max in this pixel's sample range
    let pixelMin = 0;
    let pixelMax = 0;

    for (let s = sampleStart; s < sampleEnd; s++) {
      if (s < waveformData.value!.min.length) {
        pixelMin = Math.min(pixelMin, waveformData.value!.min[s]);
        pixelMax = Math.max(pixelMax, waveformData.value!.max[s]);
      }
    }

    // Scale to canvas height (with some padding)
    const scale = (height / 2) * 0.9;
    const y1 = centerY - pixelMax * scale;
    const y2 = centerY - pixelMin * scale;
    const barHeight = Math.max(1, y2 - y1);

    ctx.fillRect(x, y1, 1, barHeight);
  }
}

async function loadWaveform() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await fetch(props.waveformUrl);
    if (!response.ok) {
      throw new Error(`Failed to fetch waveform: ${response.status}`);
    }

    const buffer = await response.arrayBuffer();
    waveformData.value = await parseWaveformData(buffer);
    drawWaveform();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load waveform';
  } finally {
    isLoading.value = false;
  }
}

// Redraw when segment times or color change
watch([() => props.startTime, () => props.endTime, () => props.color], () => {
  if (waveformData.value) {
    drawWaveform();
  }
});

// Handle resize
const resizeObserver = ref<ResizeObserver | null>(null);

onMounted(() => {
  loadWaveform();

  if (canvasRef.value) {
    resizeObserver.value = new ResizeObserver(() => {
      if (waveformData.value) {
        drawWaveform();
      }
    });
    resizeObserver.value.observe(canvasRef.value);
  }
});
</script>

<template>
  <div class="waveform-preview" :style="{ height: `${height}px` }">
    <div v-if="isLoading" class="loading">
      <span class="loading-dot"></span>
    </div>
    <div v-else-if="error" class="error" :title="error">
      <span>!</span>
    </div>
    <canvas v-else ref="canvasRef" class="canvas"></canvas>
  </div>
</template>

<style scoped>
.waveform-preview {
  position: relative;
  width: 100%;
  border-radius: var(--radius-sm, 4px);
  overflow: hidden;
  background: var(--color-grey-100, #f3f4f6);
}

.canvas {
  width: 100%;
  height: 100%;
  display: block;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.loading-dot {
  width: 8px;
  height: 8px;
  background: var(--color-grey-400);
  border-radius: 50%;
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 0.4;
  }
  50% {
    opacity: 1;
  }
}

.error {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  color: var(--color-grey-400);
  font-size: 0.75rem;
}
</style>
