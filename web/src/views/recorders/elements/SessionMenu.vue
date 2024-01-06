<script setup lang="ts">
import { type SessionInfo } from '@session-recorder/protocols/ts/sessionsource';
import { computed } from 'vue';
import { setKeepSession } from '@/grpc/procedures/setKeepSession';
import { deleteSession } from '@/grpc/procedures/deleteSession';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { useSessionData } from '@/useSessionData';
import {
  Button,
  Modal,
  useConfirmation,
} from '@session-recorder/session-waveform';

// @todo: break this down and make composable

const props = defineProps<{
  session: SessionInfo;
  recorderId: string;
}>();

const { awaitConfirmation, modalProps } = useConfirmation();

const { audioUrls } = useSessionData({
  sessionId: props.session.ID,
  recorderId: props.recorderId,
});

const ttl = computed(() => {
  if (!props.session.lifetime) {
    return undefined;
  }
  const val = Math.floor(props.session.lifetime.seconds / 60);
  if (val > 0) {
    return `${val} hours`;
  }

  const minutes = Math.floor(props.session.lifetime.seconds);
  return `${minutes} minutes`;
});

const onKeep = () => setKeepSession({ streamID: props.session.ID });
const onDelete = () => {
  awaitConfirmation().then(({ isConfirmed }) => {
    if (isConfirmed) {
      deleteSession({ streamID: props.session.ID });
    }
  });
};
</script>

<template>
  <div class="menu">
    <div v-if="ttl && !session.keep" class="balance">
      {{ ttl }} until deleted
    </div>
    <Button size="xs" v-if="!session.keep" @click="onKeep">
      <font-awesome-icon icon="fa-solid fa-heart"></font-awesome-icon>
      Keep
    </Button>
    <Button size="xs" @click="onDelete">
      <font-awesome-icon icon="fa-solid fa-trash"></font-awesome-icon>
      Delete
    </Button>
    <Button
      tag-name="router-link"
      size="xs"
      :to="`/recorders/${recorderId}/sessions/${session.ID}`"
    >
      <font-awesome-icon icon="fa-solid fa-thumbtack"></font-awesome-icon>
      Markers
    </Button>
    <template v-for="audioUrl in audioUrls" :key="audioUrl.src">
      <Button
        size="xs"
        tag-name="a"
        :href="audioUrl.src"
        target="_blank"
        download
        color="primary"
        variant="ghost"
      >
        <font-awesome-icon icon="fa-solid fa-download"></font-awesome-icon>
        {{ audioUrl.type.split('/').at(-1) }}
      </Button>
    </template>
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
