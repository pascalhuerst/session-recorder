<script setup lang="ts">
import Button from '../../lib/controls/Button.vue';
import Marker from '../../lib/controls/Marker.vue';
import { usePeaksContext } from '../../context/usePeaksContext';
import type { Segment } from '../../types';

defineProps<{
  segment: Segment;
}>();

const { commandEmitter, permissions } = usePeaksContext();
</script>

<template>
  <tr @click="() => commandEmitter.emit('jumpToSegment', segment.id)">
    <td>
      <Marker :index="segment.startIndex" :time="segment.startTime" />
    </td>
    <td>
      <Marker :index="segment.endIndex" :time="segment.endTime" />
    </td>
    <td>
      <template v-if="permissions.update">
        {{ segment.labelText }}
      </template>
    </td>
    <td>
      <Button
        v-if="permissions.delete"
        size="xs"
        variant="ghost"
        @click="() => commandEmitter.emit('removeSegment', segment.id)"
        >Remove
      </Button>
    </td>
  </tr>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
