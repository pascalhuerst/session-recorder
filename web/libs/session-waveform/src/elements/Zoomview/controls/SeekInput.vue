<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { computed } from 'vue';
import { parsePlayTime } from '../../../lib/utils/parsePlayTime';
import { usePeaksContext } from '../../../context/usePeaksContext';
import TextInput from '../../../lib/forms/TextInput.vue';

const {
  player: { duration, currentTime, seek },
} = usePeaksContext();

const playTime = computed({
  get: () => {
    if (!currentTime.value) {
      return '00:00:00';
    }
    const parsed = parsePlayTime(currentTime.value);
    return `${parsed.hours}:${parsed.minutes}:${parsed.seconds}.${parsed.milliseconds}`;
  },
  set: (value: string) => {
    const [h, m, s, ms] =
      value
        .match(/(\d{2}):(\d{2}):(\d{2})\.(\d{3})/)
        ?.slice(1, 5)
        .map((val) => parseInt(val)) || [];
    const seconds = h * 3600 + m * 60 + s + ms / 1000;
    seek(seconds);
  },
});

const maxPlayTime = computed(() => {
  if (!duration.value) {
    return '00:00:00';
  }

  const parsed = parsePlayTime(duration.value);
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
