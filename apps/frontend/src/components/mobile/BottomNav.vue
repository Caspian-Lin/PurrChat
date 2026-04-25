<template>
  <nav
    class="mobile-bottom-nav"
    :style="{ paddingBottom: safeAreaBottom }"
  >
    <button
      v-for="tab in tabs"
      :key="tab.panel"
      :class="['mobile-nav-item', { active: activePanel === tab.panel }]"
      @click="handleTabClick(tab.panel)"
    >
      <component
        :is="tab.icon"
        :size="22"
        :class="['mobile-nav-icon', { 'text-white': activePanel === tab.panel, 'text-text-tertiary': activePanel !== tab.panel }]"
      />
      <span
        :class="['mobile-nav-label', { 'text-white': activePanel === tab.panel, 'text-text-tertiary': activePanel !== tab.panel }]"
      >
        {{ tab.label }}
      </span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  BsChatLeftDots,
  BsFillPersonLinesFill,
  BsRobot,
  BsCpu,
  BsGear,
} from 'vue-icons-plus/bs';
import { useRoute } from 'vue-router';
import { usePanelController } from '../../controllers/panelController';

type Panel = 'chat' | 'friends' | 'ai' | 'bots' | 'settings';

const tabs: { panel: Panel; label: string; icon: any }[] = [
  { panel: 'chat', label: '聊天', icon: BsChatLeftDots },
  { panel: 'friends', label: '好友', icon: BsFillPersonLinesFill },
  { panel: 'ai', label: 'AI', icon: BsRobot },
  { panel: 'bots', label: 'Bots', icon: BsCpu },
  { panel: 'settings', label: '设置', icon: BsGear },
];

const route = useRoute();
const { navigateToPanel } = usePanelController();

const activePanel = computed<Panel>(() => {
  const path = route.path;
  if (path.startsWith('/chat')) return 'chat';
  if (path.startsWith('/friends')) return 'friends';
  if (path.startsWith('/ai')) return 'ai';
  if (path.startsWith('/bots')) return 'bots';
  if (path.startsWith('/settings')) return 'settings';
  return 'chat';
});

const handleTabClick = (panel: Panel) => {
  if (panel === activePanel.value) return;
  navigateToPanel(panel);
};

// 安全区底部 padding（iPhone 等有底部安全区的设备）
const safeAreaBottom = computed(() => {
  if (typeof window === 'undefined') return '0px';
  return `env(safe-area-inset-bottom, 0px)`;
});
</script>

<style scoped>
.mobile-bottom-nav {
  display: flex;
  align-items: center;
  justify-content: space-around;
  height: 56px;
  background: var(--surface-color);
  border-top: 1px solid var(--border-subtle);
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 100;
  padding-bottom: 0px;
}

.mobile-nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  flex: 1;
  height: 100%;
  background: none;
  border: none;
  cursor: pointer;
  min-height: 44px;
  -webkit-tap-highlight-color: transparent;
  transition: transform 0.15s ease;
}

.mobile-nav-item:active {
  transform: scale(0.92);
}

.mobile-nav-item.active {
  position: relative;
}

.mobile-nav-item.active::before {
  content: '';
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 32px;
  height: 2px;
  border-radius: 1px;
  background: var(--theme-primary);
}

.mobile-nav-icon {
  transition: color 0.2s ease;
}

.mobile-nav-label {
  font-size: 10px;
  font-weight: 500;
  line-height: 1;
  letter-spacing: 0.02em;
  transition: color 0.2s ease;
}
</style>
