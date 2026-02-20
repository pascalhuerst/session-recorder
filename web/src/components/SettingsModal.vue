<script lang="ts" setup>
import { storeToRefs } from 'pinia';
import { Modal, Button } from '@session-recorder/session-waveform';
import { useThemeStore, type Theme } from '../store/useThemeStore';

defineProps<{
  open: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

const themeStore = useThemeStore();
const { theme } = storeToRefs(themeStore);

const themeOptions: { value: Theme; label: string }[] = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'system', label: 'System' },
];

function handleThemeChange(value: Theme) {
  themeStore.setTheme(value);
}

function handleClose() {
  emit('close');
}
</script>

<template>
  <Modal :open="open" :size="{ width: 360 }" @close="handleClose">
    <template #header>Settings</template>

    <template #body>
      <div class="settings-content">
        <div class="setting-row">
          <label class="setting-label">Theme</label>
          <div class="theme-options">
            <button
              v-for="option in themeOptions"
              :key="option.value"
              class="theme-option"
              :class="{ active: theme === option.value }"
              @click="handleThemeChange(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <Button size="md" variant="solid" color="primary" @click="handleClose">
        Done
      </Button>
    </template>
  </Modal>
</template>

<style scoped>
.settings-content {
  display: flex;
  flex-direction: column;
  gap: var(--size-4);
}

.setting-row {
  display: flex;
  flex-direction: column;
  gap: var(--size-2);
}

.setting-label {
  font-weight: var(--weight-medium);
  font-size: var(--scale-1);
  color: var(--text-primary);
}

.theme-options {
  display: flex;
  gap: var(--size-1);
  background: var(--bg-tertiary);
  padding: var(--size-1);
  border-radius: var(--radius-md);
}

.theme-option {
  flex: 1;
  padding: var(--size-2) var(--size-3);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--scale-0);
  font-family: inherit;
  cursor: pointer;
  transition: all 0.15s ease;
}

.theme-option:hover {
  color: var(--text-primary);
}

.theme-option.active {
  background: var(--bg-primary);
  color: var(--text-primary);
  box-shadow: var(--shadow-xs);
}
</style>
