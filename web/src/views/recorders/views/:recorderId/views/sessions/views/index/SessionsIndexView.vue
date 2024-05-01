<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useRecordersStore } from '@/store/useRecordersStore';
import { useSessionsStore } from '@/store/useSessionsStore';
import { EmptyScreen } from '@session-recorder/session-waveform';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import SessionCard from '@/views/recorders/elements/SessionCard.vue';
import { cutSession } from "../../../../../../../../grpc/procedures/cutSession";

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { sessions } = storeToRefs(useSessionsStore());

const sortedSessions = computed(() => {
  return Array.from(sessions.value.values()).sort((a, b) => {
    return (
      Number(b.updated.timeCreated?.getTime()) -
      Number(a.updated.timeCreated?.getTime())
    );
  });
});
</script>

<template>
  <Button @click="() => cutSession({ recorderID: selectedRecorderId })">Cut Session</Button>

  <div v-if="sortedSessions.length" class="list">
    <template v-for="(session, index) in sortedSessions" :key="session.ID">
      <SessionCard
        :session="session"
        :recorder-id="selectedRecorderId"
        :index="sessions.size - index"
      />
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
  gap: var(--size-20);
}
</style>
