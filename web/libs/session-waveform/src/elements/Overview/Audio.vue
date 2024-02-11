<script setup lang="ts">
import { usePeaksContext } from '../../context/usePeaksContext';

const {
  audioUrls,
  layout: { audioElement },
} = usePeaksContext();
</script>

<template>
  <audio ref="audioElement">
    <source v-for="url in audioUrls" :key="url.type" v-bind="url" />
  </audio>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
