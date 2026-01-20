<script setup lang="ts">
import Button from '../../lib/controls/Button.vue';
import Checkbox from '../../lib/controls/Checkbox.vue';
import Marker from '../../lib/controls/Marker.vue';
import { usePeaksContext } from '../../context/usePeaksContext';
import TextInput from '../../lib/forms/TextInput.vue';
import { computed, ref } from 'vue';
import { parseTimeFromSeconds } from '../../lib/utils/parseTimeFromSeconds';
import { parseSecondsFromTime } from '../../lib/utils/parseSecondsFromTime';
import type { Segment } from '../../context/models/state';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { toSolidColor } from '../../lib/utils/segmentColors';

const props = defineProps<{
  segment: Segment;
  selected: boolean;
}>();

const emit = defineEmits<{
  'toggle-selection': [];
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

// Segment duration formatted
const segmentDuration = computed(() => {
  const durationSeconds = props.segment.endTime - props.segment.startTime;
  const parsed = parseTimeFromSeconds(durationSeconds);
  return `${parsed.minutes}:${parsed.seconds}`;
});

// Segment color (solid version for markers)
const segmentColor = computed(() =>
  props.segment.color ? toSolidColor(props.segment.color) : undefined
);

// FLAC render URL
const flacRender = computed(() =>
  props.segment.renders.find((r) => r.type === 'audio/flac')
);
const hasRenderedAudio = computed(
  () => props.segment.state === 'finished' && flacRender.value
);

// Audio playback for rendered segments
const audioRef = ref<HTMLAudioElement | null>(null);
const isPlaying = ref(false);

const togglePlayback = () => {
  if (!audioRef.value) return;

  if (isPlaying.value) {
    audioRef.value.pause();
  } else {
    audioRef.value.play();
  }
};

const onAudioPlay = () => {
  isPlaying.value = true;
};

const onAudioPause = () => {
  isPlaying.value = false;
};

const onAudioEnded = () => {
  isPlaying.value = false;
};
</script>

<template>
  <tr
    @click="() => commandEmitter.emit('seek', segment.startTime)"
    :class="['row', { 'row--deleted': segment.deleted, 'row--selected': selected }]"
  >
    <!-- Checkbox -->
    <td class="cell-checkbox" @click.stop>
      <Checkbox
        :modelValue="selected"
        :disabled="segment.deleted"
        size="sm"
        @update:modelValue="emit('toggle-selection')"
      />
    </td>

    <!-- Start Time -->
    <td class="cell-time">
      <Marker :index="segment.startIndex" :color="segmentColor">
        <template v-if="canUpdate">
          <TextInput
            type="time"
            v-model="startTime"
            :step="0.01"
            min="00:00:00"
            :max="maxPlayTime"
            size="sm"
            variant="ghost"
            @focus="() => commandEmitter.emit('seek', segment.startTime)"
            @click.stop
          />
        </template>
        <template v-else>
          {{ startTime }}
        </template>
      </Marker>
    </td>

    <!-- End Time -->
    <td class="cell-time">
      <Marker :index="segment.endIndex" :color="segmentColor">
        <template v-if="canUpdate">
          <TextInput
            type="time"
            v-model="endTime"
            :step="0.01"
            min="00:00:00"
            :max="maxPlayTime"
            size="sm"
            variant="ghost"
            @focus="() => commandEmitter.emit('seek', segment.endTime)"
            @click.stop
          />
        </template>
        <template v-else>
          {{ endTime }}
        </template>
      </Marker>
    </td>

    <!-- Label -->
    <td class="cell-label">
      <template v-if="canUpdate">
        <TextInput v-model="segmentLabel" size="sm" variant="ghost" @click.stop />
      </template>
      <template v-else>
        {{ segment.labelText }}
      </template>
    </td>

    <!-- Actions -->
    <td class="cell-actions">
      <div class="actions">
        <!-- Queued state: show queued indicator -->
        <template v-if="segment.state === 'queued'">
          <div class="status-indicator status-indicator--queued">
            <span class="status-dot"></span>
            <span class="status-text">queued</span>
          </div>
        </template>

        <!-- Rendering state: show processing indicator -->
        <template v-else-if="segment.state === 'rendering'">
          <div class="status-indicator status-indicator--processing">
            <span class="status-dot"></span>
            <span class="status-text">processing</span>
          </div>
        </template>

        <!-- Error state: show error and retry button -->
        <template v-else-if="segment.state === 'error'">
          <span class="error-indicator" :title="segment.errorMessage">
            <font-awesome-icon
              icon="fa-solid fa-exclamation-triangle"
            ></font-awesome-icon>
            Error
          </span>
          <Button
            size="xs"
            variant="solid"
            @click.stop="() => commandEmitter.emit('renderSegment', segment.id)"
          >
            <font-awesome-icon icon="fa-solid fa-redo"></font-awesome-icon>
            Retry
          </Button>
        </template>

        <!-- Ready/Finished state: show play button and duration -->
        <template v-else>
          <template v-if="hasRenderedAudio">
            <audio
              ref="audioRef"
              :src="flacRender!.src"
              @play="onAudioPlay"
              @pause="onAudioPause"
              @ended="onAudioEnded"
            />
            <span class="duration">{{ segmentDuration }}</span>
            <Button
              size="xs"
              shape="square"
              variant="ghost"
              color="primary"
              @click.stop="togglePlayback"
              :disabled="segment.deleted"
            >
              <font-awesome-icon v-if="isPlaying" icon="fa-solid fa-pause" />
              <font-awesome-icon v-else icon="fa-solid fa-play" />
            </Button>
            <Button
              size="xs"
              shape="square"
              variant="ghost"
              color="primary"
              @click.stop="() => commandEmitter.emit('shareSegment', segment.id)"
              :disabled="segment.deleted"
              title="Share via email"
            >
              <font-awesome-icon icon="fa-solid fa-share"></font-awesome-icon>
            </Button>
            <Button
              tag-name="a"
              size="xs"
              shape="square"
              variant="ghost"
              color="primary"
              :href="flacRender!.src"
              download
              @click.stop
            >
              <font-awesome-icon icon="fa-solid fa-download"></font-awesome-icon>
            </Button>
          </template>
        </template>
      </div>
    </td>
  </tr>
</template>

<style scoped>
.row--deleted {
  opacity: 0.5;
}

.row--selected {
  background-color: var(--bg-selected, var(--accent-subtle));
}

.cell-checkbox {
  width: 40px;
}

.cell-checkbox :deep(label) {
  display: flex;
  align-items: center;
  justify-content: center;
}

.cell-time {
  width: 140px;
}

.cell-label {
  /* flexible width */
}

.cell-actions {
  width: 160px;
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--size-2);
}

.duration {
  font-size: var(--scale-0);
  color: var(--color-grey-600);
  font-variant-numeric: tabular-nums;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: var(--size-1);
}

.status-dot {
  width: var(--size-2);
  height: var(--size-2);
  border-radius: 50%;
  background: var(--color-grey-400);
}

.status-indicator--queued .status-dot {
  background: var(--color-purple-400, #a78bfa);
  animation: pulse-purple 2s infinite;
}

.status-indicator--processing .status-dot {
  background: var(--color-grey-500);
  animation: pulse-grey 2s infinite;
}

.status-text {
  font-size: var(--scale-00);
  font-weight: bold;
  text-transform: uppercase;
  color: var(--color-grey-500);
}

@keyframes pulse-grey {
  0%, 100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(107, 114, 128, 0.4);
  }
  50% {
    transform: scale(1);
    box-shadow: 0 0 0 4px rgba(107, 114, 128, 0.2);
  }
}

@keyframes pulse-purple {
  0%, 100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(167, 139, 250, 0.4);
  }
  50% {
    transform: scale(1);
    box-shadow: 0 0 0 4px rgba(167, 139, 250, 0.2);
  }
}

.error-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--color-error, #ef4444);
  font-size: 0.875rem;
}
</style>

<i18n locale="en" lang="json"></i18n>
