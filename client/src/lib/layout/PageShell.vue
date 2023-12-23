<script setup lang="ts">
import PageHeader from "./PageHeader.vue";
import Container from "./Container.vue";
</script>

<template>
  <header>
    <Container>
      <PageHeader />
    </Container>
  </header>
  <main>
    <slot />
  </main>
</template>

<style scoped>

</style>