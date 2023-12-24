<script setup lang="ts">
import { SessionInfo } from "@session-recorder/protocols/ts/sessionsource";
import { useSessionData } from "../../../utils/useSessionData.ts";
import WaveformEditor from "../../../lib/waveform/WaveformEditor.vue";

const props = defineProps<{
  session: SessionInfo,
  recorderId: string
}>();

const { waveformUrl, audioUrls } = useSessionData({ sessionId: props.session.ID, recorderId: props.recorderId });
</script>

<template>
  <WaveformEditor :waveform-url="waveformUrl" :audio-urls="audioUrls"></WaveformEditor>
</template>

<style scoped>
</style>