<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import Button from '../../../lib/controls/Button.vue';
import TextInput from '../../../lib/forms/TextInput.vue';
import { usePeaksContext } from '../../../context/usePeaksContext';

const {
  zoom: { zoomLevel, zoomStep, adjustZoom },
} = usePeaksContext();

const onZoomOut = () => adjustZoom(-1 * zoomStep.value);
const onZoomIn = () => adjustZoom(zoomStep.value);
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
