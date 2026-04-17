<template>
  <div ref="containerRef" :class="containerClasses" :style="containerStyle">
    <!-- 内容插槽 -->
    <div class="resizable-content" :style="contentStyle">
      <slot></slot>
    </div>

    <!-- 分割器 -->
    <div
      v-if="!disabled"
      ref="resizerRef"
      :class="resizerClasses"
      @mousedown="handleMouseDown"
      @touchstart="handleTouchStart"
    >
      <div class="resizer-handle"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';

interface Props {
  // 方向：'horizontal'（水平调整宽度）或 'vertical'（垂直调整高度）
  direction?: 'horizontal' | 'vertical';
  // 初始尺寸（像素）
  initialSize?: number;
  // 最小尺寸（像素）
  minSize?: number;
  // 最大尺寸（像素）
  maxSize?: number;
  // 分割器宽度/高度
  resizerSize?: number;
  // 是否禁用调整
  disabled?: boolean;
  // 是否作为分割器使用（显示在内容下方/右侧）
  asSplitter?: boolean;
  // 存储键（用于保存尺寸到localStorage）
  storageKey?: string;
}

const props = withDefaults(defineProps<Props>(), {
  direction: 'horizontal',
  initialSize: 300,
  minSize: 200,
  maxSize: 600,
  resizerSize: 1,
  disabled: false,
  asSplitter: false,
  storageKey: '',
});

const emit = defineEmits<{
  resize: [size: number];
  resizeStart: [];
  resizeEnd: [];
}>();

// 引用
// const containerRef = ref<HTMLElement | null>(null);
// const resizerRef = ref<HTMLElement | null>(null);

// 状态
const currentSize = ref(props.initialSize);
const isResizing = ref(false);
const startPos = ref(0);
const startSize = ref(0);

// 计算属性
const containerClasses = computed(() => [
  'resizable-container',
  `resizable-container--${props.direction}`,
  {
    'resizable-container--resizing': isResizing.value,
    'resizable-container--disabled': props.disabled,
  },
]);

const resizerClasses = computed(() => [
  'resizer',
  `resizer--${props.direction}`,
  {
    'resizer--active': isResizing.value,
  },
]);

const containerStyle = computed(() => {
  if (props.direction === 'horizontal') {
    return {
      width: `${currentSize.value}px`,
      height: '100%',
      flex: '0 0 auto',
    };
  } else {
    return {
      width: '100%',
      height: `${currentSize.value}px`,
      flex: '0 0 auto',
    };
  }
});

const contentStyle = computed(() => {
  return {
    width: '100%',
    height: '100%',
    overflow: 'hidden',
  };
});

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
  startSize.value = currentSize.value;

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
  updateSize(e.clientX, e.clientY);
};

// 处理触摸移动
const handleTouchMove = (e: TouchEvent) => {
  if (!isResizing.value) return;
  e.preventDefault();
  const touch = e.touches[0];
  if (touch) {
    updateSize(touch.clientX, touch.clientY);
  }
};

// 更新尺寸
const updateSize = (clientX: number, clientY: number) => {
  const currentPos = props.direction === 'horizontal' ? clientX : clientY;
  const delta = currentPos - startPos.value;

  let newSize = startSize.value + delta;

  // 限制最小和最大尺寸
  newSize = Math.max(props.minSize, Math.min(props.maxSize, newSize));

  currentSize.value = newSize;
  emit('resize', newSize);
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

  // 保存尺寸到localStorage
  if (props.storageKey) {
    localStorage.setItem(props.storageKey, currentSize.value.toString());
  }
};

// 生命周期
onMounted(() => {
  // 从localStorage恢复尺寸
  if (props.storageKey) {
    const savedSize = localStorage.getItem(props.storageKey);
    if (savedSize) {
      const size = parseInt(savedSize, 10);
      if (!isNaN(size) && size >= props.minSize && size <= props.maxSize) {
        currentSize.value = size;
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
.resizable-container {
  position: relative;
  display: flex;
  overflow: hidden;
}

.resizable-container--horizontal {
  flex-direction: row;
}

.resizable-container--vertical {
  flex-direction: column;
}

.resizable-content {
  overflow: hidden;
}

.resizer {
  position: relative;
  background-color: transparent;
  transition: background-color 0.2s ease;
  cursor: pointer;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
}

.resizer--horizontal {
  width: 1px;
  height: 100%;
  cursor: col-resize;
  background-color: var(--border-subtle-color, #e5e7eb);
}

.resizer--vertical {
  width: 100%;
  height: 1px;
  cursor: row-resize;
  background-color: var(--border-subtle-color, #e5e7eb);
}

.resizer:hover,
.resizer--active {
  background-color: var(--theme-primary, #5a8f4e);
}

.resizer-handle {
  position: absolute;
  background-color: var(--border-color, #e5e7eb);
  transition: background-color 0.2s cubic-bezier(0.25, 1, 0.5, 1);
}

.resizer--horizontal .resizer-handle {
  width: 1px;
  height: 20px;
  border-radius: 1px;
}

.resizer--vertical .resizer-handle {
  width: 20px;
  height: 1px;
  border-radius: 1px;
}

.resizer:hover .resizer-handle,
.resizer--active .resizer-handle {
  background-color: white;
}

.resizable-container--disabled {
  pointer-events: none;
}

.resizable-container--disabled .resizer {
  display: none;
}

/* 防止文本选择 */
.resizable-container--resizing * {
  user-select: none !important;
  pointer-events: none !important;
}

.resizer {
  pointer-events: auto !important;
}
</style>
