<script setup lang="ts">
import { cutSession } from '../../../grpc/procedures/cutSession';
import { storeToRefs } from 'pinia';
import { useRecordersStore } from '../../../store/useRecordersStore';
import { computed } from 'vue';
import { SignalStatus } from '@session-recorder/protocols/ts/common';
import { Button } from '../../../../libs/session-waveform/src';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

const { recorders, selectedRecorderId } = storeToRefs(useRecordersStore());
const recorder = computed(() => {
  return recorders.value.get(selectedRecorderId.value);
});
</script>

<template>
  <div v-if="recorder" class="actions">
    <div
      v-if="recorder.status.signalStatus === SignalStatus.SIGNAL"
      class="banner"
    >
      <div>This recorder is currently recording</div>
      <Button
        @click="() => cutSession({ recorderID: selectedRecorderId })"
        color="primary"
      >
        <font-awesome-icon icon="fa-solid fa-scissors" />
        Cut Session
      </Button>
    </div>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  flex-direction: column;
  margin: var(--size-3) auto;
}

.banner {
  border-radius: var(--radius-md);
  border: 1px solid var(--color-grey-200);
  display: flex;
  gap: var(--size-2);
  padding: var(--size-4);
  font-size: var(--scale-2);
  align-items: center;
}

.banner > *:last-child {
  margin-left: auto;
}
</style>
