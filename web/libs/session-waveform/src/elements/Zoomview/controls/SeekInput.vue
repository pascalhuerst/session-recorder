<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { computed } from 'vue';
import { parseTimeFromSeconds } from '../../../lib/utils/parseTimeFromSeconds';
import { usePeaksContext } from '../../../context/usePeaksContext';
import TextInput from '../../../lib/forms/TextInput.vue';
import { parseSecondsFromTime } from '../../../lib/utils/parseSecondsFromTime';

const { commandEmitter, state } = usePeaksContext();

const duration = computed(() => state.toRef().value.player.duration);
const currentTime = computed(() => state.toRef().value.player.currentTime);

const playTime = computed({
  get: () => {
    if (!currentTime.value) {
      return '00:00:00';
    }
    const parsed = parseTimeFromSeconds(currentTime.value);
    return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
  },
  set: (value: string) => {
    const seconds = parseSecondsFromTime(value);
    commandEmitter.emit('seek', seconds);
  },
});

const maxPlayTime = computed(() => {
  if (!duration.value) {
    return '00:00:00';
  }

  const parsed = parseTimeFromSeconds(duration.value);
  return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
});

const showPicker = (ref?: HTMLInputElement) => {
  return ref?.showPicker();
};
</script>

<template>
  <TextInput
    type="time"
    v-model="playTime"
    :step="0.01"
    min="00:00:00"
    :max="maxPlayTime"
    size="md"
    variant="outlined"
    color="neutral"
  >
    <template #prepend="{ inputRef }">
      <font-awesome-icon
        icon="fa-regular fa-clock"
        @click="showPicker(inputRef)"
      />
    </template>
  </TextInput>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
