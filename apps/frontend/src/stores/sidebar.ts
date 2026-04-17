import { defineStore } from 'pinia';
import { ref } from 'vue';

export type PanelId = 'chat' | 'friends' | 'ai';

export const useSidebarStore = defineStore('sidebar', () => {
  const collapsed = ref<Record<PanelId, boolean>>({
    chat: false,
    friends: false,
    ai: false,
  });

  function toggleSidebar(panelId: PanelId) {
    collapsed.value[panelId] = !collapsed.value[panelId];
  }

  function isCollapsed(panelId: PanelId): boolean {
    return collapsed.value[panelId];
  }

  return { collapsed, toggleSidebar, isCollapsed };
});
