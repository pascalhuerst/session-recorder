<script setup lang="ts">

const props = withDefaults(defineProps<{
  color?: "primary" | "neutral" | "outlined"
  variant?: "ghost" | "solid"
  isLoading?: boolean
}>(), {
  color: "neutral",
  variant: "solid",
  isLoading: false
});


</script>

<template>
  <button :class="{
    'button': true,
    'is-loading': isLoading,
    [`is-${props.variant}`]: true,
    [`is-${props.color}`]: true,
  }">
    <slot />
  </button>
</template>

<style scoped>
.button {
  --admin-btn-bg-color: transparent;
  --admin-btn-bg-color-hover: var(--color-grey-100);
  --admin-btn-text-color: var(--color-grey-800);
  --admin-btn-text-color-hover: var(--color-grey-900);
  --admin-btn-border-color: transparent;
  --admin-btn-border-color-hover: transparent;
  --admin-btn-padding-y: 0.25rem;
  --admin-btn-padding-x: 1rem;
  --admin-btn-font-size: calc(var(--scale-0) - 2px);
  --admin-btn-icon-size: var(--scale-0);
  --admin-btn-line-height: 1.15;
  --admin-btn-font-weight: var(--weight-semibold);
  --admin-btn-border-radius: 4px;
  --admin-btn-min-height: var(--size-8);
  --admin-btn-min-width: 6rem;

  display: inline-flex;
  gap: 0.25rem;
  align-items: center;
  justify-content: center;
  line-height: var(--admin-btn-line-height);
  background-color: var(--admin-btn-bg-color);
  color: var(--admin-btn-text-color);
  border: 1px solid var(--admin-btn-border-color);
  border-radius: var(--admin-btn-border-radius);
  padding: var(--admin-btn-padding-y) var(--admin-btn-padding-x);
  font-size: var(--admin-btn-font-size);
  font-weight: var(--admin-btn-font-weight);
  text-transform: uppercase;
  text-decoration: none;
  min-width: var(--admin-btn-min-width);
  min-height: var(--admin-btn-min-height);
  cursor: pointer;
  transition: color 0.15s ease-in-out, background-color 0.15s ease-in-out, border-color 0.15s ease-in-out;

  &.is-ghost {
    --admin-btn-text-color: var(--color-grey-800);
    --admin-btn-text-color-hover: var(--color-grey-900);
    --admin-btn-text-color-disabled: #cccccc;

    &.is-primary {
      --admin-btn-text-color: var(--color-purple-500);
      --admin-btn-text-color-hover: var(--color-purple-700);
    }
  }

  &.is-outlined {
    --admin-btn-text-color: var(--color-grey-900);
    --admin-btn-text-color-hover: var(--color-black);
    --admin-btn-border-color: var(--color-grey-500);
    --admin-btn-border-color-hover: var(--color-grey-600);
    --admin-btn-bg-color: white;
    --admin-btn-text-color-disabled: #ccc;
    --admin-btn-border-color-disabled: #ccc;
    --admin-btn-bg-color-disabled: #ccc;

    &.is-primary {
      --admin-btn-bg-color-hover: tranparent;
      --admin-btn-text-color: var(--color-purple-500);
      --admin-btn-text-color-hover: var(--color-purple-700);
      --admin-btn-border-color: var(--color-purple-500);
      --admin-btn-border-color-hover: var(--color-purple-700);
    }
  }

  &.is-solid {
    --admin-btn-text-color: var(--color-grey-900);
    --admin-btn-text-color-hover: var(--color-grey-900);
    --admin-btn-border-color: var(--color-grey-200);
    --admin-btn-bg-color: var(--color-grey-200);
    --admin-btn-bg-color-hover: var(--color-grey-400);
    --admin-btn-text-color-disabled: white;
    --admin-btn-border-color-disabled: #ccc;
    --admin-btn-bg-color-disabled: #ccc;

    &.is-primary {
      --admin-btn-text-color: white;
      --admin-btn-text-color-hover: white;
      --admin-btn-border-color: var(--color-purple-500);
      --admin-btn-border-color-hover: var(--color-purple-700);
      --admin-btn-bg-color: var(--color-purple-500);
      --admin-btn-bg-color-hover: var(--color-purple-700);
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
    color: var(--admin-btn-text-color-hover);
    background-color: var(--admin-btn-bg-color-hover);
    border-color: var(--admin-btn-border-color-hover);
  }

  &:focus {
    outline: 1px solid var(--admin-btn-border-color-hover);
  }

  &:disabled {
    cursor: not-allowed;
    filter: grayscale(100%);

    background-color: var(--admin-btn-bg-color-disabled);
    border-color: var(--admin-btn-border-color-disabled);
    color: var(--admin-btn-text-color-disabled);

    &:hover,
    &:active {
      background-color: var(--admin-btn-bg-color-disabled);
      border-color: var(--admin-btn-border-color-disabled);
      color: var(--admin-btn-text-color-disabled);
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
      height: calc(var(--admin-btn-font-size) * var(--admin-btn-line-height));
      width: calc(var(--admin-btn-font-size) * var(--admin-btn-line-height));
      animation: infinite-spinning 1s infinite linear;
      border-radius: 50%;
      border-width: 2px;
      border-color: transparent var(--admin-btn-text-color) var(--admin-btn-text-color);
      border-style: solid;
      display: block;
    }
  }

  svg {
    font-size: var(--admin-btn-icon-size);
    margin-right: 0.25rem;
  }
}

</style>