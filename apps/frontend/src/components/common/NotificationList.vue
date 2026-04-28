<template>
  <div class="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-md">
    <TransitionGroup name="notification">
      <div
        v-for="notification in notifications"
        :key="notification.id"
        :class="[
          'p-4 rounded-lg shadow-lg min-w-[300px] cursor-pointer',
          getNotificationClass(notification.type),
        ]"
        @click="removeNotification(notification.id)"
      >
        <div class="flex items-start gap-3">
          <div class="flex-shrink-0">
            <span class="text-lg font-semibold">{{ getNotificationIcon(notification.type) }}</span>
          </div>
          <div class="flex-1 min-w-0">
            <div v-if="notification.title" class="font-semibold text-sm mb-1">
              {{ notification.title }}
            </div>
            <div class="text-sm opacity-90">{{ notification.message }}</div>
          </div>
          <button
            class="flex-shrink-0 text-text-tertiary hover:text-text-primary transition-colors"
            @click.stop="removeNotification(notification.id)"
          >
            ✕
          </button>
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { useNotification } from '../../composables/useNotification';

const { notifications, removeNotification } = useNotification();

const getNotificationClass = (type: 'success' | 'info' | 'warning' | 'error'): string => {
  const classes = {
    success: 'bg-green-50 text-green-900 dark:bg-green-900/20 dark:text-green-100',
    info: 'bg-blue-50 text-blue-900 dark:bg-blue-900/20 dark:text-blue-100',
    warning: 'bg-yellow-50 text-yellow-900 dark:bg-yellow-900/20 dark:text-yellow-100',
    error: 'bg-red-50 text-red-900 dark:bg-red-900/20 dark:text-red-100',
  };
  return classes[type] || classes.info;
};

const getNotificationIcon = (type: 'success' | 'info' | 'warning' | 'error'): string => {
  const icons = {
    success: '✓',
    info: 'ℹ',
    warning: '⚠',
    error: '✕',
  };
  return icons[type] || icons.info;
};
</script>

<style scoped>
.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.notification-move {
  transition: transform 0.3s ease;
}
</style>
