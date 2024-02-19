<script setup lang="ts">
import Button from '../../../lib/controls/Button.vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { usePeaksContext } from '../../../context/usePeaksContext';
import { computed } from 'vue';

const { commandEmitter, state } = usePeaksContext();

const player = computed(() => state.toRef().value.player);

const handleClick = () => {
  player.value.isPlaying
    ? commandEmitter.emit('pause')
    : commandEmitter.emit('play');
};
</script>

<template>
  <Button
    shape="square"
    size="lg"
    variant="ghost"
    color="primary"
    :disabled="!player.duration"
    @click="handleClick"
  >
    <font-awesome-icon v-if="player.isPlaying" icon="fa-solid fa-pause" />
    <font-awesome-icon v-else icon="fa-solid fa-play" />
  </Button>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
