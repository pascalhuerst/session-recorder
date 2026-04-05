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
    <transition-group name="toast" tag="div" class="toast-list">
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
  max-width: 400px;
  max-height: 600px;
  pointer-events: none;
  overflow: hidden;
}

.toast-list {
  display: flex;
  flex-direction: column;
  gap: var(--size-3);
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
  border-color: #22c55e;
  background: #f0fdf4;
}

.toast--error {
  border-color: #ef4444;
  background: #fef2f2;
}

.toast--warning {
  border-color: #eab308;
  background: #fefce8;
}

.toast--info {
  border-color: #3b82f6;
  background: #eff6ff;
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
  color: #22c55e;
}

.toast--error .toast__icon {
  color: #ef4444;
}

.toast--warning .toast__icon {
  color: #eab308;
}

.toast--info .toast__icon {
  color: #3b82f6;
}

.toast__message {
  flex: 1;
  font-size: var(--scale-1);
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

<style>
.theme-dark .toast--success {
  background: #14532d;
  color: white;
}

.theme-dark .toast--error {
  background: #7f1d1d;
  color: white;
}

.theme-dark .toast--warning {
  background: #713f12;
  color: white;
}

.theme-dark .toast--info {
  background: #1e3a5f;
  color: white;
}

.theme-dark .toast .toast__message,
.theme-dark .toast .toast__close {
  color: white;
}
</style>
