<template>
  <component :is="componentType" :style="style" ref="target">
    <template v-if="targetIsVisible">
      <slot />
    </template>
  </component>
</template>

<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core';
import { computed, onUnmounted, ref } from 'vue';
import { type Component } from 'vue';

const props = defineProps<{
  as?: Component | string;
  minHeight: number;
  preloadMargin?: number;
}>();

const target = ref(null);
const targetIsVisible = ref(false);

const componentType = computed(() => {
  return props.as || 'div';
});

const style = computed(() => ({
  minHeight: `${props.minHeight}px`,
}));

const { stop } = useIntersectionObserver(
  target,
  ([{ intersectionRatio }]) => {
    targetIsVisible.value = intersectionRatio > 0;
  },
  {
    rootMargin: `${props.preloadMargin || 0}px`,
  }
);

onUnmounted(() => {
  stop();
});
</script>

<style scoped></style>
