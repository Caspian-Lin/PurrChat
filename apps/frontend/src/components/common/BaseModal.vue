<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="fixed inset-0 z-50 flex items-center justify-center"
        style="background: var(--modal-overlay-color)"
        @click.self="handleOverlayClick"
      >
        <Transition name="modal-content">
          <div
            v-if="show"
            class="rounded-lg shadow-xl max-w-md w-full mx-4"
            style="background: var(--background-color)"
          >
            <div
              v-if="$slots.header || title"
              class="flex items-center justify-between p-4 border-b"
              style="border-color: var(--border-color)"
            >
              <h2 v-if="title" class="text-xl font-bold" style="color: var(--text-color)">
                {{ title }}
              </h2>
              <slot name="header"></slot>
              <button
                v-if="closable"
                @click="handleClose"
                class="text-text-tertiary hover:text-text-primary transition-colors"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-6 w-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>
            <div class="p-4">
              <slot></slot>
            </div>
            <div
              v-if="$slots.footer"
              class="flex items-center justify-end gap-2 p-4 border-t"
              style="border-color: var(--border-color)"
            >
              <slot name="footer"></slot>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
interface Props {
  show: boolean;
  title?: string;
  closable?: boolean;
  closeOnOverlayClick?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  closable: true,
  closeOnOverlayClick: true,
});

const emit = defineEmits<{
  'update:show': [value: boolean];
  close: [];
}>();

const handleClose = () => {
  emit('update:show', false);
  emit('close');
};

const handleOverlayClick = () => {
  if (props.closeOnOverlayClick) {
    handleClose();
  }
};
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-content-enter-active,
.modal-content-leave-active {
  transition: all 0.3s ease;
}

.modal-content-enter-from,
.modal-content-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
