<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { EmptyScreen } from '@session-recorder/session-waveform';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import SessionCard from '@/views/recorders/elements/SessionCard.vue';
import { useSessionsStore } from '../../../../../../../../store/useSessionsStore';
import { useRecordersStore } from '../../../../../../../../store/useRecordersStore';
import { type Session } from '@session-recorder/protocols/ts/sessionsource';

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { sessions } = storeToRefs(useSessionsStore());

const sortedSessions = computed(() => {
  const sorted = sessions.value.slice().sort((a, b) => {
    const aTime =
      a.info.oneofKind === 'updated' && a.info.updated.timeCreated
        ? a.info.updated.timeCreated.seconds * 1000
        : 0;
    const bTime =
      b.info.oneofKind === 'updated' && b.info.updated.timeCreated
        ? b.info.updated.timeCreated.seconds * 1000
        : 0;
    return bTime - aTime;
  });

  return sorted satisfies Session[];
});
</script>

<template>
  <div v-if="sortedSessions.length" class="list">
    <template v-for="(session, index) in sortedSessions" :key="session.iD">
      <SessionCard
        :session="session"
        :recorder-id="selectedRecorderId"
        :index="sessions.length - index"
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
