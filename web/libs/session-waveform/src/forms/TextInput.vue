<template>
  <div
    :class="['input-container', { [size]: true, [variant]: true }]"
    v-bind="inheritedAttrs.container"
  >
    <div v-if="$slots.prepend" class="prepend">
      <slot name="prepend" v-bind="{ inputProps, resetProps }" />
    </div>
    <input v-model="value" v-bind="inputProps" />
    <div v-if="$slots.actions" class="actions">
      <slot name="actions" v-bind="{ inputProps, resetProps }" />
    </div>
    <div v-if="$slots.append" class="append">
      <slot name="append" v-bind="{ inputProps, resetProps }" />
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number">
import { computed, useAttrs } from 'vue';

const props = withDefaults(
  defineProps<{
    modelValue?: T;
    disableReset?: boolean;
    size?: 'xs' | 'sm' | 'md' | 'lg';
    variant?: 'ghost' | 'outlined';
  }>(),
  {
    size: 'sm',
    variant: 'outlined',
    disableReset: false,
  }
);

const emit = defineEmits<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (e: 'update:modelValue', value?: any): void;
}>();

defineOptions({
  inheritAttrs: false,
});
const attrs = useAttrs();

const inheritedAttrs = computed(() => {
  const { style, class: className, ...rest } = attrs;

  return {
    container: { style, class: className } as object,
    input: rest,
  };
});

const inputProps = computed(() => {
  return inheritedAttrs.value.input;
});

const value = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
});

const resetProps = computed(() => {
  return {
    disabled: !value.value,
    reset: () => {
      value.value = undefined;
    },
  };
});
</script>

<style scoped>
.input-container {
  --input-height: var(--size-10);
  --input-padding: 0.25rem;
  --input-font-size: calc(var(--input-height) * 0.5);
  --input-line-height: 1;

  display: flex;
  align-items: center;
  gap: 0.25rem;

  transition: all 0.5s;
  border: 1px solid transparent;

  background: var(--color-grey-200);

  padding: 0;
  height: var(--input-height);
  line-height: var(--input-font-size);
  border-radius: var(--radius-sm);
}

.input-container.xs {
  --input-height: 1.375rem;
  --input-padding: 0.175rem;
}

.input-container.sm {
  --input-height: 1.5rem;
  --input-padding: 0.175rem;
}

.input-container.lg {
  --input-height: 2.75rem;
}

.input-container.outlined {
  border-color: var(--color-grey-200);
  background: var(--color-grey-50);
  border-width: 2px;
}

.input-container:focus-within {
  border-color: var(--color-grey-500);
  background: white;
}

.input-container.outlined:focus-within {
  border-color: var(--color-grey-500);
}

.prepend,
.append {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  color: var(--color-purple-700);
  font-size: calc(var(--input-font-size) * 0.9);
  line-height: var(--input-line-height);
  min-width: calc(var(--input-height) - 4px);
  height: calc(var(--input-height) - 4px);
}

.prepend {
  border-radius: var(--radius-sm) 0 0 var(--radius-sm);
}

input {
  all: unset;
  display: block;
  appearance: textfield;
  outline: none !important;
  border: none;
  background: none;
  padding: 0;
  width: 100%;
  min-width: 0;
  color: var(--color-grey-800);
  font-size: var(--input-font-size);
  line-height: var(--input-line-height);
  font-weight: var(--weight-medium);
}

input:focus {
  outline: none;
  box-shadow: none;
  border: none;
}

input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

input::placeholder {
  color: var(--color-grey-500);
}
</style>
