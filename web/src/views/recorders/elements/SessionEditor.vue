<script setup lang="ts">
import { SessionInfo } from '@session-recorder/protocols/ts/sessionsource';
import { useSessionData } from '../../../useSessionData.ts';
import WaveformEditor from '../../../../../web/session-waveform/src/waveform/WaveformEditor.vue';

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
  ></WaveformEditor>
</template>

<style scoped></style>
