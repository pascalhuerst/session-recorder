<script setup lang="ts"></script>

<template>
  <div class="container">
    <slot />
  </div>
</template>

<style scoped>
.container {
  width: calc(100vw - 4rem);
  max-width: 1200px;
  margin: auto;
}
</style>
