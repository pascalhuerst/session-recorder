<script setup lang="ts">
import { useRouter } from 'vue-router';
import { Recorder } from '@session-recorder/protocols/ts/sessionsource';
import { computed } from 'vue';
import DeviceCard from './DeviceCard.vue';

const router = useRouter();

const props = defineProps<{
  recorders: Map<string, Recorder>;
  selectedRecorderId?: string;
}>();

const recorders = computed(() =>
  Array.from(props.recorders.values()).sort((a, b) => {
    return a.recorderName.localeCompare(b.recorderName);
  })
);

const setSelected = (item: string) => {
  router.push(`/recorders/${item}/sessions`);
};
</script>

<template>
  <nav>
    <ul>
      <li v-for="recorder in recorders" :key="recorder.recorderID">
        <DeviceCard
          :recorder="recorder"
          :is-selected="selectedRecorderId === recorder.recorderID"
          @click="() => setSelected(recorder.recorderID)"
        />
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
</style>
