<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { EmptyScreen } from '@session-recorder/session-waveform';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import SessionCard from '@/views/recorders/elements/SessionCard.vue';
import { useRecordersStore } from '../../../../../../../../store/useRecordersStore';
import { useSessionsStore } from '../../../../../../../../store/useSessionsStore';

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { sessions } = storeToRefs(useSessionsStore());
</script>

<template>
  <div v-if="sessions.length" class="list">
    <template v-for="(session, index) in sessions" :key="session.id">
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
