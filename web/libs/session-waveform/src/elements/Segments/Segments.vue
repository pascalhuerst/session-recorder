<script setup lang="ts">
import { usePeaksContext } from '../../context/usePeaksContext';
import Segment from './Segment.vue';

const {
  segments: { segments },
} = usePeaksContext();
</script>

<template>
  <div class="segments">
    <table>
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
}
</style>

<i18n locale="en" lang="json"></i18n>
