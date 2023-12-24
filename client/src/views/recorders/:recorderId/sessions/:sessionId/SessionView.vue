<script setup lang="ts">
import DevicePicker from "./elements/DevicePicker.vue";
import OpenSessionList from "./elements/OpenSessionsList.vue";
import Container from "../../../../../lib/layout/Container.vue";
</script>

<template>
  <div class="navbar">
    <Container>
      <DevicePicker />
    </Container>
  </div>
  <div class="sessions">
    <Container>
      <OpenSessionList />
    </Container>
  </div>
</template>

<style scoped>
.navbar {
  padding: var(--size-3);
  background: var(--color-grey-100);
}

.sessions {
  margin-top: var(--size-16);
  padding: var(--size-3);
}
</style>