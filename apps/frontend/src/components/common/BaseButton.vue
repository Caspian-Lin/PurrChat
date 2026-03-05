<template>
  <button
    :class="[
      'transition-all font-medium',
      // Size classes
      size === 'small' ? 'w-4 h-4 p-0' : '',
      size === 'medium' ? 'w-8 h-8 p-0' : '',
      size === 'large' ? 'w-12 h-12 p-0' : '',
      // Circle/Shape classes
      circle ? 'rounded-full' : 'rounded-md',
      !circle ? 'px-4 py-2' : '',
      // Type classes
      type === 'primary' ? 'bg-[var(--theme-primary)] text-white hover:opacity-80' : '',
      type === 'default' ? 'bg-bg-secondary text-text-primary hover:bg-hover-bg' : '',
      type === 'tertiary' ? 'bg-transparent text-text-primary hover:bg-hover-bg' : '',
      disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
      block ? 'w-full' : '',
    ]"
    :disabled="disabled"
    @click="handleClick"
  >
    <slot></slot>
  </button>
</template>

<script setup lang="ts">
interface Props {
  type?: 'primary' | 'default' | 'tertiary';
  disabled?: boolean;
  block?: boolean;
  circle?: boolean;
  size?: 'small' | 'medium' | 'large';
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  disabled: false,
  block: false,
  circle: false,
  size: 'medium',
});

const emit = defineEmits<{
  click: [event: MouseEvent];
}>();

const handleClick = (event: MouseEvent) => {
  if (!props.disabled) {
    emit('click', event);
  }
};
</script>
