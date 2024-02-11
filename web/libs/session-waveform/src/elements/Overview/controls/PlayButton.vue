<script setup lang="ts">
import Button from '../../../lib/controls/Button.vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { usePeaksContext } from '../../../context/usePeaksContext';

const {
  player: { duration, play, pause, isPlaying },
} = usePeaksContext();

const handleClick = () => {
  isPlaying.value ? pause() : play();
};
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
