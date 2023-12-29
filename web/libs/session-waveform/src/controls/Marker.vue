<script setup lang="ts">
import { computed } from 'vue';
import { parsePlayTime } from '../utils/parsePlayTime';

const props = defineProps<{
  index: string;
  time: number;
}>();

const playTime = computed(() => {
  return parsePlayTime(props.time);
});
</script>

<template>
  <div class="marker">
    <div class="badge">
      {{ index }}
    </div>
    <div>
      {{
        `${playTime.hours}:${playTime.minutes}:${playTime.seconds}`
      }}.<small>{{ playTime.milliseconds }}</small>
    </div>
  </div>
</template>

<style scoped>
.marker {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: var(--size-2);
}

.badge {
  width: var(--size-6);
  height: var(--size-6);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-xs);
  background-color: var(--color-pink-700);
  color: white;
  font-size: var(--scale-0);
  font-weight: var(--weight-bold);
}

small {
  font-size: var(--scale-000);
  color: var(--color-grey-700);
}
</style>
