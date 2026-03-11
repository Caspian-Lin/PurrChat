<template>
  <div ref="wrapperRef" class="custom-scrollbar-wrapper">
    <!-- 内容容器 -->
    <div
      ref="containerRef"
      class="custom-scrollbar-container"
      @wheel="handleWheel"
      @scroll="handleScroll"
    >
      <slot></slot>
    </div>
    <!-- 滚动条容器（固定在右侧） -->
    <div class="custom-scrollbar-sidebar">
      <!-- 自定义垂直滚动条 -->
      <div
        ref="scrollbarThumbRef"
        class="custom-scrollbar-thumb"
        :class="{ 'custom-scrollbar-thumb-hover': isHovering || isDragging }"
        :style="{
          height: thumbHeight + 'px',
          top: thumbTop + 'px',
          display: showScrollbar ? 'block' : 'none',
        }"
        @mousedown="startDrag"
        @mouseenter="((isTrackHovering = true), (isHovering = true))"
        @mouseleave="((isTrackHovering = false), (isHovering = false))"
      ></div>
      <!-- 滚动条轨道（用于点击滚动） -->
      <div
        ref="scrollbarTrackRef"
        class="custom-scrollbar-track"
        :class="{ 'custom-scrollbar-track-hover': isTrackHovering || isHovering || isDragging }"
        :style="{ display: showScrollbar ? 'block' : 'none' }"
        @mousedown="handleTrackClick"
        @mouseenter="((isTrackHovering = true), (isHovering = true))"
        @mouseleave="((isTrackHovering = false), (isHovering = false))"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue';

interface Props {
  thumbWidth?: number;
  thumbMinHeight?: number;
}

const props = withDefaults(defineProps<Props>(), {
  thumbWidth: 8,
  thumbMinHeight: 40,
});

const wrapperRef = ref<HTMLElement | null>(null);
const containerRef = ref<HTMLElement | null>(null);
// const scrollbarThumbRef = ref<HTMLElement | null>(null);
const scrollbarTrackRef = ref<HTMLElement | null>(null);

// 状态
const thumbHeight = ref(0);
const thumbTop = ref(0);
const isDragging = ref(false);
const isHovering = ref(false);
const isTrackHovering = ref(false);
const dragStartY = ref(0);
const dragStartThumbTop = ref(0);
const scrollStartTop = ref(0);
const showScrollbar = ref(false);

// 计算滚动条高度和位置
const updateScrollbar = () => {
  if (!containerRef.value || !wrapperRef.value) return;

  const container = containerRef.value;
  const wrapper = wrapperRef.value;
  const scrollHeight = container.scrollHeight;
  const clientHeight = wrapper.clientHeight; // 使用wrapper的高度而不是container的高度
  const scrollTop = container.scrollTop;

  // 判断是否需要显示滚动条
  showScrollbar.value = scrollHeight > clientHeight;

  // 如果不需要滚动，直接返回
  if (!showScrollbar.value) {
    thumbHeight.value = 0;
    thumbTop.value = 0;
    return;
  }

  // 计算滚动条高度
  const thumbHeightRatio = clientHeight / scrollHeight;
  let newThumbHeight = Math.max(clientHeight * thumbHeightRatio, props.thumbMinHeight);

  // 计算滚动条位置
  const maxThumbTop = clientHeight - newThumbHeight;
  const scrollRatio = scrollTop / (scrollHeight - clientHeight);
  const newThumbTop = scrollRatio * maxThumbTop;

  thumbHeight.value = newThumbHeight;
  thumbTop.value = newThumbTop;
};

// 处理滚轮事件
const handleWheel = (e: WheelEvent) => {
  if (!containerRef.value) return;
  e.preventDefault();
  containerRef.value.scrollTop += e.deltaY;
};

// 处理滚动事件
const handleScroll = () => {
  updateScrollbar();
};

// 开始拖拽
const startDrag = (e: MouseEvent) => {
  e.preventDefault();
  isDragging.value = true;
  dragStartY.value = e.clientY;
  dragStartThumbTop.value = thumbTop.value;
  scrollStartTop.value = containerRef.value?.scrollTop || 0;

  document.addEventListener('mousemove', handleDrag);
  document.addEventListener('mouseup', stopDrag);
};

// 处理拖拽
const handleDrag = (e: MouseEvent) => {
  if (!containerRef.value || !wrapperRef.value) return;

  const deltaY = e.clientY - dragStartY.value;
  const clientHeight = wrapperRef.value.clientHeight; // 使用wrapper的高度
  const scrollHeight = containerRef.value.scrollHeight;
  const maxThumbTop = clientHeight - thumbHeight.value;

  // 计算新的滚动条位置（直接跟随鼠标移动）
  let newThumbTop = dragStartThumbTop.value + deltaY;
  newThumbTop = Math.max(0, Math.min(newThumbTop, maxThumbTop));

  // 更新滚动条位置
  thumbTop.value = newThumbTop;

  // 计算对应的滚动位置
  const scrollRatio = newThumbTop / maxThumbTop;
  const maxScrollTop = scrollHeight - clientHeight;
  const newScrollTop = scrollRatio * maxScrollTop;

  containerRef.value.scrollTop = newScrollTop;
};

// 停止拖拽
const stopDrag = () => {
  isDragging.value = false;
  document.removeEventListener('mousemove', handleDrag);
  document.removeEventListener('mouseup', stopDrag);
};

// 处理轨道点击
const handleTrackClick = (e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();

  if (!containerRef.value || !wrapperRef.value) return;

  const trackRect = scrollbarTrackRef.value?.getBoundingClientRect();
  if (!trackRect) return;

  const clickY = e.clientY - trackRect.top;
  const clientHeight = wrapperRef.value.clientHeight; // 使用wrapper的高度
  const scrollHeight = containerRef.value.scrollHeight;

  // 计算点击位置对应的滚动位置
  const scrollRatio = clickY / clientHeight;
  const maxScrollTop = scrollHeight - clientHeight;
  const newScrollTop = scrollRatio * maxScrollTop;

  containerRef.value.scrollTop = newScrollTop;
};

// 监听内容变化
const observeContent = () => {
  if (!containerRef.value) return;

  const observer = new MutationObserver(() => {
    nextTick(() => {
      updateScrollbar();
    });
  });

  observer.observe(containerRef.value, {
    childList: true,
    subtree: true,
    attributes: true,
    characterData: true,
  });

  return observer;
};

// 监听窗口大小变化
const handleResize = () => {
  updateScrollbar();
};

// 生命周期
let contentObserver: MutationObserver | null = null;

onMounted(() => {
  nextTick(() => {
    updateScrollbar();
    const observer = observeContent();
    if (observer) {
      contentObserver = observer;
    }
    window.addEventListener('resize', handleResize);
  });
});

onUnmounted(() => {
  if (contentObserver) {
    contentObserver.disconnect();
  }
  window.removeEventListener('resize', handleResize);
  document.removeEventListener('mousemove', handleDrag);
  document.removeEventListener('mouseup', stopDrag);
});

// 暴露滚动方法
defineExpose({
  scrollToTop: () => {
    if (containerRef.value) {
      containerRef.value.scrollTop = 0;
    }
  },
  scrollToBottom: () => {
    if (containerRef.value) {
      containerRef.value.scrollTop = containerRef.value.scrollHeight;
    }
  },
  scrollTo: (top: number) => {
    if (containerRef.value) {
      containerRef.value.scrollTop = top;
    }
  },
  updateScrollbar: () => {
    updateScrollbar();
  },
});
</script>

<style scoped>
.custom-scrollbar-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
}

.custom-scrollbar-container {
  position: relative;
  overflow: hidden; /* 隐藏原生滚动条 */
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 隐藏原生滚动条 */
.custom-scrollbar-container::-webkit-scrollbar {
  width: 0;
  height: 0;
}

.custom-scrollbar-container {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

/* 滚动条侧边栏容器（悬浮在内容上方） */
.custom-scrollbar-sidebar {
  position: absolute;
  top: 0;
  right: 0;
  width: 8px;
  height: 100%;
  z-index: 9999;
  pointer-events: none; /* 让鼠标事件穿透到下面的轨道和滑块 */
}

/* 自定义滚动条轨道 */
.custom-scrollbar-track {
  position: absolute;
  top: 0;
  right: 0;
  width: 100%;
  background: transparent;
  z-index: 9999;
  pointer-events: auto; /* 轨道接收鼠标事件 */
  transition: background 0.2s ease;
  height: 100%; /* 确保轨道高度与父组件一致 */
}

/* 轨道hover效果 */
.custom-scrollbar-track:hover,
.custom-scrollbar-track-hover {
  background: rgba(145, 145, 145, 0.4);
  width: 10px;
}

/* 自定义滚动条滑块 */
.custom-scrollbar-thumb {
  position: absolute;
  right: 1px;
  width: 6px; /* 默认细滚动条 */
  background: rgba(106, 106, 106, 0.4);
  border-radius: 4px;
  cursor: pointer;
  z-index: 10000;
  opacity: 1; /* 一直可见 */
  transition:
    width 0.2s ease,
    background 0.2s ease;
  pointer-events: auto; /* 滑块接收鼠标事件 */
}

.custom-scrollbar-thumb-hover {
  width: 8px; /* hover时变粗 */
  right: 1px;
}

.custom-scrollbar-thumb:hover {
  background: rgba(64, 64, 64, 0.6);
}

.custom-scrollbar-thumb:active {
  background: rgba(64, 64, 64, 0.6);
}

/* 深色主题 */
[data-theme='dark'] .custom-scrollbar-thumb {
  background: rgba(115, 115, 115, 0.5);
}

[data-theme='dark'] .custom-scrollbar-thumb:hover {
  background: rgba(202, 202, 202, 0.8);
}

[data-theme='dark'] .custom-scrollbar-thumb:active {
  background: rgb(211, 211, 211);
}

/* 深色主题轨道hover效果 */
[data-theme='dark'] .custom-scrollbar-track:hover,
[data-theme='dark'] .custom-scrollbar-track-hover {
  background: rgba(115, 115, 115, 0.2);
}
</style>
