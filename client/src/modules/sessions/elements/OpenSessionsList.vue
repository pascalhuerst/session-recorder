<script setup lang="ts">
import { useRoute } from "vue-router";
import { useAsyncState } from "@vueuse/core";
import { computed, watch } from "vue";
import { getOpenSessions } from "../../../client/getOpenSessions.ts";
import SessionCard from "./SessionCard.vue";

const route = useRoute();

const selectedRecorderId = computed(() => route.params.recorderId as string);

const { state, execute } = useAsyncState(async () => {
  return { sessions: await getOpenSessions(selectedRecorderId.value) };
}, { sessions: [] });

watch(selectedRecorderId, () => {
  execute();
});
</script>

<template>
  <div class="list">
    <template v-for="(session, index) in state.sessions" :key="session.id">
      <SessionCard :session="session" :recorder-id="selectedRecorderId" :index="state.sessions.length - index" />
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