import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

export type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'theme-preference';

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>(
    (localStorage.getItem(STORAGE_KEY) as Theme) || 'system'
  );

  const isDark = computed(() => {
    if (theme.value === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches;
    }
    return theme.value === 'dark';
  });

  const applyTheme = () => {
    document.documentElement.classList.toggle('theme-dark', isDark.value);
  };

  const setTheme = (value: Theme) => {
    theme.value = value;
    localStorage.setItem(STORAGE_KEY, value);
    applyTheme();
  };

  // Initialize theme on store creation
  applyTheme();

  // Listen for system preference changes
  window
    .matchMedia('(prefers-color-scheme: dark)')
    .addEventListener('change', () => {
      if (theme.value === 'system') {
        applyTheme();
      }
    });

  return { theme, isDark, setTheme };
});
