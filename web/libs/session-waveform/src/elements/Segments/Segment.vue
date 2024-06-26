<script setup lang="ts">
import Button from '../../lib/controls/Button.vue';
import Marker from '../../lib/controls/Marker.vue';
import { usePeaksContext } from '../../context/usePeaksContext';
import TextInput from '../../lib/forms/TextInput.vue';
import { computed } from 'vue';
import { parseTimeFromSeconds } from '../../lib/utils/parseTimeFromSeconds';
import { parseSecondsFromTime } from '../../lib/utils/parseSecondsFromTime';
import type { Segment } from '../../context/models/state';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

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
      <div class="buttons">
        <template v-if="segment.renders.length">
          <template v-for="render in segment.renders" :key="render.src">
            <Button tag-name="a" size="xs" color="primary" :href="render.src">
              <font-awesome-icon
                icon="fa-solid fa-download"
              ></font-awesome-icon>
              {{ render.type.split('/').at(-1) }}
            </Button>
          </template>
        </template>

        <template v-else>
          <Button
            size="xs"
            variant="solid"
            @click="() => commandEmitter.emit('renderSegment', segment.id)"
          >
            <font-awesome-icon icon="fa-solid fa-music"></font-awesome-icon>
            Render
          </Button>
        </template>

        <Button
          v-if="canDelete"
          size="xs"
          variant="ghost"
          @click="() => commandEmitter.emit('removeSegment', segment.id)"
        >
          <font-awesome-icon icon="fa-solid fa-trash"></font-awesome-icon>
          Remove
        </Button>
      </div>
    </td>
  </tr>
</template>

<style scoped>
.row--deleted {
  opacity: 0.5;
}

.buttons {
  display: flex;
  flex-direction: row;
  gap: 0.25rem;
}
</style>

<i18n locale="en" lang="json"></i18n>
