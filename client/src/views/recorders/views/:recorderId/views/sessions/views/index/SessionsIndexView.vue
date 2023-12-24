<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { useRecordersStore } from "../../../../../../../../store/useRecordersStore.ts";
import { useSessionsStore } from "../../../../../../../../store/useSessionsStore.ts";
import SessionCard from "../../../../../../elements/SessionCard.vue";
import EmptyScreen from "../../../../../../../../lib/display/EmptyScreen.vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { sessions } = storeToRefs(useSessionsStore());

const sortedSessions = computed(() => {
  return Array.from(sessions.value.values()).sort((a, b) => {
    return Number(b.timeCreated?.getTime()) - Number(a.timeCreated?.getTime());
  });
});
</script>

<template>
  <div v-if="sortedSessions.length" class="list">
    <template v-for="(session, index) in sortedSessions" :key="session.id">
      <SessionCard :session="session" :recorder-id="selectedRecorderId" :index="sessions.size - index" />
    </template>
    <router-view />
  </div>
  <div v-else>
    <EmptyScreen>
      <template #illustration>
        <font-awesome-icon icon="fa-solid fa-wave-square" />
      </template>
      <template #text>
        There are no open sessions that were recorded by this recording device
      </template>
    </EmptyScreen>
  </div>
</template>

<style scoped>
.list {
  display: flex;
  flex-direction: column;
  gap: var(--size-16);
}
</style>