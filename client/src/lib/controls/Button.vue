<script setup lang="ts">

const props = withDefaults(defineProps<{
  color?: "primary" | "neutral" | "outlined"
  variant?: "ghost" | "solid"
  size?: "md" | "sm" | "xs"
  isLoading?: boolean
  tagName?: string
}>(), {
  color: "neutral",
  size: "md",
  variant: "ghost",
  isLoading: false,
  tagName: "button"
});


</script>

<template>
  <component :is="tagName" :class="{
    'button': true,
    'is-loading': isLoading,
    [`is-${props.variant}`]: true,
    [`is-${props.color}`]: true,
    [`is-${props.size}`]: true,
  }">
    <slot />
  </component>
</template>

<style scoped>
.button {
  --btn-bg-color: transparent;
  --btn-bg-color-hover: var(--color-grey-100);
  --btn-text-color: var(--color-grey-800);
  --btn-text-color-hover: var(--color-grey-900);
  --btn-border-color: transparent;
  --btn-border-color-hover: transparent;
  --btn-padding-y: 0.25rem;
  --btn-padding-x: 1rem;
  --btn-font-size: var(--scale-1);
  --btn-icon-size: var(--scale-0);
  --btn-line-height: 1.15;
  --btn-font-weight: var(--weight-semibold);
  --btn-border-radius: 4px;
  --btn-min-height: var(--size-10);
  --btn-min-width: 6rem;

  display: inline-flex;
  gap: 0.25rem;
  align-items: center;
  justify-content: center;
  line-height: var(--btn-line-height);
  background-color: var(--btn-bg-color);
  color: var(--btn-text-color);
  border: 1px solid var(--btn-border-color);
  border-radius: var(--btn-border-radius);
  padding: var(--btn-padding-y) var(--btn-padding-x);
  font-size: var(--btn-font-size);
  font-weight: var(--btn-font-weight);
  text-transform: uppercase;
  text-decoration: none;
  min-width: var(--btn-min-width);
  min-height: var(--btn-min-height);
  cursor: pointer;
  transition: color 0.15s ease-in-out, background-color 0.15s ease-in-out, border-color 0.15s ease-in-out;

  &.is-sm {
    --btn-font-size: var(--scale-0);
    --btn-icon-size: var(--scale-0);
    --btn-min-height: var(--size-6);
    --btn-min-width: 0;
  }

  &.is-xs {
    --btn-font-size: var(--scale-00);
    --btn-icon-size: var(--scale-00);
    --btn-min-height: var(--size-4);
    --btn-min-width: 0;
    text-transform: none;
  }

  &.is-ghost {
    --btn-text-color: var(--color-grey-500);
    --btn-text-color-hover: var(--color-grey-900);
    --btn-text-color-disabled: #cccccc;

    &.is-primary {
      --btn-text-color: var(--color-purple-700);
      --btn-text-color-hover: var(--color-purple-900);
    }
  }

  &.is-outlined {
    --btn-text-color: var(--color-grey-900);
    --btn-text-color-hover: var(--color-black);
    --btn-border-color: var(--color-grey-500);
    --btn-border-color-hover: var(--color-grey-600);
    --btn-bg-color: white;
    --btn-text-color-disabled: #ccc;
    --btn-border-color-disabled: #ccc;
    --btn-bg-color-disabled: #ccc;

    &.is-primary {
      --btn-bg-color-hover: tranparent;
      --btn-text-color: var(--color-purple-500);
      --btn-text-color-hover: var(--color-purple-700);
      --btn-border-color: var(--color-purple-500);
      --btn-border-color-hover: var(--color-purple-700);
    }
  }

  &.is-solid {
    --btn-text-color: var(--color-grey-900);
    --btn-text-color-hover: var(--color-grey-900);
    --btn-border-color: var(--color-grey-200);
    --btn-bg-color: var(--color-grey-200);
    --btn-bg-color-hover: var(--color-grey-400);
    --btn-text-color-disabled: white;
    --btn-border-color-disabled: #ccc;
    --btn-bg-color-disabled: #ccc;

    &.is-primary {
      --btn-text-color: white;
      --btn-text-color-hover: white;
      --btn-border-color: var(--color-purple-500);
      --btn-border-color-hover: var(--color-purple-700);
      --btn-bg-color: var(--color-purple-500);
      --btn-bg-color-hover: var(--color-purple-700);
    }
  }

  @keyframes infinite-spinning {
    from {
      transform: rotate(0deg);
    }

    to {
      transform: rotate(360deg);
    }
  }

  &:hover,
  &:active {
    color: var(--btn-text-color-hover);
    background-color: var(--btn-bg-color-hover);
    border-color: var(--btn-border-color-hover);
  }

  &:focus {
    outline: 1px solid var(--btn-border-color-hover);
  }

  &:disabled {
    cursor: not-allowed;
    filter: grayscale(100%);

    background-color: var(--btn-bg-color-disabled);
    border-color: var(--btn-border-color-disabled);
    color: var(--btn-text-color-disabled);

    &:hover,
    &:active {
      background-color: var(--btn-bg-color-disabled);
      border-color: var(--btn-border-color-disabled);
      color: var(--btn-text-color-disabled);
    }
  }

  &.is--loading {
    position: relative;
    color: transparent !important;

    * {
      color: transparent !important;
    }

    &:hover,
    &:hover * {
      color: transparent;
    }

    &::after {
      content: '';
      position: absolute;
      margin: auto;
      left: 0;
      right: 0;
      top: 0;
      bottom: 0;
      background-color: transparent;
      height: calc(var(--btn-font-size) * var(--btn-line-height));
      width: calc(var(--btn-font-size) * var(--btn-line-height));
      animation: infinite-spinning 1s infinite linear;
      border-radius: 50%;
      border-width: 2px;
      border-color: transparent var(--btn-text-color) var(--btn-text-color);
      border-style: solid;
      display: block;
    }
  }

  :slotted(svg) {
    font-size: var(--btn-icon-size);
    margin-right: 0.25rem;
  }
}

</style>