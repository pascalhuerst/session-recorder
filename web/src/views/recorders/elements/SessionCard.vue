<script setup lang="ts">
import { computed } from 'vue';
import SessionMenu from './SessionMenu.vue';
import {
  SegmentState,
  type Session,
} from '@session-recorder/protocols/ts/sessionsource';
import { useDateFormat } from '@vueuse/core';
import { useSessionData } from '@/useSessionData';
import {
  createPeaksContext,
  providePeaksContext,
  WaveformEditor,
} from '@session-recorder/session-waveform';
import { integrateSegments } from '../../../grpc/integrateSegments';

const props = defineProps<{
  session: Session;
  recorderId: string;
  index: number;
}>();

const createdAt = computed(() => {
  if (!props.session.updated.timeCreated) {
    return { iso: '', formatted: '' };
  }
  const format =
    props.session.updated.timeCreated.getFullYear() === new Date().getFullYear()
      ? 'ddd, MMM D, HH:mm'
      : 'MMM D, YYYY HH:mm';
  return {
    iso: props.session.updated.timeCreated.toISOString(),
    formatted: useDateFormat(props.session.updated.timeCreated, format).value,
  };
});

const { waveformUrl, audioUrls } = useSessionData({
  sessionId: props.session.ID,
  recorderId: props.recorderId,
});

const buildUrlPath = (segmentId: string, ext: string) => {
  return `/session-recorder/${props.session.ID}/sessions/${props.recorderId}/segments/${segmentId}.${ext}`;
};

const ctx = createPeaksContext({
  initialState: {
    waveformUrl: waveformUrl.value,
    audioUrls: audioUrls.value as any,
    permissions: {
      create: true,
      update: true,
      delete: true,
    },
    segments: props.session.updated.segments.map((s) => ({
      id: s.segmentID,
      labelText: s.updated.name,
      startTime: s.updated.timeStart.getTime(),
      endTime: s.updated.timeEnd.getTime(),
      renders:
        s.updated.state === SegmentState.SEGMENT_STATE_FINISHED
          ? [
              {
                type: 'audio/mp3',
                src: buildUrlPath(s.segmentID, 'mp3'),
              },
              {
                type: 'audio/ogg',
                src: buildUrlPath(s.segmentID, 'ogg'),
              },
            ]
          : [],
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
        <time class="timestamp" :datetime="createdAt.iso"
          >{{ createdAt.formatted }}
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
