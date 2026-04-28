import { ref } from 'vue';

interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title?: string;
  message: string;
  duration?: number;
  timestamp: number;
}

const notifications = ref<Notification[]>([]);

let nextId = 0;

function addNotification(
  type: Notification['type'],
  message: string,
  title?: string,
  duration = 3000
) {
  const id = `notification-${nextId++}`;
  const notification: Notification = { id, type, message, title, duration, timestamp: Date.now() };
  notifications.value.push(notification);
  if (duration > 0) {
    setTimeout(() => removeNotification(id), duration);
  }
}

function removeNotification(id: string) {
  const index = notifications.value.findIndex((n) => n.id === id);
  if (index > -1) notifications.value.splice(index, 1);
}

function clearAllNotifications() {
  notifications.value = [];
}

// Convenience methods (simple toast style, same API as old useMessage)
const success = (content: string, duration?: number) =>
  addNotification('success', content, undefined, duration);
const error = (content: string, duration?: number) =>
  addNotification('error', content, undefined, duration);
const warning = (content: string, duration?: number) =>
  addNotification('warning', content, undefined, duration);
const info = (content: string, duration?: number) =>
  addNotification('info', content, undefined, duration);

export const useNotification = () => ({
  notifications,
  addNotification,
  removeNotification,
  clearAllNotifications,
  success,
  error,
  warning,
  info,
});

export type { Notification };
