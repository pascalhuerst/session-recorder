<script setup lang="ts">
import { ref } from 'vue';
import {
  createPeaksContext,
  providePeaksContext,
  WaveformView,
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
