<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { watch } from 'vue';
import { useRouter } from 'vue-router';
import { useRecordersStore } from '../../../../../../store/useRecordersStore';

const router = useRouter();

const { selectedRecorderId } = storeToRefs(useRecordersStore());

watch(
  selectedRecorderId,
  () => {
    router.replace(`/recorders/${selectedRecorderId.value}/sessions`);
  },
  { immediate: true },
);
</script>

<template>
  <slot />
</template>

<style scoped></style>
