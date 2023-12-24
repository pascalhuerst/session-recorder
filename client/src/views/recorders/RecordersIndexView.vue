<script setup lang="ts">
import { useRouter } from "vue-router";
import { watch } from "vue";
import Container from "../../lib/layout/Container.vue";
import DevicePicker from "../../components/elements/DevicePicker.vue";
import { createRecordersContext } from "../../context/createRecordersContext.ts";

const router = useRouter();

const { recorders, selectedRecorderId } = createRecordersContext();

watch(selectedRecorderId, () => {
  if (!selectedRecorderId.value) {
    const keys = Array.from(recorders.value.keys());
    router.push(`/recorders/${keys.at(0)}`);
  }
}, { immediate: true });
</script>

<template>
  <div class="navbar">
    <Container>
      <DevicePicker :recorders="recorders" :selected-recorder-id="selectedRecorderId" />
    </Container>
  </div>
  <div class="content">
    <Container>
      <router-view />
    </Container>
  </div>
</template>

<style scoped>
.navbar {
  padding: var(--size-3);
  background: var(--color-grey-100);
}

.content {
  margin-top: var(--size-16);
  padding: var(--size-3);
}
</style>