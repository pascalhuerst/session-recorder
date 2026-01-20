<script setup lang="ts"></script>

<template>
  <div class="header">
    <router-link to="/" class="logo">
      <img src="/assets/logo.png" alt="Session Recorder logo" />
      <span class="heading">Session Recorder</span>
    </router-link>
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
  color: var(--text-primary);
}

.logo img {
  max-height: var(--size-12);
  border-radius: 50%;
  transition: box-shadow 0.2s ease;
}

/* Add subtle ring around logo in dark mode for visibility */
:global(.theme-dark) .logo img {
  box-shadow: 0 0 0 2px var(--border-primary);
}

.heading {
  font-size: var(--scale-3);
  font-weight: var(--weight-semibold);
}

nav {
  margin-left: auto;
}
</style>
