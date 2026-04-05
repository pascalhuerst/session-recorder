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
  top: var(--size-6);
  right: var(--size-6);
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: var(--size-3);
  max-width: 400px;
  max-height: 600px;
  pointer-events: none;
  overflow: hidden;
}

.toast {
  pointer-events: auto;
  padding: var(--size-3);
  border-radius: var(--radius-sm);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border: 1px solid;
  background: var(--bg-primary);
  min-width: 300px;
}

.toast--success {
  border-color: var(--color-green-500);
  background: var(--color-green-50);
}

:global(.theme-dark) .toast--success {
  background: var(--color-green-950);
}

.toast--error {
  border-color: var(--color-red-500);
  background: var(--color-red-50);
}

:global(.theme-dark) .toast--error {
  background: var(--color-red-950);
}

.toast--warning {
  border-color: var(--color-yellow-500);
  background: var(--color-yellow-50);
}

:global(.theme-dark) .toast--warning {
  background: var(--color-yellow-950);
}

.toast--info {
  border-color: var(--color-blue-500);
  background: var(--color-blue-50);
}

:global(.theme-dark) .toast--info {
  background: var(--color-blue-950);
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
  color: var(--color-green-500);
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

@media (max-width: 768px) {
  .toast-container {
    right: var(--size-3);
    left: var(--size-3);
    max-width: none;
  }

  .toast {
    min-width: 0;
  }
}
</style>
