<script setup lang="ts">
import { usePeaksContext } from '../../context/usePeaksContext';
import Segment from './Segment.vue';
import { computed } from 'vue';

const { state } = usePeaksContext();
const segments = computed(() => state.toRef().value.segments);
</script>

<template>
  <div class="segments">
    <table>
      <thead>
        <tr>
          <th :style="{ width: '10%' }"></th>
          <th :style="{ width: '10%' }"></th>
          <th></th>
          <th :style="{ width: '20%' }"></th>
        </tr>
      </thead>
      <Segment
        v-for="segment in segments"
        :key="segment.id"
        :segment="segment"
      />
    </table>
  </div>
</template>

<style scoped>
.segments {
  margin-top: var(--size-10);
  padding: var(--size-6);
}

table {
  width: 100%;
}

:deep(td),
:deep(th) {
  padding: var(--size-2);
  border-bottom: 1px solid var(--color-grey-300);
  vertical-align: center;
}
</style>

<i18n locale="en" lang="json"></i18n>
