<template>
  <div class="relative" ref="dropdownRef">
    <div @click="toggleDropdown">
      <slot name="trigger"></slot>
    </div>
    <Transition name="dropdown">
      <div
        v-if="isOpen"
        ref="menuRef"
        :class="['absolute rounded-lg shadow-lg z-50 py-1 overflow-auto', positionClasses]"
        :style="dropdownStyle"
      >
        <slot></slot>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';

interface Props {
  position?: 'top' | 'bottom' | 'auto';
  align?: 'left' | 'right' | 'center';
  width?: string;
  maxHeight?: string;
  offsetTop?: string;
  offsetBottom?: string;
  offsetLeft?: string;
  offsetRight?: string;
}

const props = withDefaults(defineProps<Props>(), {
  position: 'auto',
  align: 'right',
  width: undefined,
  maxHeight: undefined,
  offsetTop: undefined,
  offsetBottom: undefined,
  offsetLeft: undefined,
  offsetRight: undefined,
});

const isOpen = ref(false);
const dropdownRef = ref<HTMLElement>();
const menuRef = ref<HTMLElement>();

// 检查是否使用了手动 offset
const useManualOffset = computed(() => {
  return (
    props.offsetTop !== undefined ||
    props.offsetBottom !== undefined ||
    props.offsetLeft !== undefined ||
    props.offsetRight !== undefined
  );
});

// 计算 dropdown 样式
const dropdownStyle = computed(() => {
  const style: Record<string, string> = {
    background: 'var(--surface-color)',
    border: '1px solid var(--border-color)',
  };

  // 如果使用了手动 offset，则应用 offset 并忽略 width/maxHeight
  if (useManualOffset.value) {
    if (props.offsetTop !== undefined) {
      style.top = props.offsetTop;
    }
    if (props.offsetBottom !== undefined) {
      style.bottom = props.offsetBottom;
    }
    if (props.offsetLeft !== undefined) {
      style.left = props.offsetLeft;
    }
    if (props.offsetRight !== undefined) {
      style.right = props.offsetRight;
    }
  } else {
    // 否则应用 width/maxHeight
    if (props.width !== undefined) {
      style.width = props.width;
    }
    if (props.maxHeight !== undefined) {
      style.maxHeight = props.maxHeight;
    }
  }

  return style;
});

// 计算菜单位置类名
const positionClasses = computed(() => {
  // 如果使用了手动 offset，不计算位置类
  if (useManualOffset.value) {
    return '';
  }

  if (!menuRef.value) {
    // 如果没有 menuRef，返回默认类
    if (props.position === 'top') return 'right-0 bottom-full mb-2';
    if (props.position === 'bottom') return 'right-0 mt-2';
    return 'right-0 mt-2';
  }

  const rect = menuRef.value.getBoundingClientRect();
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;

  let classes = '';

  // 水平位置：优先选择不超出视口的位置
  // 如果右侧超出视口且左侧不超出，则左对齐
  // 如果左侧超出视口且右侧不超出，则右对齐
  // 如果两侧都超出，选择超出较少的一侧
  const rightOverflow = rect.right - viewportWidth;
  const leftOverflow = -rect.left;

  if (props.align === 'left') {
    classes += 'left-0 ';
  } else if (props.align === 'center') {
    classes += 'left-1/2 -translate-x-1/2 ';
  } else if (props.align === 'right') {
    if (rightOverflow > 0 && leftOverflow <= 0) {
      // 右侧超出，左侧不超出，使用左对齐
      classes += 'left-0 ';
    } else if (leftOverflow > 0 && rightOverflow <= 0) {
      // 左侧超出，右侧不超出，使用右对齐
      classes += 'right-0 ';
    } else if (rightOverflow > 0 && leftOverflow > 0) {
      // 两侧都超出，选择超出较少的一侧
      if (rightOverflow < leftOverflow) {
        classes += 'right-0 ';
      } else {
        classes += 'left-0 ';
      }
    } else {
      // 默认右对齐
      classes += 'right-0 ';
    }
  }

  // 垂直位置：优先选择不超出视口的位置
  // 如果底部超出视口且顶部不超出，则向上展开
  // 如果顶部超出视口且底部不超出，则向下展开
  // 如果上下都超出，选择超出较少的一侧
  const bottomOverflow = rect.bottom - viewportHeight;
  const topOverflow = -rect.top;

  if (props.position === 'top') {
    classes += 'bottom-full mb-2';
  } else if (props.position === 'bottom') {
    classes += 'mt-2';
  } else if (props.position === 'auto') {
    if (bottomOverflow > 0 && topOverflow <= 0) {
      // 底部超出，顶部不超出，向上展开
      classes += 'bottom-full mb-2';
    } else if (topOverflow > 0 && bottomOverflow <= 0) {
      // 顶部超出，底部不超出，向下展开
      classes += 'mt-2';
    } else if (bottomOverflow > 0 && topOverflow > 0) {
      // 上下都超出，选择超出较少的一侧
      if (bottomOverflow < topOverflow) {
        classes += 'bottom-full mb-2';
      } else {
        classes += 'mt-2';
      }
    } else {
      // 默认向下展开
      classes += 'mt-2';
    }
  }

  return classes;
});

const toggleDropdown = () => {
  isOpen.value = !isOpen.value;
  if (isOpen.value) {
    nextTick(() => {
      updatePosition();
    });
  }
};

const updatePosition = () => {
  // 强制重新计算位置
  if (menuRef.value) {
    // 触发 computed 重新计算
    void positionClasses.value;
  }
};

const closeDropdown = (event: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false;
  }
};

// 监听窗口大小变化，更新菜单位置
const handleResize = () => {
  if (isOpen.value) {
    updatePosition();
  }
};

onMounted(() => {
  document.addEventListener('click', closeDropdown);
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  document.removeEventListener('click', closeDropdown);
  window.removeEventListener('resize', handleResize);
});

// 监听 isOpen 变化
watch(isOpen, (newVal) => {
  if (newVal) {
    nextTick(() => {
      updatePosition();
    });
  }
});

defineExpose({
  close: () => {
    isOpen.value = false;
  },
});
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
