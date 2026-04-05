<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';
import { EmptyScreen } from '@session-recorder/session-waveform';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import SessionCard from '@/views/recorders/elements/SessionCard.vue';
import RecordingSessionCard from '@/views/recorders/elements/RecordingSessionCard.vue';
import { useRecordersStore } from '../../../../../../../../store/useRecordersStore';
import { useSessionsStore } from '../../../../../../../../store/useSessionsStore';
import type { Session } from '@/types';

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { sessions } = storeToRefs(useSessionsStore());

// Separate recording sessions from finished/processing sessions
const recordingSession = computed(() =>
  sessions.value.find(s => s.state === 'recording')
);

const nonRecordingSessions = computed(() =>
  sessions.value.filter(s => s.state !== 'recording')
);

// Group non-recording sessions by date
const sessionsByDate = computed(() => {
  const groups: { date: string; dateLabel: string; sessions: { session: Session; index: number }[] }[] = [];
  const dateMap = new Map<string, { session: Session; index: number }[]>();

  nonRecordingSessions.value.forEach((session, idx) => {
    const dateKey = session.startedAt.toDateString();
    if (!dateMap.has(dateKey)) {
      dateMap.set(dateKey, []);
    }
    dateMap.get(dateKey)!.push({
      session,
      index: nonRecordingSessions.value.length - idx,
    });
  });

  // Convert to array and format date labels
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  for (const [dateKey, items] of dateMap) {
    const date = new Date(dateKey);
    let dateLabel: string;

    if (date.toDateString() === today.toDateString()) {
      dateLabel = 'Today';
    } else if (date.toDateString() === yesterday.toDateString()) {
      dateLabel = 'Yesterday';
    } else if (date.getFullYear() === today.getFullYear()) {
      dateLabel = date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' });
    } else {
      dateLabel = date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' });
    }

    groups.push({ date: dateKey, dateLabel, sessions: items });
  }

  // Sort groups by date descending (most recent first)
  groups.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());

  return groups;
});
</script>

<template>
  <div v-if="sessions.length" class="list">
    <!-- Recording session at top, outside date grouping -->
    <RecordingSessionCard
      v-if="recordingSession"
      :session="recordingSession"
      :recorder-id="selectedRecorderId"
    />

    <!-- Date-grouped sessions -->
    <div v-for="group in sessionsByDate" :key="group.date" class="date-group">
      <h2 class="date-header">{{ group.dateLabel }}</h2>
      <div class="sessions">
        <SessionCard
          v-for="item in group.sessions"
          :key="item.session.id"
          :session="item.session"
          :recorder-id="selectedRecorderId"
          :index="item.index"
        />
      </div>
    </div>
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
  gap: var(--size-8);
}

.date-group {
  display: flex;
  flex-direction: column;
  gap: var(--size-4);
}

.date-header {
  font-size: var(--scale-2);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  margin: 0;
  padding: var(--size-2) 0;
  position: sticky;
  top: 0;
  background: var(--bg-primary);
  z-index: 10;
}

.sessions {
  display: flex;
  flex-direction: column;
  gap: var(--size-6);
  padding-left: var(--size-4);
}

@media (max-width: 768px) {
  .list {
    gap: var(--size-4);
  }

  .sessions {
    padding-left: 0;
    gap: var(--size-4);
  }
}
</style>
