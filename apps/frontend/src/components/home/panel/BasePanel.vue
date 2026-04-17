<template>
  <div class="flex h-full overflow-hidden">
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
/* 折叠时禁止侧栏内的交互 */
.base-panel__sidebar--collapsed {
  pointer-events: none;
}
</style>
