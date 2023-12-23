<script setup lang="ts">
import { OpenSession } from "../../../grpc/procedures/streamSessions.ts";
import WaveformCanvas from "../../../lib/waveform/WaveformCanvas.vue";
import { computed } from "vue";
import { env } from "../../../env.ts";
import SessionLifetime from "./SessionLifetime.vue";

const props = defineProps<{
  session: OpenSession
  recorderId: string
  index: number
}>();

const waveformUrl = computed(() => {
  return new URL(`${props.recorderId}/${props.session.id}/waveform.dat`, env.VITE_SERVER_URL).toString();
});

const audioUrls = computed(() => {
  return [
    {
      src: new URL(`${props.recorderId}/${props.session.id}/data.ogg`, env.VITE_SERVER_URL).toString(),
      type: "audio/ogg"
    }
  ];
});

const createdAt = computed(() => {
  const dt = new Intl.DateTimeFormat("en-DE", {
    weekday: "short",
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  });

  return dt.format(new Date(props.session.timestamp));
});
</script>

<template>
  <div class="card">
    <div class="heading">
      <span class="index">#{{ props.index }}</span>
      <time class="timestamp" :datetime="props.session.timestamp">{{ createdAt }}</time>
      <div class="lifetime">
        <SessionLifetime :session="session" />
      </div>
    </div>

    <WaveformCanvas :waveform-url="waveformUrl" :audio-urls="audioUrls"></WaveformCanvas>
  </div>
</template>

<style scoped>
.card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--scale-00);
  border-radius: var(--radius-sm);
  box-shadow: rgba(0, 0, 0, 0.05) 0px 6px 24px 0px, rgba(0, 0, 0, 0.08) 0px 0px 0px 1px;
}

.heading {
  position: absolute;
  top: 0;
  left: 0;
  width: calc(100% - calc(2 * var(--size-4)));
  margin-left: var(--size-4);
  margin-top: calc(-1 * var(--size-8));
  display: flex;
  gap: var(--size-2);
}

.heading .index {
  padding: var(--size-2) var(--size-3);
  background-color: var(--color-grey-800);
  color: white;
  border-radius: var(--radius-sm);
  font-size: var(--scale-2);
  font-weight: var(--weight-bold);
}

.heading .timestamp {
  margin-top: var(--size-2);
}

.heading .lifetime {
  margin-top: var(--size-1);
  margin-left: auto;
}
</style>