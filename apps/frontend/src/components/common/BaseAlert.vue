<template>
  <div
    :class="[
      'p-4 rounded-md',
      type === 'error' ? 'bg-red-100 text-red-800' : '',
      type === 'warning' ? 'bg-yellow-100 text-yellow-800' : '',
      type === 'info' ? 'bg-blue-100 text-blue-800' : '',
      type === 'success' ? 'bg-green-100 text-green-800' : '',
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
