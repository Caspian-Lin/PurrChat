<template>
  <Transition name="save-btn">
    <button
      v-if="isDirty"
      :disabled="isSaving"
      class="fixed bottom-6 right-6 z-40 flex items-center gap-2 px-5 py-2.5 text-white font-medium text-sm rounded-[var(--radius-sm,8px)] shadow-lg transition-all duration-200 hover:opacity-90 active:scale-[0.98] disabled:opacity-60 disabled:cursor-not-allowed"
      :style="{ backgroundColor: 'var(--theme-primary)' }"
      @click="$emit('save')"
    >
      <svg
        v-if="isSaving"
        class="w-4 h-4 animate-spin"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      <svg
        v-else
        class="w-4 h-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
      </svg>
      <span>{{ isSaving ? '保存中...' : '保存更改' }}</span>
    </button>
  </Transition>
</template>

<script setup lang="ts">
interface Props {
  isDirty: boolean;
  isSaving: boolean;
}

defineProps<Props>();

defineEmits<{
  save: [];
}>();
</script>

<style scoped>
.save-btn-enter-active {
  transition: all 200ms cubic-bezier(0.25, 1, 0.5, 1);
}
.save-btn-leave-active {
  transition: all 150ms cubic-bezier(0.16, 1, 0.3, 1);
}
.save-btn-enter-from {
  opacity: 0;
  transform: translateY(16px);
}
.save-btn-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
