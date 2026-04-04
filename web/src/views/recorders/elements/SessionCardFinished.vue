<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import {
  createPeaksContext,
  providePeaksContext,
  WaveformView,
  getSegmentColor,
  intToChar,
} from '@session-recorder/session-waveform';
import { integrateSegments } from '../../../grpc/integrateSegments';
import { shareSegment } from '../../../grpc/procedures/shareSegment';
import { toastService } from '../../../services/Toaster/ToastService';
import ShareModal from '../../../components/ShareModal.vue';
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

// Share segment modal state
const showShareModal = ref(false);
const sharingSegmentId = ref<string | null>(null);

const sharingSegmentName = computed(() => {
  if (!sharingSegmentId.value) return 'Segment';
  const segment = props.session.segments.find((s) => s.id === sharingSegmentId.value);
  return segment?.name || 'Segment';
});

// Handle share segment command from the waveform library
ctx.commandEmitter.on('shareSegment', (segmentId: string) => {
  sharingSegmentId.value = segmentId;
  showShareModal.value = true;
});

const onShareClose = () => {
  showShareModal.value = false;
  sharingSegmentId.value = null;
};

const onShareConfirm = (emails: string[]) => {
  if (!sharingSegmentId.value) return;

  const segmentId = sharingSegmentId.value;
  const recipientText = emails.length === 1 ? emails[0] : `${emails.length} recipients`;

  // Close dialog immediately and show info toast
  showShareModal.value = false;
  sharingSegmentId.value = null;
  toastService.info(`Sending download link to ${recipientText}...`);

  // Send in background
  shareSegment({
    recorderId: props.recorderId,
    sessionId: props.session.id,
    segmentId: segmentId,
    recipientEmails: emails,
  })
    .then(() => {
      toastService.success(`Download link sent to ${recipientText}`);
    })
    .catch((error) => {
      console.error('Failed to share segment:', error);
      toastService.error('Failed to send email. Please try again.');
    });
};

// Sync segment state changes from session to peaks context
// This is needed because the peaks context has its own copy of segments
// and needs to be updated when the backend broadcasts state changes.
// We derive a lightweight key from just the fields we care about (state, errorMessage,
// downloadFiles) to avoid deep-watching the entire segments array.
watch(
  () =>
    props.session.segments
      .map((s) => `${s.id}:${s.state}:${s.errorMessage ?? ''}:${s.downloadFiles?.ogg ?? ''}`)
      .join('|'),
  () => {
    const newSegments = props.session.segments;
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
  }
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

  <ShareModal
    :open="showShareModal"
    :item-name="sharingSegmentName"
    @close="onShareClose"
    @share="onShareConfirm"
  />
</template>
