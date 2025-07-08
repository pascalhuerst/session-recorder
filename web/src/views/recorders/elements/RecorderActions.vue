<script setup lang="ts">
import { cutSession } from '../../../grpc/procedures/cutSession';
import { storeToRefs } from 'pinia';
import { useRecordersStore } from '../../../store/useRecordersStore';
import { computed } from 'vue';
import { SignalStatus } from '@session-recorder/protocols/ts/common';
import { Button } from '@session-recorder/session-waveform';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { toastService } from '../../../services/Toaster';

const { recorders, selectedRecorderId } = storeToRefs(useRecordersStore());
const recorder = computed(() => {
  return recorders.value.get(selectedRecorderId.value);
});

const handleCutSession = async () => {
  try {
    await cutSession({ recorderID: selectedRecorderId.value });
    toastService.success(
      'Session has ended. It may take some seconds before the session files are prepared.'
    );
  } catch (error) {
    console.error('Failed to cut session:', error);
    toastService.error('Failed to cut session');
  }
};
</script>

<template>
  <div v-if="recorder" class="actions">
    <div
      v-if="
        recorder.info.oneofKind === 'status' &&
        recorder.info.status.signalStatus === SignalStatus.SIGNAL
      "
      class="banner"
    >
      <div>This recorder is currently recording</div>
      <Button @click="handleCutSession" color="primary">
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
