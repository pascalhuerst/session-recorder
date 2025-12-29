<script lang="ts" setup>
import { ref, computed } from 'vue';
import { Modal, Button, TextInput } from '@session-recorder/session-waveform';

const props = defineProps<{
  open: boolean;
  itemName: string;
  isLoading?: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'share', email: string): void;
}>();

const email = ref('');
const error = ref('');

const isValidEmail = computed(() => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email.value);
});

const canSubmit = computed(() => {
  return email.value.trim() !== '' && isValidEmail.value && !props.isLoading;
});

function handleClose() {
  email.value = '';
  error.value = '';
  emit('close');
}

function handleSubmit() {
  if (!isValidEmail.value) {
    error.value = 'Please enter a valid email address';
    return;
  }
  error.value = '';
  emit('share', email.value);
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && canSubmit.value) {
    handleSubmit();
  }
}
</script>

<template>
  <Modal :open="open" :size="{ width: 400 }" @close="handleClose">
    <template #header>Share Recording</template>

    <template #body>
      <div class="share-modal-content">
        <p class="description">
          Send a download link for <strong>{{ itemName }}</strong> via email.
        </p>

        <div class="field">
          <label for="recipient-email">Recipient Email</label>
          <TextInput
            id="recipient-email"
            v-model="email"
            type="email"
            placeholder="Enter email address"
            size="lg"
            :disabled="isLoading"
            @keydown="handleKeydown"
          />
          <span v-if="error" class="error">{{ error }}</span>
        </div>
      </div>
    </template>

    <template #footer>
      <Button size="md" @click="handleClose" :disabled="isLoading">
        Cancel
      </Button>
      <Button
        size="md"
        variant="primary"
        @click="handleSubmit"
        :disabled="!canSubmit"
      >
        {{ isLoading ? 'Sending...' : 'Send' }}
      </Button>
    </template>
  </Modal>
</template>

<style scoped>
.share-modal-content {
  display: flex;
  flex-direction: column;
  gap: var(--size-4);
}

.description {
  color: var(--color-grey-600);
  margin: 0;
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
}

.field label {
  font-weight: var(--weight-medium);
  font-size: var(--scale-1);
}

.error {
  color: var(--color-red-500);
  font-size: var(--scale-0);
}
</style>
