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
import { useTimeAgo } from '@vueuse/core';
import { toastService } from '../../../services/Toaster/ToastService';
import ShareModal from '../../../components/ShareModal.vue';

// @todo: break this down and make composable

const props = defineProps<{
  session: Session;
  recorderId: string;
}>();

const { awaitConfirmation, modalProps } = useConfirmation();

const showShareModal = ref(false);

const sessionName = computed(() => {
  return props.session.name || 'Untitled Recording';
});

const expiresIn = computed(() => {
  const { expiresAt } = props.session;
  if (!expiresAt) {
    return null;
  }
  return useTimeAgo(expiresAt, { showSecond: false }).value;
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

const onShareConfirm = (emails: string[]) => {
  const recipientText = emails.length === 1 ? emails[0] : `${emails.length} recipients`;

  // Close dialog immediately and show info toast
  showShareModal.value = false;
  toastService.info(`Sending download link to ${recipientText}...`);

  // Send in background
  shareSession({
    recorderId: props.recorderId,
    sessionId: props.session.id,
    recipientEmails: emails,
  })
    .then(() => {
      toastService.success(`Download link sent to ${recipientText}`);
    })
    .catch((error) => {
      console.error('Failed to share session:', error);
      toastService.error('Failed to send email. Please try again.');
    });
};

const onShareClose = () => {
  showShareModal.value = false;
};
</script>

<template>
  <div class="menu">
    <template v-if="!session.keep && expiresIn">
      <div class="expiry">Expires {{ expiresIn }}</div>
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

.expiry {
  font-size: var(--scale-0);
  color: var(--text-muted);
  margin: 0 var(--size-1);
}

@media (max-width: 768px) {
  .menu {
    flex-wrap: wrap;
  }

  .expiry {
    width: 100%;
    margin: 0;
  }
}
</style>
