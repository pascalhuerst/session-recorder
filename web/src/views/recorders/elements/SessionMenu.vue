<script setup lang="ts">
import { computed, ref } from 'vue';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import {
  Button,
  Modal,
  useConfirmation,
} from '@session-recorder/session-waveform';
import { setKeepSession } from '../../../grpc/procedures/setKeepSession';
import { deleteSession } from '../../../grpc/procedures/deleteSession';
import { shareSession } from '../../../grpc/procedures/shareSession';
import type { Session } from '../../../types';
import { useDateFormat } from '@vueuse/core';
import { toastService } from '../../../services/Toaster/ToastService';
import ShareModal from '../../../components/ShareModal.vue';

// @todo: break this down and make composable

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const { awaitConfirmation, modalProps } = useConfirmation();

const showShareModal = ref(false);
const isSharing = ref(false);

const sessionName = computed(() => {
  return props.session.name || 'Untitled Recording';
});

const displayExpiryDate = computed(() => {
  const { expiresAt } = props.session;
  if (!expiresAt) {
    return null;
  }
  const format =
    expiresAt.getFullYear() === new Date().getFullYear()
      ? 'ddd, MMM D, HH:mm'
      : 'MMM D, YYYY HH:mm';
  return {
    iso: expiresAt.toISOString(),
    formatted: useDateFormat(expiresAt, format).value,
  };
});

const onKeep = () => {
  const keepAction = !props.session.keep;
  return setKeepSession({
    recorderId: props.recorderId,
    sessionId: props.session.id,
    keep: keepAction,
  })
    .then(() => {
      toastService.success(
        keepAction ? 'Session kept successfully' : 'Session unkeep successfully'
      );
    })
    .catch((error) => {
      console.error('Failed to update session keep status:', error);
      toastService.error('Failed to update session keep status');
    });
};

const onDelete = () => {
  awaitConfirmation().then(({ isConfirmed }) => {
    if (isConfirmed) {
      deleteSession({
        recorderId: props.recorderId,
        sessionId: props.session.id,
      })
        .then(() => {
          toastService.success('Session deleted successfully');
        })
        .catch((error) => {
          console.error('Failed to delete session:', error);
          toastService.error('Failed to delete session');
        });
    }
  });
};

const onShare = () => {
  showShareModal.value = true;
};

const onShareConfirm = async (email: string) => {
  isSharing.value = true;
  try {
    await shareSession({
      recorderId: props.recorderId,
      sessionId: props.session.id,
      recipientEmail: email,
    });
    toastService.success(`Download link sent to ${email}`);
    showShareModal.value = false;
  } catch (error) {
    console.error('Failed to share session:', error);
    toastService.error('Failed to send email. Please try again.');
  } finally {
    isSharing.value = false;
  }
};

const onShareClose = () => {
  showShareModal.value = false;
};
</script>

<template>
  <div class="menu">
    <template v-if="!session.keep && displayExpiryDate">
      <div class="balance">Kept until {{ displayExpiryDate.formatted }}</div>
      <Button size="xs" @click="onKeep">
        <font-awesome-icon icon="fa-solid fa-heart"></font-awesome-icon>
        Keep
      </Button>
    </template>
    <Button size="xs" @click="onDelete">
      <font-awesome-icon icon="fa-solid fa-trash"></font-awesome-icon>
      Delete
    </Button>
    <Button size="xs" @click="onShare">
      <font-awesome-icon icon="fa-solid fa-share"></font-awesome-icon>
      Share
    </Button>
    <Button
      size="xs"
      tag-name="a"
      :href="session.downloadFiles.flac"
      target="_blank"
      download
      color="primary"
      variant="ghost"
    >
      <font-awesome-icon icon="fa-solid fa-download"></font-awesome-icon>
      flac
    </Button>
  </div>
  <Modal :open="modalProps.open.value" @close="modalProps.onClose">
    <template #header>Are you sure?</template>
    <template #body
      >You are about to permanently delete a session recording?
    </template>
    <template #footer>
      <Button @click="modalProps.onConfirm" variant="ghost" color="neutral">
        Delete
      </Button>
      <Button @click="modalProps.onClose" variant="solid" color="primary">
        Keep
      </Button>
    </template>
  </Modal>

  <ShareModal
    :open="showShareModal"
    :item-name="sessionName"
    :is-loading="isSharing"
    @close="onShareClose"
    @share="onShareConfirm"
  />
</template>

<style scoped>
.menu {
  display: flex;
  align-items: center;
  gap: var(--size-1);
}

.balance {
  font-size: var(--scale-0);
  color: var(--color-red-500);
  margin: 0 var(--size-1);
}
</style>
