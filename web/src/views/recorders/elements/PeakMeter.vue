<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';

const props = defineProps<{
  level: number; // 0.0 - 1.0
  clipping: boolean;
}>();

// Peak hold: tracks the highest recent level with slow decay
const peakHold = ref(0);
let decayInterval: ReturnType<typeof setInterval> | null = null;

watch(
  () => props.level,
  (newLevel) => {
    if (newLevel > peakHold.value) {
      peakHold.value = newLevel;
    }
  }
);

// Decay the peak hold indicator over time
decayInterval = setInterval(() => {
  if (peakHold.value > 0) {
    peakHold.value = Math.max(0, peakHold.value - 0.02);
  }
}, 50);

onUnmounted(() => {
  if (decayInterval) clearInterval(decayInterval);
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
  position: absolute;
  top: 6px;
  right: 6px;
  width: 80px;
  height: 6px;
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
  width: 2px;
  background-color: white;
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
