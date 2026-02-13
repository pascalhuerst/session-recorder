<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { useRecordersStore } from '@/store/useRecordersStore';
import { useShowsStore } from '@/store/useShowsStore';
import { useSessionsStore } from '@/store/useSessionsStore';
import LiveShowTimeline from './elements/LiveShowTimeline.vue';
import LiveRecordingWaveform from './elements/LiveRecordingWaveform.vue';

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const showsStore = useShowsStore();
const { sessions } = storeToRefs(useSessionsStore());

showsStore.fetchShows();

const liveShow = computed(() =>
  showsStore.shows.find(
    (s) => s.recorderId === selectedRecorderId.value && s.state === 'live',
  ),
);

const recordingSession = computed(() =>
  sessions.value.find((s) => s.state === 'recording'),
);

const exitPath = computed(
  () => `/recorders/${selectedRecorderId.value}/sessions`,
);
</script>

<template>
  <div class="live-mode">
    <div class="live-mode-header">
      <span class="live-mode-title">Live Mode</span>
      <router-link :to="exitPath" class="exit-btn">
        <font-awesome-icon icon="fa-solid fa-xmark" />
        Exit
      </router-link>
    </div>
    <div class="live-mode-content">
      <LiveShowTimeline v-if="liveShow" :show="liveShow" />
      <LiveRecordingWaveform
        v-else-if="recordingSession"
        :session="recordingSession"
        :recorder-id="selectedRecorderId"
      />
      <div v-else class="empty-state">
        <font-awesome-icon icon="fa-solid fa-tower-broadcast" class="empty-icon" />
        <p>No active show or recording</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.live-mode {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: var(--bg-primary);
  display: flex;
  flex-direction: column;
}

.live-mode-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--size-3) var(--size-4);
  border-bottom: 1px solid var(--border-primary);
  flex-shrink: 0;
}

.live-mode-title {
  font-size: var(--scale-1);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}

.exit-btn {
  display: flex;
  align-items: center;
  gap: var(--size-2);
  padding: var(--size-2) var(--size-3);
  border-radius: var(--radius-sm);
  font-size: var(--scale-0);
  text-decoration: none;
  color: var(--text-muted);
  transition: all 0.15s ease;
}

.exit-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.live-mode-content {
  flex: 1;
  display: flex;
  min-height: 0;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--size-4);
  color: var(--text-muted);
}

.empty-icon {
  font-size: 3rem;
  opacity: 0.3;
}

.empty-state p {
  font-size: var(--scale-2);
}
</style>
