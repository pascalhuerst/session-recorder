<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useRecordersStore } from "@/store/useRecordersStore";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { watch } from "vue";
import { EmptyScreen } from "@session-recorder/session-waveform";

const { recorders, selectedRecorderId } = storeToRefs(useRecordersStore());

watch(
  [recorders, selectedRecorderId],
  () => {
    if (!selectedRecorderId.value && recorders.value.size) {
      const keys = Array.from(recorders.value.keys());
      selectedRecorderId.value = keys[0];
    }
  },
  { immediate: true, deep: true }
);
</script>

<template>
  <EmptyScreen>
    <template #illustration>
      <font-awesome-icon icon="fa-solid fa-microchip" />
    </template>
    <template #text>
      Please select a recorder to access the list of recorded sessions
    </template>
  </EmptyScreen>
</template>
