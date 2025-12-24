<script setup lang="ts">
import { computed, ref } from 'vue';
import SessionMenu from './SessionMenu.vue';
import { useDateFormat } from '@vueuse/core';
import {
  createPeaksContext,
  providePeaksContext,
  WaveformEditor,
} from '@session-recorder/session-waveform';
import { integrateSegments } from '../../../grpc/integrateSegments';
import { setName } from '../../../grpc/procedures/setName';
import { toastService } from '../../../services/Toaster/ToastService';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
  index: number;
}>();

const titleRef = ref<HTMLElement | null>(null);
const isEditing = ref(false);
const editedName = ref('');

const displayName = computed(() => {
  return props.session.name || `Untitled #${props.index}`;
});

const displayDate = computed(() => {
  const { startedAt } = props.session;
  const format =
    startedAt.getFullYear() === new Date().getFullYear()
      ? 'ddd, MMM D, HH:mm'
      : 'MMM D, YYYY HH:mm';
  return {
    iso: startedAt.toISOString(),
    formatted: useDateFormat(startedAt, format).value,
  };
});

const startEditing = () => {
  isEditing.value = true;
  editedName.value = props.session.name || '';

  // Select all text so user can replace immediately
  requestAnimationFrame(() => {
    const selection = window.getSelection();
    const range = document.createRange();
    if (titleRef.value && selection) {
      range.selectNodeContents(titleRef.value);
      selection.removeAllRanges();
      selection.addRange(range);
    }
  });
};

const cancelEditing = () => {
  isEditing.value = false;
  if (titleRef.value) {
    titleRef.value.textContent = displayName.value;
  }
};

const saveTitle = async () => {
  if (!isEditing.value) return;

  const newName = titleRef.value?.textContent?.trim() || '';
  const oldName = props.session.name || '';

  isEditing.value = false;

  if (newName === oldName || newName === displayName.value) {
    return;
  }

  if (!newName) {
    if (titleRef.value) {
      titleRef.value.textContent = displayName.value;
    }
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
    if (titleRef.value) {
      titleRef.value.textContent = displayName.value;
    }
  }
};

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    event.preventDefault();
    titleRef.value?.blur();
  } else if (event.key === 'Escape') {
    event.preventDefault();
    cancelEditing();
    titleRef.value?.blur();
  }
};

const ctx = createPeaksContext({
  initialState: {
    waveformUrl: props.session.inlineFiles.waveform,
    audioUrls: [
      {
        src: props.session.inlineFiles.ogg,
        type: 'audio/ogg',
      },
      {
        src: props.session.inlineFiles.flac,
        type: 'audio/flac',
      },
    ],
    permissions: {
      create: false,
      update: true,
      delete: true,
    },
    segments: props.session.segments.map((s) => ({
      id: s.id,
      labelText: s.name,
      startTime: s.timeStart.getTime(),
      endTime: s.timeEnd.getTime(),
    })),
  },
});

providePeaksContext(ctx);
integrateSegments(props.session, ctx);
</script>

<template>
  <div class="card">
    <div class="header">
      <span
        ref="titleRef"
        class="title"
        :class="{ editing: isEditing }"
        contenteditable="true"
        @focus="startEditing"
        @blur="saveTitle"
        @keydown="onKeydown"
        >{{ displayName }}</span
      >

      <div class="metadata">
        <time class="timestamp" :datetime="displayDate.iso"
          >{{ displayDate.formatted }}
        </time>
        <div class="menu">
          <SessionMenu :session="session" :recorder-id="recorderId" />
        </div>
      </div>
    </div>

    <WaveformEditor />
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
}

.title {
  font-size: var(--scale-3);
  font-weight: var(--weight-medium);
  cursor: text;
  border-radius: var(--radius-xs);
  padding: var(--size-1) var(--size-2);
  margin: calc(-1 * var(--size-1)) calc(-1 * var(--size-2));
  outline: none;
  border: 1px solid transparent;
  transition: background-color 0.15s ease, border-color 0.15s ease,
    box-shadow 0.15s ease;
}

.title:hover {
  background-color: var(--color-grey-50);
  border-color: var(--color-grey-200);
}

.title:focus,
.title.editing {
  background-color: white;
  border-color: var(--color-purple-500);
  box-shadow: 0 0 0 3px
    color-mix(in srgb, var(--color-purple-300) 25%, transparent);
}

.metadata {
  top: 0;
  left: 0;
  display: flex;
  flex-wrap: nowrap;
  align-items: baseline;
  gap: var(--size-2);
  font-size: var(--scale-1);
}

.timestamp {
  font-size: var(--scale-0);
  font-weight: var(--weight-normal);
  color: var(--color-grey-500);
}

.menu {
  margin-left: auto;
}
</style>
