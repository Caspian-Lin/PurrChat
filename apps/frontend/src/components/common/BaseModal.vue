<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="fixed inset-0 z-50 flex items-center justify-center"
        style="background: var(--modal-overlay-color)"
        @mousedown.self="handleOverlayMouseDown"
        @click.self="handleOverlayClick"
      >
        <Transition name="modal-content">
          <div
            v-if="show"
            :class="['rounded-[var(--radius-lg)] elevated-lg max-w-md w-full mx-4', props.class]"
            style="background: var(--strong-background-color)"
          >
            <div
              v-if="$slots.header || title"
              class="flex items-center justify-between px-6 py-4 border-b"
              style="border-color: var(--border-color)"
            >
              <h2
                v-if="title"
                class="text-xl font-bold font-display"
                style="color: var(--text-color)"
              >
                {{ title }}
              </h2>
              <slot name="header"></slot>
              <button
                v-if="closable"
                @click="handleClose"
                class="bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
              >
                <BsX class="text-2xl" />
              </button>
            </div>
            <div class="px-6 py-5">
              <slot></slot>
            </div>
            <div
              v-if="$slots.footer"
              class="flex items-center justify-end gap-2 px-6 py-4 border-t"
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
import { ref, watch, onBeforeUnmount } from 'vue';
import { BsX } from 'vue-icons-plus/bs';
interface Props {
  show: boolean;
  title?: string;
  closable?: boolean;
  closeOnOverlayClick?: boolean;
  class?: string;
}

const props = withDefaults(defineProps<Props>(), {
  closable: true,
  closeOnOverlayClick: true,
});

const emit = defineEmits<{
  'update:show': [value: boolean];
  close: [];
}>();

// 记录 mousedown 是否发生在 overlay 上，确保完整的点击（按下+释放）都在 overlay 区域
const mouseDownOnOverlay = ref(false);

const handleClose = () => {
  emit('update:show', false);
  emit('close');
};

const handleOverlayMouseDown = () => {
  mouseDownOnOverlay.value = true;
};

const handleDocumentMouseUp = () => {
  mouseDownOnOverlay.value = false;
};

const handleOverlayClick = () => {
  if (props.closeOnOverlayClick && mouseDownOnOverlay.value) {
    handleClose();
  }
  mouseDownOnOverlay.value = false;
};

// modal 显示时监听全局 mouseup，确保拖出 modal 后释放能正确重置状态
watch(
  () => props.show,
  (show) => {
    if (show) {
      document.addEventListener('mouseup', handleDocumentMouseUp);
    } else {
      document.removeEventListener('mouseup', handleDocumentMouseUp);
    }
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  document.removeEventListener('mouseup', handleDocumentMouseUp);
});
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s cubic-bezier(0.25, 1, 0.5, 1);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-content-enter-active,
.modal-content-leave-active {
  transition: all 0.3s cubic-bezier(0.25, 1, 0.5, 1);
}

.modal-content-enter-from,
.modal-content-leave-to {
  opacity: 0;
  transform: scale(0.97);
}
</style>
