<script setup lang="ts">
import Button from '../../../lib/controls/Button.vue';
import { usePeaksContext } from '../../../context/usePeaksContext';

const {
  segments: { addSegment },
} = usePeaksContext();
</script>

<template>
  <Button size="sm" @click="addSegment" class="add-segment-btn">
    Add Segment
  </Button>
</template>

<style scoped></style>

<i18n locale="en" lang="json"></i18n>
