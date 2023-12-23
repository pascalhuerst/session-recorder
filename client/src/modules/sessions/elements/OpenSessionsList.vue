<script setup lang="ts">
import { useRoute } from "vue-router";
import { computed, ref, watch } from "vue";
import { streamSessions } from "../../../grpc/procedures/streamSessions.ts";
import SessionCard from "./SessionCard.vue";
import { SessionInfo} from '@session-recorder/protocols/ts/sessionsource.ts';

const route = useRoute();

const sessions = ref<Set<SessionInfo>>(new Set);
const selectedRecorderId = computed(() => route.params.recorderId as string);

watch(selectedRecorderId, () => {
  sessions.value.clear();

  streamSessions({
    request: {
      recorderID: selectedRecorderId.value
    },
    onMessage: (session) => {
      sessions.value.add(session);
    }
  });
}, {
  immediate: true
});
</script>

<template>
  <div class="list">
    <template v-for="(session, index) in sessions" :key="session.id">
      <SessionCard :session="session" :recorder-id="selectedRecorderId" :index="sessions.size - index" />
    </template>
  </div>
</template>

<style scoped>
.list {
  display: flex;
  flex-direction: column;
  gap: var(--size-16);
}
</style>