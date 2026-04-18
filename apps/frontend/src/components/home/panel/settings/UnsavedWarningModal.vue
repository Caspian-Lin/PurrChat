<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="fixed inset-0 z-50 flex items-center justify-center"
        style="background: var(--modal-overlay-color)"
        @click.self="$emit('cancel')"
      >
        <div
          class="max-w-sm w-full mx-4 shadow-lg"
          :style="{
            backgroundColor: 'var(--strong-background-color)',
            borderRadius: 'var(--radius-lg, 16px)',
          }"
        >
          <div class="px-6 py-5">
            <h2 class="text-lg font-semibold text-text-primary mb-2">未保存的更改</h2>
            <p class="text-sm text-text-secondary">你有未保存的设置更改。离开后更改将丢失。</p>
          </div>
          <div
            class="flex justify-end gap-2 px-6 py-4"
            style="border-top: 1px solid var(--border-color)"
          >
            <button
              class="px-4 py-2 text-sm text-text-secondary rounded-[var(--radius-sm,8px)] transition-colors duration-200"
              style="background: transparent"
              onmouseenter="this.style.background = 'var(--hover-background)'"
              onmouseleave="this.style.background = 'transparent'"
              @click="$emit('cancel')"
            >
              取消
            </button>
            <button
              class="px-4 py-2 text-sm text-text-primary rounded-[var(--radius-sm,8px)] transition-colors duration-200"
              style="background: var(--surface-tertiary-color)"
              onmouseenter="this.style.background = 'var(--hover-background)'"
              onmouseleave="this.style.background = 'var(--surface-tertiary-color)'"
              @click="$emit('discard')"
            >
              不保存
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
interface Props {
  show: boolean;
}

defineProps<Props>();

defineEmits<{
  cancel: [];
  discard: [];
}>();
</script>

<style scoped>
.modal-enter-active {
  transition: opacity 200ms cubic-bezier(0.25, 1, 0.5, 1);
}
.modal-leave-active {
  transition: opacity 150ms cubic-bezier(0.16, 1, 0.3, 1);
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
