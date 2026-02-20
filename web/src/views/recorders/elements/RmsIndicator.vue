<script setup lang="ts">
defineProps<{
  value: number;
}>();
</script>

<template>
  <div class="status">
    <div class="track">
      <div
        class="indicator"
        :style="{
          'clip-path': `inset(0px ${100 - value}% 0px 0px)`,
        }"
      ></div>
    </div>
    <div class="text">{{ Math.ceil(value) }}%</div>
  </div>
</template>

<style scoped>
.status {
  display: flex;
  gap: var(--size-2);
  align-items: center;
  flex: 1;
  min-width: 0;
}

.text {
  font-size: var(--scale-00);
  color: var(--text-secondary);
  flex-shrink: 0;
}

.indicator {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  transition: clip-path 0.1s ease 0s;
  background-image: linear-gradient(
    to right,
    rgb(0, 136, 0) 0%,
    lime 55%,
    rgb(255, 255, 0) 75%,
    red 95%
  );
}

.track {
  position: relative;
  height: var(--size-1);
  flex: 1;
  min-width: var(--size-10);
  background-color: var(--border-secondary);
  border-radius: var(--radius-xs);
  overflow: hidden;
}
</style>
