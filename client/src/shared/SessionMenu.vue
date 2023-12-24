<script setup lang="ts">
import { type SessionInfo } from "protocols/ts/sessionsource.ts";
import { computed } from "vue";
import { setKeepSession } from "../grpc/procedures/setKeepSession.ts";
import { deleteSession } from "../grpc/procedures/deleteSession.ts";
import { useConfirmation } from "../lib/disclosure/useConfirmation.ts";
import Modal from "../lib/disclosure/Modal.vue";
import Button from "../lib/controls/Button.vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";

const props = defineProps<{
  session: SessionInfo
  audioUrls: Array<{ src: string, type: string }>
}>();

const { awaitConfirmation, modalProps } = useConfirmation();

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
  awaitConfirmation()
    .then(({ isConfirmed }) => {
      if (isConfirmed) {
        deleteSession({ streamID: props.session.ID });
      }
    });
};
</script>

<template>
  <div class="menu">
    <div v-if="ttl && !session.keepSession" class="balance">
      {{ ttl }} until deleted
    </div>
    <Button size="xs" v-if="!session.keepSession" @click="onKeep">
      Keep
    </Button>
    <Button size="xs" @click="onDelete">
      Delete
    </Button>
    <template v-for="audioUrl in audioUrls" :key="audioUrl.src">
      <Button size="xs" tag-name="a" :href="audioUrl.src" target="_blank" download color="primary" variant="ghost">
        <font-awesome-icon icon="fa-solid fa-download"></font-awesome-icon>
        {{ audioUrl.type.split("/").at(-1) }}
      </Button>
    </template>
  </div>
  <Modal :open="modalProps.open.value" @close="modalProps.onClose">
    <template #header>Are you sure?</template>
    <template #body>You are about to permanently delete a session recording?</template>
    <template #footer>
      <Button @click="modalProps.onConfirm" variant="ghost" color="neutral">
        Delete
      </Button>
      <Button @click="modalProps.onClose" variant="ghost" color="primary">
        Skip
      </Button>
    </template>
  </Modal>
</template>

<style scoped>
.menu {
  display: flex;
  align-items: center;
  gap: var(--size-2);
}

.balance {
  font-size: var(--scale-00);
  color: var(--color-red-500);
}

button {
  border: 0;
  background: none;
  font-size: var(--scale-00);
  color: var(--color-grey-600);
}
</style>