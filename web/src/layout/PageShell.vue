<script setup lang="ts">
import { ref } from 'vue';
import SettingsModal from '../components/SettingsModal.vue';

const settingsOpen = ref(false);
</script>

<template>
  <div class="page-shell">
    <aside class="sidebar">
      <div class="sidebar-header">
        <router-link to="/" class="logo">
          <img src="/assets/logo.png" alt="Session Recorder logo" />
          <span class="logo-text">Session Recorder</span>
        </router-link>
      </div>
      <div class="sidebar-content">
        <slot name="sidebar" />
      </div>
      <div class="sidebar-footer">
        <button class="settings-btn" @click="settingsOpen = true">
          <font-awesome-icon icon="fa-solid fa-gear" />
          <span>Settings</span>
        </button>
      </div>
    </aside>
    <div class="main-wrapper">
      <main>
        <slot />
      </main>
    </div>
  </div>
  <SettingsModal :open="settingsOpen" @close="settingsOpen = false" />
</template>

<style scoped>
.page-shell {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  display: flex;
  flex-direction: column;
  width: 240px;
  min-width: 240px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-primary);
}

.sidebar-header {
  padding: var(--size-4);
  border-bottom: 1px solid var(--border-primary);
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--size-2);
  text-decoration: none;
  color: var(--text-primary);
}

.logo img {
  width: 32px;
  height: 32px;
  border-radius: 50%;
}

:global(.theme-dark) .logo img {
  box-shadow: 0 0 0 1px var(--border-primary);
}

.logo-text {
  font-size: var(--scale-1);
  font-weight: var(--weight-semibold);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--size-2);
}

.sidebar-footer {
  padding: var(--size-2) var(--size-4);
  border-top: 1px solid var(--border-primary);
}

.settings-btn {
  display: flex;
  align-items: center;
  gap: var(--size-2);
  width: 100%;
  padding: var(--size-2);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: var(--scale-0);
  font-family: inherit;
  cursor: pointer;
  transition: all 0.15s ease;
}

.settings-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.main-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow-y: auto;
  height: 100vh;
}

main {
  flex: 1;
}
</style>
