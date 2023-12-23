<script setup lang="ts">
import { toRef, watch } from "vue";
import VisuallyHidden from "../display/VisuallyHidden/VisuallyHidden.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    size?: "small" | "medium";
  }>(),
  {
    size: "small"
  }
);

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
}>();

const isChecked = toRef(props.modelValue);
watch(isChecked, () => {
  emit("update:modelValue", isChecked.value);
});
</script>

<template>
  <label :class="['control', { [props.size]: true }]">
    <VisuallyHidden>
      <input type="checkbox" v-model="isChecked" />
    </VisuallyHidden>
    <span class="cluster">
            <span :class="['switch', { 'is-checked': isChecked }]">
                <span class="slider"></span>
            </span>

            <slot></slot>
        </span>
  </label>
</template>

<style scoped>
.control.small {
  --switch-size: 12px;
}

.control.medium {
  --switch-size: 16px;
}

.cluster {
  display: flex;
  align-items: center;
  gap: var(--size-2);
}

.switch {
  position: relative;
  transition: all 0.25s;
  border: 1px solid var(--color-grey-600);
  border-radius: calc(var(--switch-size) + 1px);
  width: calc(var(--switch-size) * 2.3);
  min-width: calc(var(--switch-size) * 2.3);
  height: calc(var(--switch-size) + 4px);
}

.slider {
  display: block;
  position: absolute;
  top: 50%;
  left: 1px;
  transform: translateY(-50%);
  transition: left 0.25s;
  border-radius: 50%;
  background-color: var(--color-grey-600);
  width: var(--switch-size);
  height: var(--switch-size);
}

.switch.is-checked {
  background-color: var(--color-grey-600);
}

.switch.is-checked .slider {
  background-color: white;
  left: calc(100% - 1px - var(--switch-size));
}

.control:has(input:focus-visible) .slider {
  outline: 1px solid white;
  outline-offset: 2px;
}

:slotted(span) {
  color: var(--color-grey-600);
  font-size: var(--scale-0);
  font-weight: normal;
}

.control.small :slotted(span) {
  font-size: var(--scale-00);
}

.control.medium :slotted(span) {
  font-size: var(--scale-00);
}
</style>
