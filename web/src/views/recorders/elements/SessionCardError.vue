<script setup lang="ts">
import { ref } from 'vue';
import { Button } from '@session-recorder/session-waveform';
import { deleteSession } from '../../../grpc/procedures/deleteSession';
import { toastService } from '../../../services/Toaster/ToastService';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const isDeleting = ref(false);

const handleDelete = async () => {
  if (isDeleting.value) return;
  isDeleting.value = true;

  try {
    await deleteSession({
      recorderID: props.recorderId,
      sessionID: props.session.id,
    });
    toastService.success('Session deleted');
  } catch (error) {
    console.error('Failed to delete session:', error);
    toastService.error('Failed to delete session');
  } finally {
    isDeleting.value = false;
  }
};
</script>

<template>
  <div class="error-content">
    <div class="error-icon">
      <svg
        width="32"
        height="32"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" />
        <path
          d="M12 8V12M12 16H12.01"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
        />
      </svg>
    </div>
    <p class="error-message">{{ session.errorMessage || 'An error occurred while processing this session' }}</p>
    <div class="actions">
      <Button
        variant="secondary"
        :disabled="isDeleting"
        @click="handleDelete"
      >
        {{ isDeleting ? 'Deleting...' : 'Delete Session' }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.error-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--size-2);
  padding: var(--size-4);
  min-height: 120px;
  background: var(--color-red-50);
  border-radius: var(--radius-sm);
}

.error-icon {
  color: var(--color-red-500);
}

.error-message {
  font-size: var(--scale-1);
  color: var(--color-red-700);
  text-align: center;
  margin: 0;
  max-width: 400px;
}

.actions {
  display: flex;
  gap: var(--size-2);
  margin-top: var(--size-1);
}
</style>
