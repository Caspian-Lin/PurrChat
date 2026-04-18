<template>
  <section id="settings-panels" class="settings-section">
    <h2 class="settings-section__title">面板</h2>
    <p class="settings-section__desc">选择在侧边栏中显示的面板。</p>

    <div class="space-y-2">
      <label
        v-for="panel in allPanels"
        :key="panel.id"
        class="flex items-center justify-between p-3 rounded-[var(--radius-sm,8px)] cursor-pointer transition-colors duration-200"
        :class="panelSettings.visiblePanels.includes(panel.id) ? '' : 'opacity-50'"
        style="background: transparent"
        onmouseenter="this.style.background = 'var(--hover-background)'"
        onmouseleave="this.style.background = 'transparent'"
      >
        <div class="flex items-center gap-3">
          <component :is="panel.icon" :size="18" class="text-text-secondary" />
          <span class="text-sm text-text-primary">{{ panel.label }}</span>
        </div>
        <button
          class="w-10 h-6 rounded-full relative transition-colors duration-200 flex-shrink-0"
          :style="{
            backgroundColor: panelSettings.visiblePanels.includes(panel.id)
              ? 'var(--theme-primary)'
              : 'var(--border-color)',
          }"
          @click="togglePanel(panel.id)"
        >
          <span
            class="absolute top-1 w-4 h-4 rounded-full bg-white transition-transform duration-200"
            :style="{
              left: panelSettings.visiblePanels.includes(panel.id) ? '20px' : '4px',
            }"
          />
        </button>
      </label>
    </div>
  </section>
</template>

<script setup lang="ts">
import { BsChatLeftDots, BsFillPersonLinesFill, BsRobot } from 'vue-icons-plus/bs';
import type { PanelVisibilitySettings } from '../../../../models/types';

interface Props {
  panelSettings: PanelVisibilitySettings;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [settings: Partial<PanelVisibilitySettings>];
}>();

const allPanels = [
  { id: 'chat' as const, label: '聊天', icon: BsChatLeftDots },
  { id: 'friends' as const, label: '好友', icon: BsFillPersonLinesFill },
  { id: 'ai' as const, label: 'AI', icon: BsRobot },
];

function togglePanel(panelId: 'chat' | 'friends' | 'ai') {
  const current = [...props.panelSettings.visiblePanels];
  const index = current.indexOf(panelId);
  if (index >= 0) {
    current.splice(index, 1);
  } else {
    current.push(panelId);
  }
  emit('update', { visiblePanels: current });
}
</script>
