import { computed, ref } from 'vue';
import { defineStore } from 'pinia';
import type { Show } from '../types';
import * as showsApi from '../grpc/procedures/shows';

export const useShowsStore = defineStore('shows', () => {
  const shows = ref<Show[]>([]);
  const loading = ref(false);

  const upcomingShows = computed(() =>
    shows.value.filter((s) => s.state === 'draft'),
  );

  const pastShows = computed(() =>
    shows.value.filter((s) => s.state === 'ended' || s.state === 'archived'),
  );

  async function fetchShows() {
    loading.value = true;
    try {
      shows.value = await showsApi.listShows();
    } finally {
      loading.value = false;
    }
  }

  async function createShow(
    show: Parameters<typeof showsApi.createShow>[0],
  ) {
    await showsApi.createShow(show);
    await fetchShows();
  }

  async function updateShow(
    show: Parameters<typeof showsApi.updateShow>[0],
  ) {
    await showsApi.updateShow(show);
    await fetchShows();
  }

  async function deleteShow(showId: string) {
    await showsApi.deleteShow(showId);
    await fetchShows();
  }

  async function startShow(showId: string) {
    await showsApi.startShow(showId);
    await fetchShows();
  }

  async function renderAllActs(showId: string) {
    await showsApi.renderAll(showId);
    await fetchShows();
  }

  async function distributeAllActs(showId: string) {
    await showsApi.distributeAll(showId);
    await fetchShows();
  }

  return {
    shows,
    loading,
    upcomingShows,
    pastShows,
    fetchShows,
    createShow,
    updateShow,
    deleteShow,
    startShow,
    renderAllActs,
    distributeAllActs,
  };
});
