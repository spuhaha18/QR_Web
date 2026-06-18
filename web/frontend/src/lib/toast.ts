import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error';
export interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
  hiding?: boolean;
}

export const toasts = writable<ToastItem[]>([]);

let nextId = 0;

export function showToast(message: string, type: ToastType): void {
  const id = nextId++;
  toasts.update((list) => [...list, { id, type, message }]);

  // Match original UX: visible 4s, then 0.3s slide-out before removal.
  setTimeout(() => {
    toasts.update((list) => list.map((t) => (t.id === id ? { ...t, hiding: true } : t)));
    setTimeout(() => {
      toasts.update((list) => list.filter((t) => t.id !== id));
    }, 300);
  }, 4000);
}

export function showSuccess(message: string): void {
  showToast(message, 'success');
}

export function showError(message: string): void {
  showToast(message, 'error');
}
