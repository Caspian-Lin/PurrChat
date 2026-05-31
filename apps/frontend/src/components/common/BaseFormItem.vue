<template>
  <div>
    <label
      v-if="label"
      class="block text-sm font-medium mb-2"
      style="color: var(--text-secondary-color)"
    >
      {{ label }}
    </label>
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
interface Props {
  label?: string;
}

defineProps<Props>();
</script>
