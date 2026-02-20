<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useRecordersStore } from '../../store/useRecordersStore';
import PageShell from '../../layout/PageShell.vue';
import DevicePicker from './elements/DevicePicker.vue';

const { recorders, selectedRecorderId } = storeToRefs(useRecordersStore());
</script>

<template>
  <PageShell>
    <template #sidebar>
      <DevicePicker
        :recorders="recorders"
        :selected-recorder-id="selectedRecorderId"
      />
    </template>
    <router-view />
  </PageShell>
</template>
