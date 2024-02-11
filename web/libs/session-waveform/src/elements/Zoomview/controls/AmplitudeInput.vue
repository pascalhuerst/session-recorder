<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import TextInput from '../../../lib/forms/TextInput.vue';
import Button from '../../../lib/controls/Button.vue';
import { usePeaksContext } from '../../../context/usePeaksContext';

const {
  amplitude: { amplitudeScale, amplitudeStep, adjustAmplitudeScale },
} = usePeaksContext();

const onIncrease = () => adjustAmplitudeScale(amplitudeStep.value);
const onDecrease = () => adjustAmplitudeScale(-1 * amplitudeStep.value);
</script>

<template>
  <TextInput
    type="number"
    v-model="amplitudeScale"
    :step="amplitudeStep"
    size="md"
    variant="outlined"
    color="neutral"
  >
    <template #prepend>
      <font-awesome-icon icon="fa-solid fa-arrow-up-right-dots" />
    </template>

    <template #actions>
      <div class="input__actions">
        <Button
          shape="square"
          size="xs"
          color="primary"
          variant="ghost"
          @click="onDecrease"
          :disabled="amplitudeScale <= 0"
        >
          <font-awesome-icon icon="fa-solid fa-minus" />
        </Button>

        <Button
          shape="square"
          size="xs"
          color="primary"
          variant="ghost"
          @click="onIncrease"
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
