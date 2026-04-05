<script setup lang="ts">
import type { Session } from '@/types';

defineProps<{
  session: Session;
}>();
</script>

<template>
  <div class="processing-overview">
    <span class="processing-label">Processing…</span>
  </div>
</template>

<style scoped>
.processing-overview {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 80px;
  border-top: 1px solid var(--border-primary);
  border-bottom: 1px solid var(--border-primary);
}

.processing-label {
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  color: var(--color-grey-500);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
