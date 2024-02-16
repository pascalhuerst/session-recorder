<script setup lang="ts">
import Button from '../../../lib/controls/Button.vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { usePeaksContext } from '../../../context/usePeaksContext';
import { onClickOutside } from '@vueuse/core';
import { watch } from 'vue';

const {
  commandEmitter,
  layout: { canvasElement },
  player: { duration, isPlaying },
} = usePeaksContext();

const handleClick = () => {
  isPlaying.value ? commandEmitter.emit('pause') : commandEmitter.emit('play');
};

onClickOutside(canvasElement.value, () => {
  if (isPlaying.value) {
    commandEmitter.emit('pause');
  }
});

watch(
  duration,
  () => {
    console.log({ duration: duration.value });
  },
  { immediate: true }
);
</script>

<template>
  <Button
    shape="square"
    size="lg"
    variant="ghost"
    color="primary"
    :disabled="!duration"
    @click="handleClick"
  >
    <font-awesome-icon v-if="isPlaying" icon="fa-solid fa-pause" />
    <font-awesome-icon v-else icon="fa-solid fa-play" />
  </Button>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
