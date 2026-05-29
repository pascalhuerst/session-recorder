<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, nextTick } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import SessionMenu from './SessionMenu.vue';
import StatusIndicator from './StatusIndicator.vue';
import SessionCardRecording from './SessionCardRecording.vue';
import SessionCardProcessing from './SessionCardProcessing.vue';
import SessionCardError from './SessionCardError.vue';
import SessionCardFinished from './SessionCardFinished.vue';
import { Button, Modal, useConfirmation } from '@session-recorder/session-waveform';
import { useDateFormat } from '@vueuse/core';
import { deleteSession } from '../../../grpc/procedures/deleteSession';
import { toastService } from '../../../services/Toaster/ToastService';
import { setName } from '../../../grpc/procedures/setName';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
  index: number;
}>();

const inputRef = ref<HTMLInputElement | null>(null);
const finishedCardRef = ref<InstanceType<typeof SessionCardFinished> | null>(null);
const isEditing = ref(false);
const editedName = ref('');

// Allow editing for recording, processing, and finished sessions
const canEdit = computed(() => props.session.state !== 'error');

const displayName = computed(() => {
  return props.session.name || `Untitled #${props.index}`;
});

const timeRange = computed(() => {
  const { startedAt, finishedAt } = props.session;
  const formatTime = (date: Date) => useDateFormat(date, 'HH:mm').value;
  const start = formatTime(startedAt);
  if (finishedAt) {
    const end = formatTime(finishedAt);
    return `${start}–${end}`;
  }
  return start;
});

// State checks
const isRecording = computed(() => props.session.state === 'recording');
const isProcessing = computed(() => props.session.state === 'processing');
const now = ref(new Date());
let interval: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  if (isRecording.value) {
    interval = setInterval(() => {
      now.value = new Date();
    }, 1000);
  }
});

onUnmounted(() => {
  if (interval) {
    clearInterval(interval);
  }
});

const elapsedTime = computed(() => {
  if (!isRecording.value) return '';
  const diff = now.value.getTime() - props.session.startedAt.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  const pad = (n: number) => n.toString().padStart(2, '0');

  if (hours > 0) {
    return `${pad(hours)}:${pad(minutes % 60)}:${pad(seconds % 60)}`;
  }
  return `${pad(minutes)}:${pad(seconds % 60)}`;
});

const startEditing = () => {
  if (!canEdit.value) return;

  // Edit in a dedicated <input> bound to local state. The display title is a
  // {{ }} interpolation, which must NOT be made contenteditable: Vue re-renders
  // that text node on every incoming session update and would wipe the user's
  // keystrokes mid-edit. editedName is local and survives those re-renders.
  editedName.value = props.session.name || '';
  isEditing.value = true;

  nextTick(() => {
    inputRef.value?.focus();
    inputRef.value?.select();
  });
};

const cancelEditing = () => {
  isEditing.value = false;
  editedName.value = '';
};

const saveTitle = async () => {
  if (!isEditing.value) return;

  isEditing.value = false;

  const newName = editedName.value.trim();
  const oldName = props.session.name || '';

  if (!newName || newName === oldName) {
    return;
  }

  try {
    await setName({
      recorderId: props.recorderId,
      sessionId: props.session.id,
      name: newName,
    });
    toastService.success('Session renamed successfully');
  } catch (error) {
    console.error('Failed to rename session:', error);
    toastService.error('Failed to rename session');
  }
};

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    event.preventDefault();
    inputRef.value?.blur(); // blur triggers saveTitle
  } else if (event.key === 'Escape') {
    event.preventDefault();
    cancelEditing();
    inputRef.value?.blur();
  }
};

// Only create waveform context for finished sessions with files
const isFinished = computed(() => props.session.state === 'finished');
const hasWaveformFiles = computed(() => !!props.session.inlineFiles);

// Track expanded state for UI - get from child component
const isExpanded = computed(() => finishedCardRef.value?.isExpanded ?? false);

const toggleExpanded = () => {
  finishedCardRef.value?.toggleExpanded();
};

// Delete confirmation for processing sessions
const { awaitConfirmation, modalProps } = useConfirmation();

const onDeleteProcessing = () => {
  awaitConfirmation().then(({ isConfirmed }) => {
    if (isConfirmed) {
      deleteSession({
        recorderId: props.recorderId,
        sessionId: props.session.id,
      })
        .then(() => {
          toastService.success('Session deleted successfully');
        })
        .catch((error) => {
          console.error('Failed to delete session:', error);
          toastService.error('Failed to delete session');
        });
    }
  });
};
</script>

<template>
  <div class="card" :class="[session.state, { expanded: isExpanded }]">
    <div class="header">
      <!-- Show expand toggle only for finished sessions -->
      <button
        v-if="isFinished"
        class="expand-toggle"
        :class="{ expanded: isExpanded }"
        :aria-expanded="isExpanded"
        aria-label="Toggle session details"
        @click="toggleExpanded"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M6 4L10 8L6 12"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>

      <span class="time-range">{{ timeRange }}</span>

      <input
        v-if="isEditing"
        ref="inputRef"
        v-model="editedName"
        type="text"
        class="title title-input editing"
        @blur="saveTitle"
        @keydown="onKeydown"
      />
      <span
        v-else
        class="title"
        :class="{ readonly: !canEdit }"
        @click="startEditing"
        >{{ displayName }}</span
      >

      <!-- Show status indicator for recording/processing sessions -->
      <StatusIndicator v-if="isRecording" :is-recording="true" />
      <StatusIndicator v-else-if="isProcessing" :is-processing="true" />

      <!-- Elapsed time for recording sessions -->
      <span v-if="isRecording" class="elapsed-time">{{ elapsedTime }}</span>

      <div class="spacer"></div>

      <!-- Actions -->
      <div class="actions">
        <Button
          v-if="isProcessing || session.state === 'error'"
          size="xs"
          @click="onDeleteProcessing"
        >
          <font-awesome-icon icon="fa-solid fa-trash" />
          Delete
        </Button>
        <SessionMenu v-if="isFinished" :session="session" :recorder-id="recorderId" />
      </div>
    </div>

    <!-- State-specific content -->
    <SessionCardRecording
      v-if="session.state === 'recording'"
      :session="session"
      :recorder-id="recorderId"
    />
    <SessionCardProcessing
      v-else-if="session.state === 'processing'"
      :session="session"
    />
    <SessionCardError
      v-else-if="session.state === 'error'"
      :session="session"
      :recorder-id="recorderId"
    />
    <SessionCardFinished
      v-else-if="isFinished && hasWaveformFiles"
      ref="finishedCardRef"
      :session="session"
      :recorder-id="recorderId"
    />

    <!-- Delete confirmation modal for processing sessions -->
    <Modal :open="modalProps.open.value" @close="modalProps.onClose">
      <template #header>Are you sure?</template>
      <template #body>
        You are about to permanently delete a session that is still processing.
      </template>
      <template #footer>
        <Button @click="modalProps.onConfirm" variant="ghost" color="neutral">
          Delete
        </Button>
        <Button @click="modalProps.onClose" variant="solid" color="primary">
          Keep
        </Button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
}

.header {
  width: 100%;
  padding: var(--size-1);
  display: flex;
  align-items: center;
  gap: var(--size-2);
}

.title {
  font-size: var(--scale-2);
  font-weight: var(--weight-semibold);
  border-radius: var(--radius-xs);
  cursor: text;
  outline: 1px solid transparent;
  outline-offset: var(--size-1);
  transition: background-color 0.15s ease, border-color 0.15s ease,
    box-shadow 0.15s ease;
}

.title:hover {
  outline-color: var(--border-primary);
}

.title:focus,
.title.editing {
  outline-color: var(--accent);
}

/* The edit <input> reuses .title for size/weight but must shed the native
   input chrome so it blends with the display span. */
.title-input {
  border: none;
  background: transparent;
  padding: 0;
  margin: 0;
  font-family: inherit;
  color: inherit;
  min-width: 0;
}

.title.readonly {
  cursor: default;
}

.title.readonly:hover {
  outline-color: transparent;
}

.time-range {
  font-size: var(--scale-0);
  font-weight: var(--weight-normal);
  color: var(--text-muted);
  white-space: nowrap;
}

.elapsed-time {
  font-family: monospace;
  font-size: var(--scale-1);
  font-weight: var(--weight-semibold);
  color: var(--color-red-500);
}

.spacer {
  flex: 1;
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--size-1);
}

.expand-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--size-5);
  height: var(--size-5);
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: var(--radius-xs);
  color: var(--text-muted);
  transition: transform 0.2s ease, color 0.15s ease, background-color 0.15s ease;
}

.expand-toggle:hover {
  background-color: var(--bg-hover);
  color: var(--text-primary);
}

.expand-toggle.expanded {
  transform: rotate(90deg);
}
</style>
