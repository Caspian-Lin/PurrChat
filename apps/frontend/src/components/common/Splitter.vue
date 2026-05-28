<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div ref="el" :class="splitterClasses" @pointerdown="onPointerDown">
    <div class="splitter-indicator" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';

interface Props {
  direction?: 'horizontal' | 'vertical';
  initialPosition?: number;
  minPosition?: number;
  maxPosition?: number;
  disabled?: boolean;
  storageKey?: string;
}

const props = withDefaults(defineProps<Props>(), {
  direction: 'horizontal',
  initialPosition: 300,
  minPosition: 100,
  maxPosition: 800,
  disabled: false,
  storageKey: '',
});

const emit = defineEmits<{
  resize: [position: number];
  resizeStart: [];
  resizeEnd: [];
}>();

const el = ref<HTMLElement | null>(null);
const isActive = ref(false);

const splitterClasses = computed(() => [
  'splitter',
  `splitter--${props.direction}`,
  { 'splitter--active': isActive.value, 'splitter--disabled': props.disabled },
]);

// 拖拽状态（不用 ref，避免不必要的响应式开销）
let startPos = 0;
let startPosition = 0;

function getSavedPosition(): number {
  if (props.storageKey) {
    const saved = localStorage.getItem(props.storageKey);
    if (saved) {
      const pos = parseInt(saved, 10);
      if (!isNaN(pos) && pos >= props.minPosition && pos <= props.maxPosition) {
        return pos;
      }
    }
  }
  return props.initialPosition;
}

function onPointerDown(e: PointerEvent) {
  if (props.disabled) return;
  e.preventDefault();

  const target = el.value;
  if (!target) return;

  target.setPointerCapture(e.pointerId);
  isActive.value = true;

  startPos = props.direction === 'horizontal' ? e.clientX : e.clientY;
  startPosition = getSavedPosition();

  document.body.style.userSelect = 'none';
  document.body.style.cursor = props.direction === 'horizontal' ? 'col-resize' : 'row-resize';

  emit('resizeStart');
}

function onPointerMove(e: PointerEvent) {
  if (!isActive.value) return;

  const currentPos = props.direction === 'horizontal' ? e.clientX : e.clientY;
  const delta = currentPos - startPos;

  // 垂直方向时，拖动方向与高度变化相反（往上拖 → 输入区变大）
  const newPosition =
    props.direction === 'vertical' ? startPosition - delta : startPosition + delta;

  const clamped = Math.max(props.minPosition, Math.min(props.maxPosition, newPosition));
  emit('resize', clamped);

  if (props.storageKey) {
    localStorage.setItem(props.storageKey, clamped.toString());
  }
}

function onPointerUp(e: PointerEvent) {
  if (!isActive.value) return;

  const target = el.value;
  if (target) {
    try {
      target.releasePointerCapture(e.pointerId);
    } catch {
      // pointer 可能已自动释放
    }
  }

  isActive.value = false;
  document.body.style.userSelect = '';
  document.body.style.cursor = '';

  emit('resizeEnd');
}

onMounted(() => {
  const target = el.value;
  if (!target) return;

  target.addEventListener('pointermove', onPointerMove);
  target.addEventListener('pointerup', onPointerUp);
  target.addEventListener('pointercancel', onPointerUp);

  // 恢复保存的位置并通知父组件
  const saved = getSavedPosition();
  if (saved !== props.initialPosition) {
    emit('resize', saved);
  }
});

onUnmounted(() => {
  const target = el.value;
  if (!target) return;

  target.removeEventListener('pointermove', onPointerMove);
  target.removeEventListener('pointerup', onPointerUp);
  target.removeEventListener('pointercancel', onPointerUp);
});
</script>

<style scoped>
.splitter {
  position: relative;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: transparent;
}

/* —— 水平分割（左右拖拽） —— */
.splitter--horizontal {
  width: 1px;
  height: 100%;
  cursor: col-resize;
}

/* —— 垂直分割（上下拖拽） —— */
.splitter--vertical {
  width: 100%;
  height: 1px;
  cursor: row-resize;
}

/* 可视化指示线 */
.splitter-indicator {
  background-color: var(--border-subtle-color, #e5e7eb);
  border-radius: 9999px;
  transition:
    background-color 0.2s cubic-bezier(0.25, 1, 0.5, 1),
    transform 0.15s cubic-bezier(0.25, 1, 0.5, 1);
}

.splitter--horizontal .splitter-indicator {
  width: 1px;
  height: 24px;
}

.splitter--vertical .splitter-indicator {
  width: 24px;
  height: 1px;
}

/* 悬停 / 拖拽中 */
.splitter:hover .splitter-indicator,
.splitter--active .splitter-indicator {
  background-color: var(--theme-primary);
}

.splitter--vertical.splitter--active .splitter-indicator {
  transform: scaleY(2);
}

.splitter--horizontal.splitter--active .splitter-indicator {
  transform: scaleX(2);
}

.splitter--disabled {
  pointer-events: none;
  opacity: 0.5;
}
</style>
