<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';

const props = defineProps<{
  level: number; // 0.0 - 1.0
  clipping: boolean;
}>();

// Peak hold: tracks the highest recent level with hold time before decay
const peakHold = ref(0);
let decayInterval: ReturnType<typeof setInterval> | null = null;
let holdTimer: ReturnType<typeof setTimeout> | null = null;
let isHolding = false;

watch(
  () => props.level,
  (newLevel) => {
    if (newLevel > peakHold.value) {
      peakHold.value = newLevel;
      isHolding = true;
      if (holdTimer) clearTimeout(holdTimer);
      holdTimer = setTimeout(() => {
        isHolding = false;
      }, 1500);
    }
  }
);

// Decay the peak hold indicator over time (only after hold period)
decayInterval = setInterval(() => {
  if (!isHolding && peakHold.value > 0) {
    peakHold.value = Math.max(0, peakHold.value - 0.02);
  }
}, 50);

onUnmounted(() => {
  if (decayInterval) clearInterval(decayInterval);
  if (holdTimer) clearTimeout(holdTimer);
});
</script>

<template>
  <div class="peak-meter">
    <div class="meter-track">
      <!-- Clipping indicator on the right end -->
      <div class="clip-indicator" :class="{ active: clipping }" />
      <!-- Filled bar showing current level -->
      <div
        class="meter-fill"
        :style="{ clipPath: `inset(0 ${(1 - level) * 100}% 0 0)` }"
      />
      <!-- Peak hold line -->
      <div
        class="peak-hold"
        :style="{ left: `${peakHold * 100}%` }"
      />
    </div>
  </div>
</template>

<style scoped>
.peak-meter {
  margin-left: auto;
  width: 120px;
  height: 4px;
  flex-shrink: 0;
}

.meter-track {
  position: relative;
  width: 100%;
  height: 100%;
  background-color: var(--border-secondary, #333);
  border-radius: 2px;
  overflow: hidden;
}

.meter-fill {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  transition: clip-path 0.05s linear;
  background-image: linear-gradient(
    to right,
    rgb(0, 136, 0) 0%,
    lime 55%,
    rgb(255, 255, 0) 80%,
    red 100%
  );
}

.peak-hold {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background-color: rgba(255, 255, 255, 0.8);
  transition: left 0.05s linear;
}

.clip-indicator {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 0;
  width: 4px;
  background-color: var(--border-secondary, #333);
  z-index: 1;
  transition: background-color 0.1s ease;
}

.clip-indicator.active {
  background-color: red;
}
</style>
