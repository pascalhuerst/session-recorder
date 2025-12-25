<script setup lang="ts">
import type { Session } from '@/types';

defineProps<{
  session: Session;
}>();
</script>

<template>
  <div class="processing-content">
    <div class="spinner-container">
      <div class="spinner" />
    </div>
    <p class="message">Processing audio...</p>
    <p class="hint">This may take a few moments</p>
  </div>
</template>

<style scoped>
.processing-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--size-2);
  padding: var(--size-4);
  min-height: 120px;
}

.spinner-container {
  display: flex;
  align-items: center;
  justify-content: center;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-grey-200);
  border-top-color: var(--color-yellow-500);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.message {
  font-size: var(--scale-1);
  font-weight: var(--weight-medium);
  color: var(--color-grey-700);
  margin: 0;
}

.hint {
  font-size: var(--scale-0);
  color: var(--color-grey-500);
  margin: 0;
}
</style>
