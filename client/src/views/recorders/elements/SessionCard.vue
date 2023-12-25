<script setup lang="ts">
import { computed } from "vue";
import SessionMenu from "./SessionMenu.vue";
import { SessionInfo } from "@session-recorder/protocols/ts/sessionsource";
import { useSessionData } from "../../../utils/useSessionData.ts";
import { useDateFormat } from "@vueuse/core";
import WaveformEditor from "../../../lib/waveform/WaveformEditor.vue";

const props = defineProps<{
  session: SessionInfo,
  recorderId: string
  index: number
}>();


const createdAt = computed(() => {
  if (!props.session.timeCreated) {
    return { iso: "", formatted: "" };
  }
  const format = props.session.timeCreated.getFullYear() === new Date().getFullYear() ? "ddd, MMM D, HH:mm" : "MMM D, YYYY HH:mm";
  return {
    iso: props.session.timeCreated.toISOString(),
    formatted: useDateFormat(props.session.timeCreated, format).value
  };
});

const { waveformUrl, audioUrls } = useSessionData({ sessionId: props.session.ID, recorderId: props.recorderId });
</script>

<template>
  <div class="card">
    <div class="header">
      <router-link :to="`/recorders/${recorderId}/sessions/${session.ID}`" class="name">
        <span>Untitled #{{ props.index }}</span>
      </router-link>

      <div class="metadata">
        <time class="timestamp" :datetime="createdAt.iso">{{ createdAt.formatted }}</time>
        <div class="menu">
          <SessionMenu :session="session" :recorder-id="recorderId" />
        </div>
      </div>
    </div>

    <WaveformEditor :waveform-url="waveformUrl" :audio-urls="audioUrls"></WaveformEditor>
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