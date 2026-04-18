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
            :class="[
              'modal-container rounded-[var(--radius-lg)] elevated-lg max-w-md w-full mx-4',
              props.class,
            ]"
            style="background: var(--strong-background-color)"
          >
            <div
              v-if="$slots.header || title"
              class="flex items-center justify-between px-6 py-4 border-b flex-shrink-0"
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
                class="modal-close-btn"
                @click="handleClose"
              >
                <BsX class="text-2xl" />
              </button>
            </div>
            <div class="modal-body px-6 py-5">
              <slot></slot>
            </div>
            <div
              v-if="$slots.footer"
              class="flex items-center justify-end gap-2 px-6 py-4 border-t flex-shrink-0"
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
/* ── 容器：限制高度不超出视口，内部滚动 ── */
.modal-container {
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 4rem);
}

/* ── 内容区域：溢出时滚动 ── */
.modal-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* ── 关闭按钮 ── */
.modal-close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm, 8px);
  flex-shrink: 0;
  color: var(--text-tertiary-color);
  background: transparent;
  transition:
    background 0.2s cubic-bezier(0.25, 1, 0.5, 1),
    color 0.2s cubic-bezier(0.25, 1, 0.5, 1);
}

.modal-close-btn:hover {
  background: var(--surface-hover);
  color: var(--text-color);
}

/* ── 动画 ── */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s cubic-bezier(0.25, 1, 0.5, 1);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-content-enter-active {
  transition: all 0.3s cubic-bezier(0.25, 1, 0.5, 1);
}

.modal-content-leave-active {
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-content-enter-from,
.modal-content-leave-to {
  opacity: 0;
  transform: scale(0.97);
}
</style>
