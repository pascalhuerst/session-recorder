<script setup lang="ts">
import { toastService } from './ToastService';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

const toasts = toastService.toasts;

const getIcon = (type: string) => {
  switch (type) {
    case 'success':
      return 'fa-solid fa-check-circle';
    case 'error':
      return 'fa-solid fa-exclamation-circle';
    case 'warning':
      return 'fa-solid fa-exclamation-triangle';
    case 'info':
    default:
      return 'fa-solid fa-info-circle';
  }
};

const removeToast = (id: string) => {
  toastService.removeToast(id);
};
</script>

<template>
  <div class="toast-container">
    <transition-group name="toast" tag="div">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="['toast', `toast--${toast.type}`]"
      >
        <div class="toast__content">
          <font-awesome-icon :icon="getIcon(toast.type)" class="toast__icon" />
          <span class="toast__message">{{ toast.message }}</span>
          <button
            @click="removeToast(toast.id)"
            class="toast__close"
            aria-label="Close"
          >
            <font-awesome-icon icon="fa-solid fa-times" />
          </button>
        </div>
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: var(--size-4);
  right: var(--size-4);
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: var(--size-2);
  max-width: 400px;
  max-height: 600px;
  pointer-events: none;
  overflow: hidden;
}

.toast {
  pointer-events: auto;
  padding: var(--size-2);
  border-radius: var(--radius-xs);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border: 1px solid;
  background: var(--bg-primary);
  min-width: 300px;
}

.toast--success {
  border-color: var(--accent);
  background: var(--accent-subtle);
}

.toast--error {
  border-color: var(--color-red-500);
  background: var(--bg-secondary);
}

.toast--warning {
  border-color: var(--color-yellow-500);
  background: var(--bg-secondary);
}

.toast--info {
  border-color: var(--color-blue-500);
  background: var(--bg-secondary);
}

.toast__content {
  display: flex;
  align-items: center;
  gap: var(--size-2);
}

.toast__icon {
  flex-shrink: 0;
}

.toast--success .toast__icon {
  color: var(--accent);
}

.toast--error .toast__icon {
  color: var(--color-red-500);
}

.toast--warning .toast__icon {
  color: var(--color-yellow-500);
}

.toast--info .toast__icon {
  color: var(--color-blue-500);
}

.toast__message {
  flex: 1;
  font-size: var(--scale-1);
  color: var(--text-primary);
}

.toast__close {
  background: none;
  border: none;
  cursor: pointer;
  padding: var(--size-1);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
}

.toast__close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

/* Transition animations */
.toast-enter-active {
  transition: all 0.3s ease-out;
}

.toast-leave-active {
  transition: all 0.3s ease-in;
}

.toast-enter-from {
  transform: translateX(100%);
  opacity: 0;
}

.toast-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

.toast-move {
  transition: transform 0.3s ease;
}
</style>
