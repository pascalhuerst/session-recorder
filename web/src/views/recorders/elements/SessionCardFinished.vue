<script setup lang="ts">
import { ref, watch } from 'vue';
import {
  createPeaksContext,
  providePeaksContext,
  WaveformView,
  getSegmentColor,
  intToChar,
} from '@session-recorder/session-waveform';
import { integrateSegments } from '../../../grpc/integrateSegments';
import type { Session } from '@/types';

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const waveformRef = ref<InstanceType<typeof WaveformView> | null>(null);
const isExpanded = ref(false);

// Create peaks context - this component only mounts for finished sessions with files
const ctx = createPeaksContext({
  initialState: {
    waveformUrl: props.session.inlineFiles!.waveform,
    audioUrls: [
      {
        src: props.session.inlineFiles!.ogg,
        type: 'audio/ogg',
      },
      {
        src: props.session.inlineFiles!.flac,
        type: 'audio/flac',
      },
    ],
    expanded: false,
    permissions: {
      create: true,
      update: true,
      delete: true,
    },
    segments: props.session.segments.map((s, index) => {
      // Generate startIndex/endIndex for segment labeling (A-B, C-D, etc.)
      const startChar = intToChar(index * 2);
      const endChar = intToChar(index * 2 + 1);
      // Convert absolute timestamps to seconds offset from session start (Peaks.js uses seconds)
      // Clamp to 0 to handle clock drift or timestamps recorded before session start
      const sessionStartMs = props.session.startedAt.getTime();
      return {
        id: s.id,
        labelText: s.name,
        startTime: Math.max(0, (s.timeStart.getTime() - sessionStartMs) / 1000),
        endTime: Math.max(0, (s.timeEnd.getTime() - sessionStartMs) / 1000),
        color: getSegmentColor(index),
        state: s.state,
        errorMessage: s.errorMessage,
        startIndex: startChar,
        endIndex: endChar,
        renders:
          s.state === 'finished' && s.downloadFiles
            ? [
                { type: 'audio/ogg', src: s.downloadFiles.ogg },
                { type: 'audio/flac', src: s.downloadFiles.flac },
              ]
            : [],
      };
    }),
  },
});

providePeaksContext(ctx);
integrateSegments(props.session, props.recorderId, ctx);

// Sync segment state changes from session to peaks context
// This is needed because the peaks context has its own copy of segments
// and needs to be updated when the backend broadcasts state changes
watch(
  () => props.session.segments,
  (newSegments) => {
    ctx.state.update((state) => ({
      ...state,
      segments: state.segments.map((ctxSegment) => {
        const sessionSegment = newSegments.find((s) => s.id === ctxSegment.id);
        if (!sessionSegment) {
          return ctxSegment;
        }
        return {
          ...ctxSegment,
          state: sessionSegment.state,
          errorMessage: sessionSegment.errorMessage,
          renders:
            sessionSegment.state === 'finished' && sessionSegment.downloadFiles
              ? [
                  { type: 'audio/ogg', src: sessionSegment.downloadFiles.ogg },
                  { type: 'audio/flac', src: sessionSegment.downloadFiles.flac },
                ]
              : ctxSegment.renders,
        };
      }),
    }));
  },
  { deep: true }
);

ctx.eventEmitter.on('expandedChanged', (expanded) => {
  isExpanded.value = expanded;
});

const toggleExpanded = () => {
  waveformRef.value?.toggleExpanded();
};

defineExpose({
  toggleExpanded,
  isExpanded,
});
</script>

<template>
  <WaveformView ref="waveformRef" />
</template>
