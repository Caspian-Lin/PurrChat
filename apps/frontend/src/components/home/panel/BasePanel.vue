<template>
  <!-- 移动端：直接全屏显示内容（忽略侧栏） -->
  <div v-if="isMobile" class="mobile-base-panel">
    <slot />
  </div>

  <!-- 桌面端：侧栏 + 主视图 -->
  <div v-else class="flex h-full overflow-hidden">
    <!-- 侧栏 -->
    <div
      class="base-panel__sidebar flex-shrink-0 overflow-hidden"
      :class="{ 'base-panel__sidebar--collapsed': sidebarStore.isCollapsed(panelId) }"
      :style="sidebarStyle"
    >
      <ResizableContainer
        direction="horizontal"
        :initial-size="initialSidebarWidth"
        :min-size="minSidebarWidth"
        :max-size="maxSidebarWidth"
        :storage-key="`purr-chat-sidebar-${panelId}`"
        class="h-full bg-bg-primary border-r border-border-subtle"
      >
        <slot name="sidebar" />
      </ResizableContainer>
    </div>

    <!-- 主视图 -->
    <div class="flex-1 flex flex-col bg-bg-tertiary min-w-0">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { PanelId } from '../../../stores/sidebar';
import { useSidebarStore } from '../../../stores/sidebar';
import { usePlatform } from '../../../composables/usePlatform';
import ResizableContainer from '../../common/ResizableContainer.vue';

interface Props {
  panelId: PanelId;
  initialSidebarWidth?: number;
  minSidebarWidth?: number;
  maxSidebarWidth?: number;
}

const props = withDefaults(defineProps<Props>(), {
  initialSidebarWidth: 320,
  minSidebarWidth: 250,
  maxSidebarWidth: 500,
});

const { isMobile } = usePlatform();
const sidebarStore = useSidebarStore();

const sidebarStyle = computed(() => {
  const collapsed = sidebarStore.isCollapsed(props.panelId);
  return {
    maxWidth: collapsed ? '0px' : `${props.maxSidebarWidth}px`,
    opacity: collapsed ? 0 : 1,
    overflow: 'hidden' as const,
    transition: [
      `max-width var(--duration-normal, 300ms) var(--ease-out-quart, cubic-bezier(0.25, 1, 0.5, 1))`,
      `opacity var(--duration-fast, 200ms) var(--ease-out-quart, cubic-bezier(0.25, 1, 0.5, 1))`,
    ].join(', '),
  };
});
</script>

<style scoped>
.base-panel__sidebar--collapsed {
  pointer-events: none;
}

.mobile-base-panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--background-color);
}
</style>
