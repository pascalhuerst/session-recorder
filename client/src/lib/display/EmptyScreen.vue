<script setup lang="ts">

</script>

<template>
  <div class="empty-screen">
    <div v-if="$slots.illustration" class="illustration">
      <slot name="illustration" />
    </div>
    <div v-if="$slots.text" class="text">
      <slot name="text" />
    </div>
  </div>
</template>

<style scoped>
.empty-screen {
  width: 100%;
  height: 60vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: var(--size-6);
}

.illustration {
  font-size: var(--scale-fluid-10);
  color: var(--color-purple-700);
  opacity: 0.2;
}

.text {
  width: 80%;
  max-width: 600px;
  font-size: var(--scale-fluid-5);
  color: var(--color-grey-400);
  text-align: center;
  font-weight: var(--weight-medium);
}
</style>