<script lang="ts" setup>
import { computed, onMounted, ref, useAttrs, watch } from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";

const props = defineProps<{
  open: boolean;
  // Use this for a fixed size modal
  size?: {
    width: number;
    height?: number;
  };
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const attrs = useAttrs();
defineOptions({
  inheritAttrs: false
});

const dialogEl = ref<HTMLDialogElement>();

const customStyle = computed(() =>
  props.size ? { width: `${props.size.width}px`, height: props.size.height ? `${props.size.height}px` : "auto" } : {}
);

onMounted(() => {
  if (props.open) {
    dialogEl.value?.showModal();
  }
});

watch(
  () => props.open,
  (open) => {
    if (open) {
      dialogEl.value?.showModal();
    } else {
      dialogEl.value?.close();
    }
  }
);

function close() {
  dialogEl.value?.close();
  emit("close");
}
</script>

<template>
  <Teleport to="#modals">
    <!-- Pressing Esc emits cancel -->
    <dialog ref="dialogEl" @cancel="close" v-bind="attrs">
      <div class="content-container" :style="customStyle">
        <header>
          <slot name="header"></slot>
          <button class="close" @click="close">
            <font-awesome-icon icon="fa-solid fa-times" />
          </button>
        </header>

        <section>
          <slot name="body"></slot>
        </section>

        <footer>
          <slot name="footer"></slot>
        </footer>
      </div>

    </dialog>
  </Teleport>
</template>

<style scoped>
dialog {
  display: none;
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translateX(-50%) translateY(-50%);
  border: 0;
  margin: 0;
  padding: var(--size-6);
  font-size: var(--scale-0);
  border-radius: var(--radius-sm);
}

dialog[open] {
  display: flex;
}

dialog::backdrop {
  /* This is --color-jet, but CSS variables don't work (yet) inside ::backdrop */
  background-color: rgba(36, 36, 36, 0.5);
}

.content-container {
  display: flex;
  flex-grow: 1;
  flex-direction: column;
  max-width: 90vw;
  max-height: 90vh;
}

header {
  display: flex;
  flex-grow: 0;
  flex-direction: row;
  align-items: flex-start;
  justify-content: space-between;
  font-weight: var(--weight-semibold);
  text-transform: uppercase;
  font-size: var(--scale-2);
}

.close {
  display: flex;
  width: var(--size-6);
  height: var(--size-6);
  align-items: center;
  justify-content: center;
  border: none;
  background: none;
  font-size: var(--scale-2);
}

section {
  flex-grow: 1;
  margin: var(--size-6) 0;
  overflow-y: auto;
}

footer {
  display: flex;
  flex-grow: 0;
  flex-direction: row;
  justify-content: flex-end;
  gap: var(--size-2);
}
</style>
