<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import Button from '../../../lib/controls/Button.vue';
import TextInput from '../../../lib/forms/TextInput.vue';
import { usePeaksContext } from '../../../context/usePeaksContext';
import { computed } from 'vue';

const { commandEmitter, state } = usePeaksContext();

const zoomLevel = computed(() => state.toRef().value.zoom.zoomLevel);
const zoomStep = computed(() => state.toRef().value.zoom.zoomStep);

const normalizeZoom = (offset: number) => {
  return Math.max(zoomStep.value, zoomLevel.value + offset);
};

const onZoomOut = () => {
  commandEmitter.emit('setZoomLevel', normalizeZoom(-1 * zoomStep.value));
};
const onZoomIn = () => {
  commandEmitter.emit('setZoomLevel', normalizeZoom(zoomStep.value));
};
</script>

<template>
  <TextInput
    type="number"
    v-model="zoomLevel"
    :step="zoomStep"
    :min="0"
    size="md"
    variant="outlined"
    color="neutral"
  >
    <template #prepend>
      <font-awesome-icon icon="fa-solid fa-arrows-left-right-to-line" />
    </template>

    <template #actions>
      <div class="input__actions">
        <Button
          shape="square"
          size="xs"
          color="primary"
          variant="ghost"
          @click="onZoomOut"
          :disabled="zoomLevel <= 0"
        >
          <font-awesome-icon icon="fa-solid fa-minus" />
        </Button>

        <Button
          shape="square"
          size="xs"
          color="primary"
          variant="ghost"
          @click="onZoomIn"
        >
          <font-awesome-icon icon="fa-solid fa-plus" />
        </Button>
      </div>
    </template>
  </TextInput>
</template>

<style scoped>
.input__actions {
  display: flex;
  height: 100%;
  gap: var(--input-padding);
  margin: var(--input-padding);
}
</style>

<i18n locale="en" lang="json"></i18n>
