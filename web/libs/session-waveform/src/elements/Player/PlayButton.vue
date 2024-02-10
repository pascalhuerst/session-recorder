<script setup lang="ts">
import Button from '../../controls/Button.vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { useAudioControls } from '../../context/useAudioControls';

const { player, onToggle, isPlaying } = useAudioControls();
</script>

<template>
  <Button
    shape="square"
    size="lg"
    variant="ghost"
    color="primary"
    :disabled="!player"
    @click="onToggle"
  >
    <font-awesome-icon v-if="isPlaying" icon="fa-solid fa-pause" />
    <font-awesome-icon v-else icon="fa-solid fa-play" />
  </Button>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
