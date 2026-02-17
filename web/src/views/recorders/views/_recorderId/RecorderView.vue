<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useRecordersStore } from '@/store/useRecordersStore';
import { useShowsStore } from '@/store/useShowsStore';

const route = useRoute();
const { selectedRecorderId } = storeToRefs(useRecordersStore());
const showsStore = useShowsStore();

showsStore.fetchShows();

const activeTab = computed(() => {
  if (route.path.includes('/shows')) return 'shows';
  return 'sessions';
});

const hasLiveShow = computed(() =>
  showsStore.shows.some(
    (s) => s.recorderId === selectedRecorderId.value && s.state === 'live',
  ),
);

const sessionsPath = computed(
  () => `/recorders/${selectedRecorderId.value}/sessions`,
);
const showsPath = computed(
  () => `/recorders/${selectedRecorderId.value}/shows`,
);
const livePath = computed(
  () => `/recorders/${selectedRecorderId.value}/live`,
);
</script>

<template>
  <div class="recorder-view">
    <div class="recorder-tabs">
      <nav class="tab-nav">
        <router-link
          :to="sessionsPath"
          class="tab"
          :class="{ active: activeTab === 'sessions' }"
        >
          Sessions
        </router-link>
        <router-link
          :to="showsPath"
          class="tab"
          :class="{ active: activeTab === 'shows' }"
        >
          Shows
        </router-link>
      </nav>
      <router-link :to="livePath" class="live-mode-btn">
        <span v-if="hasLiveShow" class="live-dot" />
        Live Mode
      </router-link>
    </div>
    <div class="tab-content">
      <router-view />
    </div>
  </div>
</template>

<style scoped>
.recorder-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.recorder-tabs {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-primary);
  padding: 0 var(--size-4);
  flex-shrink: 0;
}

.tab-nav {
  display: flex;
  gap: var(--size-4);
}

.tab {
  padding: var(--size-3) 0;
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  color: var(--text-muted);
  text-decoration: none;
  border-bottom: 2px solid transparent;
  transition: all 0.15s ease;
}

.tab:hover {
  color: var(--text-primary);
}

.tab.active {
  color: var(--text-primary);
  font-weight: var(--weight-semibold);
  border-bottom-color: currentColor;
}

.live-mode-btn {
  display: flex;
  align-items: center;
  gap: var(--size-1);
  padding: var(--size-1) var(--size-3);
  border-radius: var(--radius-sm);
  font-size: var(--scale-0);
  text-decoration: none;
  color: var(--text-muted);
  transition: all 0.15s ease;
}

.live-mode-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #dc2626;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

.tab-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--size-4);
}
</style>
