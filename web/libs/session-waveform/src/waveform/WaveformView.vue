<script setup lang="ts">
import { ref, watchEffect, computed, nextTick, onBeforeUnmount } from 'vue';
import VirtualizedItem from '../lib/disclosure/VirtualizedItem.vue';
import { usePeaksContext } from '../context/usePeaksContext';
import Overview from '../elements/Overview/Overview.vue';
import Zoomview from '../elements/Zoomview/Zoomview.vue';
import Audio from '../elements/Overview/Audio.vue';
import Segments from '../elements/Segments/Segments.vue';
import { useWaverformLayoutProvider } from './useWaverformLayoutProvider';

const props = withDefaults(
  defineProps<{
    height?: number;
  }>(),
  {
    height: 200,
  }
);

const context = usePeaksContext();
const { provide } = useWaverformLayoutProvider();

const overviewRef = ref<HTMLElement>();
const zoomviewRef = ref<HTMLElement>();
const audioRef = ref<HTMLElement>();

// Local expanded state for CSS control - mirrors context state
const isExpanded = ref(context.state.select((st) => st.expanded));

provide({
  overviewRef,
  zoomviewRef,
  audioRef,
});

watchEffect(() => {
  if (overviewRef.value && zoomviewRef.value && audioRef.value) {
    context.commandEmitter.emit('mount', {
      overview: overviewRef.value,
      zoomview: zoomviewRef.value,
      audio: audioRef.value,
    });
  }
});

// Listen for expanded state changes from context
context.eventEmitter.on('expandedChanged', (expanded) => {
  isExpanded.value = expanded;
});

// Expose toggle function for parent components
const toggleExpanded = async () => {
  const newExpanded = !isExpanded.value;

  if (newExpanded) {
    // Expanding: first show container so it gets dimensions
    isExpanded.value = true;
    // Wait for DOM update, then create zoomview
    await nextTick();
    requestAnimationFrame(() => {
      context.commandEmitter.emit('setExpanded', true);
    });
  } else {
    // Collapsing: destroy zoomview first, then hide container
    context.commandEmitter.emit('setExpanded', false);
  }
};

onBeforeUnmount(() => {
  context.commandEmitter.emit('destroy');
});

defineExpose({
  toggleExpanded,
  isExpanded,
});

// Compute min height based on expanded state
// Overview is ~80px, expanded adds zoomview (~100px) + segments
const minHeight = computed(() => (isExpanded.value ? props.height : 80));
</script>

<template>
  <VirtualizedItem
    :min-height="minHeight"
    :preload-margin="400"
    class="canvas"
  >
    <Overview />
    <!--
      Zoomview must always be in DOM with dimensions for Peaks.js to initialize.
      When collapsed, we hide it visually but keep it rendered.
    -->
    <div class="expanded-content" :class="{ collapsed: !isExpanded }">
      <Zoomview />
      <Segments />
    </div>
    <Audio />
  </VirtualizedItem>
  <slot />
</template>

<style scoped>
.canvas {
  position: relative;
  width: 100%;
  border-top: 1px solid var(--border-primary);
  border-bottom: 1px solid var(--border-primary);
  overflow: hidden;
}

.expanded-content {
  /* Normal state - visible */
  width: 100%;
}

.expanded-content.collapsed {
  /*
    Hide visually but keep dimensions for Peaks.js initialization.
    Position absolute takes it out of flow (no layout impact when collapsed),
    width: 100% ensures it gets parent width, visibility: hidden hides it.
  */
  position: absolute;
  width: 100%;
  visibility: hidden;
  pointer-events: none;
}
</style>
