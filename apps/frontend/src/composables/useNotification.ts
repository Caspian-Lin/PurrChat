import { ref } from 'vue';

export interface Notification {
  id: string;
  type: 'success' | 'info' | 'warning' | 'error';
  title: string;
  message: string;
  duration?: number;
  timestamp: number;
}

const notifications = ref<Notification[]>([]);

export function useNotification() {
  const addNotification = (
    type: Notification['type'],
    title: string,
    message: string,
    duration: number = 3000
  ) => {
    const notification: Notification = {
      id: Date.now().toString() + Math.random().toString(),
      type,
      title,
      message,
      duration,
      timestamp: Date.now(),
    };

    notifications.value.unshift(notification);

    // 自动移除通知
    if (duration > 0) {
      setTimeout(() => {
        removeNotification(notification.id);
      }, duration);
    }
  };

  const removeNotification = (id: string) => {
    const index = notifications.value.findIndex((n) => n.id === id);
    if (index !== -1) {
      notifications.value.splice(index, 1);
    }
  };

  const clearAllNotifications = () => {
    notifications.value = [];
  };

  return {
    notifications,
    addNotification,
    removeNotification,
    clearAllNotifications,
  };
}
