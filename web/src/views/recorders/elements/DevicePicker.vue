<script setup lang="ts">
import { useRouter } from 'vue-router';
import { type Recorder } from '@session-recorder/protocols/ts/sessionsource';
import { computed } from 'vue';

const router = useRouter();

const props = defineProps<{
  recorders: Map<string, Recorder>;
  selectedRecorderId?: string;
}>();

const recorders = computed(() => Array.from(props.recorders.values()));

const setSelected = (item: string) => {
  router.push(`/recorders/${item}/sessions`);
};
</script>

<template>
  <nav>
    <ul>
      <li v-for="recorder in recorders" :key="recorder.recorderID">
        <button
          :class="[
            'link',
            { 'is-active': selectedRecorderId === recorder.recorderID },
          ]"
          @click="setSelected(recorder.recorderID)"
        >
          {{ recorder.recorderName }}
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
