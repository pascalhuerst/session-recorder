<script setup lang="ts">
import { SessionInfo } from '@session-recorder/protocols/ts/sessionsource';
import { useSessionData } from '@/useSessionData';
import { WaveformEditor } from '@session-recorder/session-waveform';

const props = defineProps<{
  session: SessionInfo;
  recorderId: string;
}>();

const { waveformUrl, audioUrls } = useSessionData({
  sessionId: props.session.ID,
  recorderId: props.recorderId,
});
</script>

<template>
  <WaveformEditor
    :waveform-url="waveformUrl"
    :audio-urls="audioUrls"
    :segments="[]"
    :permissions="{
      create: true,
      update: true,
      delete: true,
    }"
  ></WaveformEditor>
</template>

<style scoped></style>
