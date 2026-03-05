<template>
  <input
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :class="[
      'w-full px-3 py-2 rounded-md outline-none transition-colors',
      'bg-bg-secondary border border-border-color',
      'text-text-primary placeholder-text-tertiary',
      'focus:border-[var(--theme-primary)]',
      disabled ? 'opacity-50 cursor-not-allowed' : '',
    ]"
    @input="handleInput"
  />
</template>

<script setup lang="ts">
interface Props {
  modelValue?: string | number;
  type?: 'text' | 'password' | 'email' | 'tel';
  placeholder?: string;
  disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
  type: 'text',
  placeholder: '',
  disabled: false,
});

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit('update:modelValue', target.value);
};
</script>
