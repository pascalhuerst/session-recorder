<script setup lang="ts">
import { useRoute, useRouter } from "vue-router";
import { computed, ref } from "vue";
import { streamRecorders } from "../../../grpc/procedures/streamRecorders.ts";
import { RecorderInfo } from "@session-recorder/protocols/ts/sessionsource.ts";

const router = useRouter();
const route = useRoute();

const recorders = ref<Set<RecorderInfo>>(new Set);

streamRecorders({
  onMessage: (recorderInfo) => {
    recorders.value.add(recorderInfo);
  }
});

const selectedRecorderId = computed(() => route.params.recorderId);

const setSelected = (item: string) => {
  router.push(`/recorders/${item}/sessions`);
};
</script>

<template>
  <nav>
    <ul>
      <li v-for="recorder in recorders" :key="recorder.ID">
        <button :class="['link', { 'is-active': selectedRecorderId === recorder.ID }]" @click="setSelected(recorder.ID)">
          {{ recorder.name }}
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