<script setup lang="ts">
import { computed } from 'vue';
import SessionMenu from './SessionMenu.vue';
import { useDateFormat } from '@vueuse/core';
import {
  createPeaksContext,
  providePeaksContext,
  WaveformEditor,
} from '@session-recorder/session-waveform';
import { integrateSegments } from '../../../grpc/integrateSegments';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
  index: number;
}>();

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
      <span>Untitled #{{ props.index }}</span>

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

.metadata {
  top: 0;
  left: 0;
  display: flex;
  flex-wrap: nowrap;
  align-items: baseline;
  gap: var(--size-2);
  font-size: var(--scale-1);
}

.name {
  font-size: var(--scale-1);
  font-weight: var(--weight-bold);
  color: var(--color-purple-700);
  text-decoration: none;
}

.timestamp {
  font-size: var(--scale-2);
  font-weight: var(--weight-semibold);
}

.menu {
  margin-left: auto;
}
</style>
