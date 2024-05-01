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
          'clip-path': `inset(0px ${100 - value * 100}% 0px 0px)`,
        }"
      ></div>
    </div>
    <div class="text">{{ Math.ceil(value * 100) }}%</div>
  </div>
</template>

<style scoped>
.status {
  display: flex;
  gap: var(--size-1);
  align-items: center;
}

.text {
  font-size: var(--size-3);
  color: var(--color-grey-800);
}

.indicator {
  position: relative;
  transition: clip-path 0.1s ease 0s;
  height: var(--size-1);
  width: 100%;
  background-image: linear-gradient(
    to left,
    red 5%,
    rgb(255, 255, 0) 25%,
    lime 45%,
    rgb(0, 136, 0) 100%
  );
}

.track {
  height: var(--size-1);
  width: 100%;
  background-color: var(--color-grey-200);
}
</style>
