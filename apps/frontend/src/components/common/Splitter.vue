<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div :class="splitterClasses" @mousedown="handleMouseDown" @touchstart="handleTouchStart">
    <div class="splitter-handle"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';

interface Props {
  // 方向：'horizontal'（水平分割）或 'vertical'（垂直分割）
  direction?: 'horizontal' | 'vertical';
  // 初始位置（像素，相对于容器）
  initialPosition?: number;
  // 最小位置（像素）
  minPosition?: number;
  // 最大位置（像素）
  maxPosition?: number;
  // 分割器宽度/高度
  splitterSize?: number;
  // 是否禁用调整
  disabled?: boolean;
  // 存储键（用于保存位置到localStorage）
  storageKey?: string;
}

const props = withDefaults(defineProps<Props>(), {
  direction: 'horizontal',
  initialPosition: 300,
  minPosition: 100,
  maxPosition: 800,
  splitterSize: 1,
  disabled: false,
  storageKey: '',
});

const emit = defineEmits<{
  resize: [position: number];
  resizeStart: [];
  resizeEnd: [];
}>();

// 引用
// const splitterRef = ref<HTMLElement | null>(null);

// 状态
const currentPosition = ref(props.initialPosition);
const isResizing = ref(false);
const startPos = ref(0);
const startPosition = ref(0);

// 计算属性
const splitterClasses = computed(() => [
  'splitter',
  `splitter--${props.direction}`,
  {
    'splitter--active': isResizing.value,
    'splitter--disabled': props.disabled,
  },
]);

// 处理鼠标按下
const handleMouseDown = (e: MouseEvent) => {
  if (props.disabled) return;
  e.preventDefault();
  startResize(e.clientX, e.clientY);
};

// 处理触摸开始
const handleTouchStart = (e: TouchEvent) => {
  if (props.disabled) return;
  e.preventDefault();
  const touch = e.touches[0];
  if (touch) {
    startResize(touch.clientX, touch.clientY);
  }
};

// 开始调整大小
const startResize = (clientX: number, clientY: number) => {
  isResizing.value = true;
  startPos.value = props.direction === 'horizontal' ? clientX : clientY;
  startPosition.value = currentPosition.value;

  emit('resizeStart');

  // 添加事件监听器
  document.addEventListener('mousemove', handleMouseMove);
  document.addEventListener('mouseup', handleMouseUp);
  document.addEventListener('touchmove', handleTouchMove, { passive: false });
  document.addEventListener('touchend', handleTouchEnd);

  // 防止文本选择
  document.body.style.userSelect = 'none';
  document.body.style.cursor = props.direction === 'horizontal' ? 'col-resize' : 'row-resize';
};

// 处理鼠标移动
const handleMouseMove = (e: MouseEvent) => {
  if (!isResizing.value) return;
  e.preventDefault();
  updatePosition(e.clientX, e.clientY);
};

// 处理触摸移动
const handleTouchMove = (e: TouchEvent) => {
  if (!isResizing.value) return;
  e.preventDefault();
  const touch = e.touches[0];
  if (touch) {
    updatePosition(touch.clientX, touch.clientY);
  }
};

// 更新位置
const updatePosition = (clientX: number, clientY: number) => {
  const currentPos = props.direction === 'horizontal' ? clientX : clientY;
  const delta = currentPos - startPos.value;

  let newPosition = startPosition.value + delta;

  // 垂直方向时，拖动方向与高度变化相反
  if (props.direction === 'vertical') {
    newPosition = startPosition.value - delta;
  }

  // 限制最小和最大位置
  newPosition = Math.max(props.minPosition, Math.min(props.maxPosition, newPosition));

  currentPosition.value = newPosition;
  emit('resize', newPosition);
};

// 处理鼠标释放
const handleMouseUp = () => {
  endResize();
};

// 处理触摸结束
const handleTouchEnd = () => {
  endResize();
};

// 结束调整大小
const endResize = () => {
  if (!isResizing.value) return;

  isResizing.value = false;

  // 移除事件监听器
  document.removeEventListener('mousemove', handleMouseMove);
  document.removeEventListener('mouseup', handleMouseUp);
  document.removeEventListener('touchmove', handleTouchMove);
  document.removeEventListener('touchend', handleTouchEnd);

  // 恢复文本选择
  document.body.style.userSelect = '';
  document.body.style.cursor = '';

  emit('resizeEnd');

  // 保存位置到localStorage
  if (props.storageKey) {
    localStorage.setItem(props.storageKey, currentPosition.value.toString());
  }
};

// 生命周期
onMounted(() => {
  // 从localStorage恢复位置
  if (props.storageKey) {
    const savedPosition = localStorage.getItem(props.storageKey);
    if (savedPosition) {
      const position = parseInt(savedPosition, 10);
      if (!isNaN(position) && position >= props.minPosition && position <= props.maxPosition) {
        currentPosition.value = position;
        emit('resize', currentPosition.value);
      }
    }
  }
});

onUnmounted(() => {
  // 清理事件监听器
  document.removeEventListener('mousemove', handleMouseMove);
  document.removeEventListener('mouseup', handleMouseUp);
  document.removeEventListener('touchmove', handleTouchMove);
  document.removeEventListener('touchend', handleTouchEnd);
});
</script>

<style scoped>
.splitter {
  position: relative;
  background-color: transparent;
  transition: background-color 0.2s ease;
  cursor: pointer;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
}

.splitter--horizontal {
  width: 1px;
  height: 100%;
  cursor: col-resize;
  border-left: 1px solid var(--border-color, #e5e7eb);
  border-right: 1px solid var(--border-color, #e5e7eb);
}

.splitter--vertical {
  width: 100%;
  height: 1px;
  cursor: row-resize;
  border-top: 1px solid var(--border-color, #e5e7eb);
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.splitter:hover,
.splitter--active {
  background-color: var(--primary-color, #3b82f6);
}

.splitter-handle {
  position: absolute;
  background-color: var(--text-tertiary, #9ca3af);
  transition: background-color 0.2s ease;
}

.splitter--horizontal .splitter-handle {
  width: 1px;
  height: 20px;
  border-radius: 1px;
}

.splitter--vertical .splitter-handle {
  width: 20px;
  height: 1px;
  border-radius: 1px;
}

.splitter:hover .splitter-handle,
.splitter--active .splitter-handle {
  background-color: white;
}

.splitter--disabled {
  pointer-events: none;
  opacity: 0.5;
}

/* 防止文本选择 */
.splitter--active * {
  user-select: none !important;
}
</style>
