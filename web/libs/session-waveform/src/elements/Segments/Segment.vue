<script setup lang="ts">
import Button from '../../lib/controls/Button.vue';
import Marker from '../../lib/controls/Marker.vue';
import { usePeaksContext } from '../../context/usePeaksContext';
import TextInput from '../../lib/forms/TextInput.vue';
import { computed } from 'vue';
import { parseTimeFromSeconds } from '../../lib/utils/parseTimeFromSeconds';
import { parseSecondsFromTime } from '../../lib/utils/parseSecondsFromTime';
import type { Segment } from '../../context/models/state';

const props = defineProps<{
  segment: Segment;
}>();

const { commandEmitter, state } = usePeaksContext();

const permissions = computed(() => state.toRef().value.permissions);
const duration = computed(() => state.toRef().value.player.duration);

const segmentLabel = computed({
  get: () => props.segment.labelText,
  set: (labelText) => {
    commandEmitter.emit('updateSegment', props.segment.id, { labelText });
  },
});
const startTime = computed({
  get: () => {
    const parsed = parseTimeFromSeconds(props.segment.startTime);
    return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
  },
  set: (value: string) => {
    const seconds = parseSecondsFromTime(value);
    commandEmitter.emit('updateSegment', props.segment.id, {
      startTime: seconds,
    });
  },
});
const endTime = computed({
  get: () => {
    const parsed = parseTimeFromSeconds(props.segment.endTime);
    return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
  },
  set: (value: string) => {
    const seconds = parseSecondsFromTime(value);
    commandEmitter.emit('updateSegment', props.segment.id, {
      endTime: seconds,
    });
  },
});
const maxPlayTime = computed(() => {
  if (!duration.value) {
    return '00:00:00';
  }

  const parsed = parseTimeFromSeconds(duration.value);
  return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
});
const canUpdate = computed(() => {
  return permissions.value.update && !props.segment.deleted;
});
const canDelete = computed(() => {
  return permissions.value.delete && !props.segment.deleted;
});
</script>

<template>
  <tr
    @click="() => commandEmitter.emit('seek', segment.startTime)"
    :class="['row', { 'row--deleted': segment.deleted }]"
  >
    <td>
      <Marker :index="segment.startIndex">
        <template v-if="canUpdate">
          <TextInput
            type="time"
            v-model="startTime"
            :step="0.01"
            min="00:00:00"
            :max="maxPlayTime"
            size="sm"
            variant="ghost"
          />
        </template>
        <template v-else>
          {{ startTime }}
        </template>
      </Marker>
    </td>
    <td>
      <Marker :index="segment.endIndex">
        <template v-if="canUpdate">
          <TextInput
            type="time"
            v-model="endTime"
            :step="0.01"
            min="00:00:00"
            :max="maxPlayTime"
            size="sm"
            variant="ghost"
          />
        </template>
        <template v-else>
          {{ endTime }}
        </template>
      </Marker>
    </td>
    <td>
      <template v-if="canUpdate">
        <TextInput v-model="segmentLabel" size="sm" variant="ghost" />
      </template>
      <template v-else>
        {{ segment.labelText }}
      </template>
    </td>
    <td>
      <Button
        v-if="canDelete"
        size="xs"
        variant="ghost"
        @click="() => commandEmitter.emit('removeSegment', segment.id)"
        >Remove
      </Button>
    </td>
  </tr>
</template>

<style scoped>
.row--deleted {
  opacity: 0.5;
}
</style>

<i18n locale="en" lang="json"></i18n>
