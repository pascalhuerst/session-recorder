<script setup lang="ts">
import Marker from '../../lib/controls/Marker.vue';
import { usePeaksContext } from '../../context/usePeaksContext';

const {
  segments: { segments, selectSegment },
} = usePeaksContext();
</script>

<template>
  <div class="segments">
    <table>
      <tr
        v-for="segment in segments"
        :key="segment.id"
        @click="() => selectSegment(segment.id)"
      >
        <td>
          <Marker :index="segment.startIndex" :time="segment.startTime" />
        </td>
        <td>
          <Marker :index="segment.endIndex" :time="segment.endTime" />
        </td>
        <td>{{ segment.labelText }}</td>
        <td></td>
      </tr>
      <tr>
        <td></td>
        <td></td>
        <td></td>
        <td></td>
      </tr>
    </table>
  </div>
</template>

<style scoped>
.segments {
  margin-top: var(--size-10);
  padding: var(--size-6);
}

.segments table {
  width: 100%;

  td,
  th {
    padding: var(--size-2);
    border-bottom: 1px solid var(--color-grey-300);
  }
}
</style>

<i18n locale="en" lang="json"></i18n>
