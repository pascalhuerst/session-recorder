<script setup lang="ts">
import { usePeaksContext } from '../../context/usePeaksContext';
import { useWaverformLayoutProvider } from '../../waveform/useWaverformLayoutProvider';
import { computed } from 'vue';

const { inject } = useWaverformLayoutProvider();
const { audioRef } = inject();

const { state } = usePeaksContext();
const audioUrls = computed(() => state.toRef().value.audioUrls);
</script>

<template>
  <audio ref="audioRef">
    <source v-for="url in audioUrls" :key="url.type" v-bind="url" />
  </audio>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
