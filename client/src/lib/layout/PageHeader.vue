<script setup lang="ts"></script>

<template>
  <div class="header">
    <router-link to="/" class="logo">
      <img src="../../assets/logo.png" alt="Logo" />
      <span class="heading">Session Recorder</span>
    </router-link>

    <nav>
      Menu
    </nav>
  </div>
</template>

<style scoped>
.header {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  padding: var(--size-8) 0;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  text-decoration: none;
  color: var(--color-black);
}

.logo img {
  max-height: var(--size-12);
}

.heading {
  font-size: var(--scale-3);
  font-weight: var(--weight-semibold);
}

nav {
  margin-left: auto;
}
</style>