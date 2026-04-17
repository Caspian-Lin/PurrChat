<template>
  <div
    class="p-4 rounded-md border"
    :style="{
      background: alertBgColor,
      color: alertTextColor,
      borderColor: alertBorderColor,
    }"
  >
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

type AlertType = 'error' | 'warning' | 'info' | 'success';

interface Props {
  type?: AlertType;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'info',
});

const getColor = (type: AlertType) => {
  const map: Record<AlertType, { bg: string; text: string; border: string }> = {
    error: {
      bg: 'var(--color-error-bg)',
      text: 'var(--color-error)',
      border: 'var(--color-error)',
    },
    warning: {
      bg: 'var(--color-warning-bg)',
      text: 'var(--color-warning)',
      border: 'var(--color-warning)',
    },
    info: { bg: 'var(--color-info-bg)', text: 'var(--color-info)', border: 'var(--color-info)' },
    success: {
      bg: 'var(--color-success-bg)',
      text: 'var(--color-success)',
      border: 'var(--color-success)',
    },
  };
  return map[type];
};

const alertBgColor = computed(() => getColor(props.type).bg);
const alertTextColor = computed(() => getColor(props.type).text);
const alertBorderColor = computed(() => getColor(props.type).border);
</script>
