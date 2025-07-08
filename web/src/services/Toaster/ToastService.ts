import { reactive } from 'vue';

export interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info' | 'warning';
  duration?: number;
  persistent?: boolean;
}

export interface ToastState {
  toasts: Toast[];
}

class ToastService {
  private state = reactive<ToastState>({
    toasts: [],
  });

  private toastCounter = 0;

  get toasts() {
    return this.state.toasts;
  }

  private generateId(): string {
    return `toast-${++this.toastCounter}-${Date.now()}`;
  }

  private addToast(toast: Omit<Toast, 'id'>): string {
    const id = this.generateId();
    const newToast: Toast = {
      id,
      duration: 4000,
      persistent: false,
      ...toast,
    };

    this.state.toasts.push(newToast);

    if (!newToast.persistent && newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        this.removeToast(id);
      }, newToast.duration);
    }

    return id;
  }

  success(
    message: string,
    options?: Partial<Pick<Toast, 'duration' | 'persistent'>>
  ): string {
    return this.addToast({
      message,
      type: 'success',
      ...options,
    });
  }

  error(
    message: string,
    options?: Partial<Pick<Toast, 'duration' | 'persistent'>>
  ): string {
    return this.addToast({
      message,
      type: 'error',
      duration: 6000, // Errors stay longer by default
      ...options,
    });
  }

  info(
    message: string,
    options?: Partial<Pick<Toast, 'duration' | 'persistent'>>
  ): string {
    return this.addToast({
      message,
      type: 'info',
      ...options,
    });
  }

  warning(
    message: string,
    options?: Partial<Pick<Toast, 'duration' | 'persistent'>>
  ): string {
    return this.addToast({
      message,
      type: 'warning',
      duration: 5000, // Warnings stay a bit longer
      ...options,
    });
  }

  removeToast(id: string): void {
    const index = this.state.toasts.findIndex((toast) => toast.id === id);
    if (index > -1) {
      this.state.toasts.splice(index, 1);
    }
  }

  clearAll(): void {
    this.state.toasts.length = 0;
  }
}

export const toastService = new ToastService();
