<script setup lang="ts">
import PageShell from "../../lib/layout/PageShell.vue";
import Container from "../../lib/layout/Container.vue";
import DevicePicker from "../../shared/DevicePicker.vue";
import { storeToRefs } from "pinia";
import { useRecordersStore } from "../../store/useRecordersStore.ts";

const { recorders, selectedRecorderId } = storeToRefs(useRecordersStore());

</script>

<template>
  <PageShell>
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
  </PageShell>
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

