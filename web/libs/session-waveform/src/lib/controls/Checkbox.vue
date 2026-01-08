<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    modelValue?: boolean;
    disabled?: boolean;
    indeterminate?: boolean;
    size?: 'sm' | 'md';
  }>(),
  {
    modelValue: false,
    disabled: false,
    indeterminate: false,
    size: 'md',
  }
);

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
}>();

const checked = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
});
</script>

<template>
  <label
    :class="[
      'checkbox',
      `is-${size}`,
      {
        'is-checked': modelValue,
        'is-indeterminate': indeterminate,
        'is-disabled': disabled,
      },
    ]"
  >
    <input
      type="checkbox"
      v-model="checked"
      :disabled="disabled"
      :indeterminate="indeterminate"
      class="checkbox__input"
    />
    <span class="checkbox__box">
      <font-awesome-icon
        v-if="indeterminate"
        icon="fa-solid fa-minus"
        class="checkbox__icon"
      />
      <font-awesome-icon
        v-else-if="modelValue"
        icon="fa-solid fa-check"
        class="checkbox__icon"
      />
    </span>
    <span v-if="$slots.default" class="checkbox__label">
      <slot />
    </span>
  </label>
</template>

<style scoped>
.checkbox {
  --checkbox-size: 1.25rem;
  --checkbox-border-color: var(--color-grey-400);
  --checkbox-bg-color: white;
  --checkbox-check-color: white;
  --checkbox-checked-bg: var(--color-purple-500);
  --checkbox-checked-border: var(--color-purple-500);

  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  user-select: none;

  &.is-sm {
    --checkbox-size: 1rem;
  }

  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.checkbox__input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
  pointer-events: none;
}

.checkbox__box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--checkbox-size);
  height: var(--checkbox-size);
  border: 2px solid var(--checkbox-border-color);
  border-radius: 3px;
  background-color: var(--checkbox-bg-color);
  transition: all 0.15s ease-in-out;

  .checkbox.is-checked &,
  .checkbox.is-indeterminate & {
    background-color: var(--checkbox-checked-bg);
    border-color: var(--checkbox-checked-border);
  }

  .checkbox:hover:not(.is-disabled) & {
    border-color: var(--color-purple-400);
  }

  .checkbox:focus-within & {
    outline: 2px solid var(--color-purple-300);
    outline-offset: 2px;
  }
}

.checkbox__icon {
  color: var(--checkbox-check-color);
  font-size: calc(var(--checkbox-size) * 0.6);
}

.checkbox__label {
  font-size: var(--scale-0);
  color: var(--color-grey-800);
}
</style>
