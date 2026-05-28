<template>
  <Teleport to="body">
    <Transition name="context-menu">
      <div
        v-if="visible"
        ref="menuRef"
        class="context-menu"
        :style="menuStyle"
      >
        <button
          v-for="action in actions"
          :key="action.key"
          class="context-menu__item"
          :class="{ 'context-menu__item--danger': action.danger }"
          @click="handleAction(action)"
        >
          <component v-if="action.icon" :is="action.icon" class="context-menu__item__icon" />
          <span>{{ action.label }}</span>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import type { Component } from 'vue';

export interface ContextMenuAction {
  key: string;
  label: string;
  icon?: Component;
  danger?: boolean;
  disabled?: boolean;
  handler: () => void;
}

interface Position {
  x: number;
  y: number;
}

interface Props {
  visible: boolean;
  position: Position;
  actions: ContextMenuAction[];
}

const props = defineProps<Props>();

const emit = defineEmits<{
  close: [];
}>();

const menuRef = ref<HTMLElement | null>(null);

// 计算菜单位置，确保不超出视口
const menuStyle = computed(() => {
  const style: Record<string, string> = {
    position: 'fixed',
    zIndex: '9999',
    background: 'var(--surface-color)',
    border: '1px solid var(--border-color)',
    borderRadius: 'var(--radius-md)',
    boxShadow: 'var(--shadow-lg)',
    padding: '4px',
    minWidth: '120px',
  };

  let x = props.position.x;
  let y = props.position.y;

  // 如果菜单已渲染，检查是否超出视口
  if (menuRef.value) {
    const rect = menuRef.value.getBoundingClientRect();
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    // 右侧超出，左移
    if (x + rect.width > viewportWidth) {
      x = viewportWidth - rect.width - 8;
    }

    // 底部超出，上移
    if (y + rect.height > viewportHeight) {
      y = viewportHeight - rect.height - 8;
    }

    // 确保不超出左边界和上边界
    x = Math.max(8, x);
    y = Math.max(8, y);
  }

  style.left = `${x}px`;
  style.top = `${y}px`;

  return style;
});

function handleAction(action: ContextMenuAction) {
  if (action.disabled) return;
  action.handler();
  emit('close');
}

// 点击外部关闭
function onClickOutside(event: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(event.target as Node)) {
    emit('close');
  }
}

// Escape 键关闭
function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    emit('close');
  }
}

// 滚动关闭
function onScroll() {
  emit('close');
}

onMounted(() => {
  document.addEventListener('mousedown', onClickOutside);
  document.addEventListener('keydown', onKeyDown);
  // 监听所有可滚动容器的滚动事件
  document.addEventListener('scroll', onScroll, true);
});

onUnmounted(() => {
  document.removeEventListener('mousedown', onClickOutside);
  document.removeEventListener('keydown', onKeyDown);
  document.removeEventListener('scroll', onScroll, true);
});

// visible 变化时重新计算位置
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      nextTick(() => {
        // 触发 computed 重新计算
        void menuStyle.value;
      });
    }
  }
);
</script>

<style scoped>
.context-menu-enter-active,
.context-menu-leave-active {
  transition: all 0.2s ease;
}

.context-menu-enter-from,
.context-menu-leave-to {
  opacity: 0;
  transform: scale(0.96);
}

.context-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: none;
  color: var(--text-color);
  font-size: 14px;
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: background-color 0.15s ease;
}

.context-menu__item:hover {
  background: var(--surface-hover-color);
}

.context-menu__item--danger {
  color: #ef4444;
}

.context-menu__item--danger:hover {
  background: rgba(239, 68, 68, 0.1);
}

.context-menu__item__icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
</style>
