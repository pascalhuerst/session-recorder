<script lang="ts" setup>
import { ref, computed } from 'vue';
import { Modal, Button, TextInput } from '@session-recorder/session-waveform';

defineProps<{
  open: boolean;
  itemName: string;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'share', emails: string[]): void;
}>();

const emailInput = ref('');
const error = ref('');

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const parsedEmails = computed(() => {
  return emailInput.value
    .split(/[,;]/)
    .map((e) => e.trim())
    .filter((e) => e !== '');
});

const validEmails = computed(() => {
  return parsedEmails.value.filter((e) => emailRegex.test(e));
});

const invalidEmails = computed(() => {
  return parsedEmails.value.filter((e) => !emailRegex.test(e));
});

const hasInput = computed(() => emailInput.value.trim() !== '');

const canSubmit = computed(() => {
  return (
    hasInput.value &&
    validEmails.value.length > 0 &&
    invalidEmails.value.length === 0
  );
});

function handleClose() {
  emailInput.value = '';
  error.value = '';
  emit('close');
}

function handleSubmit() {
  if (invalidEmails.value.length > 0) {
    error.value = `Invalid email: ${invalidEmails.value.join(', ')}`;
    return;
  }
  if (validEmails.value.length === 0) {
    error.value = 'Please enter at least one email address';
    return;
  }
  error.value = '';
  emit('share', validEmails.value);
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
          <label for="recipient-email">Recipient Email(s)</label>
          <TextInput
            id="recipient-email"
            v-model="emailInput"
            type="text"
            placeholder="Email addresses"
            size="lg"
            @keydown="handleKeydown"
          />
          <span class="hint">Use comma or semicolon to separate emails</span>
          <span v-if="error" class="error">{{ error }}</span>
          <span v-else-if="validEmails.length > 1" class="hint">
            {{ validEmails.length }} recipients
          </span>
        </div>
      </div>
    </template>

    <template #footer>
      <Button size="md" @click="handleClose">
        Cancel
      </Button>
      <Button
        size="md"
        variant="solid"
        color="primary"
        @click="handleSubmit"
        :disabled="!canSubmit"
      >
        Send
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

.hint {
  color: var(--color-grey-500);
  font-size: var(--scale-0);
}
</style>
