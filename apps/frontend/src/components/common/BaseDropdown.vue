<template>
  <div class="relative" ref="dropdownRef">
    <div @click="toggleDropdown">
      <slot name="trigger"></slot>
    </div>
    <Transition name="dropdown">
      <div
        v-if="isOpen"
        ref="menuRef"
        :class="['absolute w-48 rounded-lg shadow-lg z-50 py-1', positionClasses]"
        style="background: var(--surface-color); border: 1px solid var(--border-color)"
      >
        <slot></slot>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';

const isOpen = ref(false);
const dropdownRef = ref<HTMLElement>();
const menuRef = ref<HTMLElement>();

// 计算菜单位置类名
const positionClasses = computed(() => {
  if (!menuRef.value) return 'right-0 mt-2';

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

  // 垂直位置：优先选择不超出视口的位置
  // 如果底部超出视口且顶部不超出，则向上展开
  // 如果顶部超出视口且底部不超出，则向下展开
  // 如果上下都超出，选择超出较少的一侧
  const bottomOverflow = rect.bottom - viewportHeight;
  const topOverflow = -rect.top;

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
