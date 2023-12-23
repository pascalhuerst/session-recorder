<script setup lang="ts">
import { useAsyncState } from "@vueuse/core";
import { getRecordingDevices } from "../../../client/getRecordingDevices.ts";
import { useRoute, useRouter } from "vue-router";
import { computed } from "vue";

const router = useRouter();
const route = useRoute();

const { state } = useAsyncState(async () => {
  return { recorderIds: await getRecordingDevices() };
}, { recorderIds: [] });

const selectedRecorderId = computed(() => route.params.recorderId);

const setSelected = (item: string) => {
  router.push(`/recorders/${item}/sessions`);
};
</script>

<template>
  <nav>
    <ul>
      <li v-for="recorderId in state.recorderIds" :key="recorderId">
        <button :class="['link', { 'is-active': selectedRecorderId === recorderId }]" @click="setSelected(recorderId)">
          {{ recorderId }}
        </button>
      </li>
    </ul>
  </nav>
</template>

<style scoped>
ul {
  display: flex;
  flex-wrap: nowrap;
  gap: var(--size-2);
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-x: auto;
}

.link {
  display: inline-block;
  border: none;
  color: var(--color-black);
  background: transparent;
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  padding: var(--size-1) var(--size-2);
  white-space: nowrap;
  border-radius: var(--radius-sm);
  text-decoration: none;
  cursor: pointer;
}

.link:hover,
.link.is-active {
  color: #fff;
  background: var(--color-black);
}
</style>