<template>
  <div
    :class="[
      'p-4 rounded-md border',
      type === 'error'
        ? 'bg-red-50 dark:bg-red-900/30 text-red-900 dark:text-red-200 border-red-200 dark:border-red-800'
        : '',
      type === 'warning'
        ? 'bg-yellow-50 dark:bg-yellow-900/30 text-yellow-900 dark:text-yellow-200 border-yellow-200 dark:border-yellow-800'
        : '',
      type === 'info'
        ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-900 dark:text-blue-200 border-blue-200 dark:border-blue-800'
        : '',
      type === 'success'
        ? 'bg-green-50 dark:bg-green-900/30 text-green-900 dark:text-green-200 border-green-200 dark:border-green-800'
        : '',
    ]"
  >
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
interface Props {
  type?: 'error' | 'warning' | 'info' | 'success';
}

withDefaults(defineProps<Props>(), {
  type: 'info',
});
</script>
