<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';
import { useShowsStore } from '@/store/useShowsStore';
import { useRecordersStore } from '@/store/useRecordersStore';

const showsStore = useShowsStore();
const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { loading } = storeToRefs(showsStore);

showsStore.fetchShows();

const recorderShows = computed(() =>
  showsStore.shows.filter((s) => s.recorderId === selectedRecorderId.value),
);

const upcomingShows = computed(() =>
  recorderShows.value.filter((s) => s.state === 'draft'),
);

const liveShows = computed(() =>
  recorderShows.value.filter((s) => s.state === 'live'),
);

const pastShows = computed(() =>
  recorderShows.value.filter(
    (s) => s.state === 'ended' || s.state === 'archived',
  ),
);

const newShowPath = computed(
  () => `/recorders/${selectedRecorderId.value}/shows/new`,
);

function showPath(showId: string) {
  return `/recorders/${selectedRecorderId.value}/shows/${showId}`;
}
</script>

<template>
  <div class="shows-list">
    <div class="shows-header">
      <h1>Shows</h1>
      <router-link :to="newShowPath" class="new-show-btn">+ New Show</router-link>
    </div>

    <section v-if="liveShows.length > 0">
      <h2>Live</h2>
      <div class="show-cards">
        <router-link
          v-for="show in liveShows"
          :key="show.id"
          :to="showPath(show.id)"
          class="show-card live"
        >
          <div class="show-card-header">
            <span class="show-card-name">{{ show.name }}</span>
            <span class="show-card-badge live">live</span>
          </div>
          <span class="show-card-date">{{ show.date.toLocaleDateString() }}</span>
          <span class="show-card-acts">{{ show.acts.length }} {{ show.acts.length === 1 ? 'act' : 'acts' }}</span>
        </router-link>
      </div>
    </section>

    <section v-if="upcomingShows.length > 0">
      <h2>Upcoming</h2>
      <div class="show-cards">
        <router-link
          v-for="show in upcomingShows"
          :key="show.id"
          :to="showPath(show.id)"
          class="show-card"
        >
          <span class="show-card-name">{{ show.name }}</span>
          <span class="show-card-date">{{ show.date.toLocaleDateString() }}</span>
          <span class="show-card-acts">{{ show.acts.length }} {{ show.acts.length === 1 ? 'act' : 'acts' }}</span>
        </router-link>
      </div>
    </section>

    <section v-if="pastShows.length > 0">
      <h2>Past</h2>
      <div class="show-cards">
        <router-link
          v-for="show in pastShows"
          :key="show.id"
          :to="showPath(show.id)"
          class="show-card"
          :class="show.state"
        >
          <div class="show-card-header">
            <span class="show-card-name">{{ show.name }}</span>
            <span class="show-card-badge" :class="show.state">{{ show.state }}</span>
          </div>
          <span class="show-card-date">{{ show.date.toLocaleDateString() }}</span>
          <span class="show-card-acts">{{ show.acts.length }} {{ show.acts.length === 1 ? 'act' : 'acts' }}</span>
        </router-link>
      </div>
    </section>

    <p v-if="!loading && recorderShows.length === 0" class="empty">
      No shows for this recorder yet.
    </p>
  </div>
</template>

<style scoped>
.shows-list {
  max-width: 800px;
}

.shows-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--size-4);
}

h1 {
  font-size: var(--scale-3);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
}

h2 {
  font-size: var(--scale-1);
  font-weight: var(--weight-semibold);
  margin-bottom: var(--size-3);
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

section {
  margin-bottom: var(--size-6);
}

.new-show-btn {
  padding: var(--size-2) var(--size-3);
  border: 1px dashed var(--border-primary);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: var(--scale-0);
  text-decoration: none;
  transition: all 0.15s ease;
}

.new-show-btn:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
  background: var(--bg-hover);
}

.show-cards {
  display: flex;
  flex-direction: column;
  gap: var(--size-2);
}

.show-card {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
  padding: var(--size-3) var(--size-4);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--text-primary);
  transition: all 0.15s ease;
}

.show-card:hover {
  background: var(--bg-hover);
  border-color: var(--text-muted);
}

.show-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.show-card-name {
  font-weight: var(--weight-semibold);
  font-size: var(--scale-1);
}

.show-card-date,
.show-card-acts {
  font-size: var(--scale-0);
  color: var(--text-muted);
}

.show-card-badge {
  font-size: var(--scale-00);
  padding: 2px 6px;
  border-radius: var(--radius-xs);
  text-transform: uppercase;
  font-weight: var(--weight-semibold);
}

.show-card-badge.live {
  background: #dc2626;
  color: white;
}

.show-card-badge.ended {
  background: #2563eb;
  color: white;
}

.show-card-badge.archived {
  background: var(--color-grey-300);
  color: var(--color-grey-600);
}

.empty {
  color: var(--text-muted);
  font-size: var(--scale-1);
}
</style>
