<template>
  <div
    class="flex flex-col items-center justify-between px-4 py-5 gap-4 bg-bg-primary border-r border-border-subtle relative overflow-hidden"
  >
    <!-- Logo -->
    <div class="flex-0 flex flex-col items-center">
      <div class="text-xl font-bold tracking-tight leading-tight text-center">
        <span class="font-display" style="color: var(--theme-primary)">Purr</span>
        <br />
        <span class="text-text-tertiary font-body">Chat</span>
      </div>
    </div>

    <!-- 会话和好友按钮 -->
    <div class="flex flex-col items-center gap-2">
      <button
        :class="[
          'w-12 h-12 p-0 rounded-xl flex items-center justify-center transition-all',
          activePanel === 'chat'
            ? 'bg-[var(--theme-primary)]'
            : 'bg-bg-quaternary hover:bg-hover-bg',
        ]"
        @click="handlePanelClick('chat')"
      >
        <BsChatLeftDots
          :class="['', activePanel === 'chat' ? 'text-white' : 'text-text-tertiary']"
          :size="24"
        />
      </button>
      <button
        :class="[
          'w-12 h-12 p-0 rounded-xl flex items-center justify-center transition-all',
          activePanel === 'friends'
            ? 'bg-[var(--theme-primary)]'
            : 'bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary',
        ]"
        @click="handlePanelClick('friends')"
      >
        <BsFillPersonLinesFill
          :class="['', activePanel === 'friends' ? 'text-white' : 'text-text-tertiary']"
          :size="24"
        />
      </button>
      <button
        :class="[
          'w-12 h-12 p-0 rounded-xl flex items-center justify-center transition-all',
          activePanel === 'ai'
            ? 'bg-[var(--theme-primary)]'
            : 'bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary',
        ]"
        @click="handlePanelClick('ai')"
      >
        <BsRobot
          :class="['', activePanel === 'ai' ? 'text-white' : 'text-text-tertiary']"
          :size="24"
        />
      </button>
      <button
        :class="[
          'w-12 h-12 p-0 rounded-xl flex items-center justify-center transition-all',
          activePanel === 'bots'
            ? 'bg-[var(--theme-primary)]'
            : 'bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary',
        ]"
        @click="handlePanelClick('bots')"
      >
        <BsCpu
          :class="['', activePanel === 'bots' ? 'text-white' : 'text-text-tertiary']"
          :size="24"
        />
      </button>
      <button
        :class="[
          'w-12 h-12 p-0 rounded-xl flex items-center justify-center transition-all',
          activePanel === 'settings'
            ? 'bg-[var(--theme-primary)]'
            : 'bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary',
        ]"
        @click="handlePanelClick('settings')"
      >
        <BsGear
          :class="['', activePanel === 'settings' ? 'text-white' : 'text-text-tertiary']"
          :size="24"
        />
      </button>
    </div>

    <!-- 底部区域 - 主题切换、个人资料 -->
    <div class="w-12 flex flex-col items-center gap-4 mt-auto">
      <!-- 在线状态指示器（只在离线或连接中时显示） -->
      <div
        v-if="!isOnline"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-all"
        :style="{
          background: isOffline ? 'var(--color-error-bg)' : 'var(--color-warning-bg)',
          borderColor: isOffline ? 'var(--color-error)' : 'var(--color-warning)',
        }"
        :title="connectionStatusText"
      >
        <div
          class="w-2 h-2 rounded-full animate-pulse"
          :style="{
            background: isOffline ? 'var(--color-error)' : 'var(--color-warning)',
          }"
        ></div>
        <!-- <span :class="['text-xs font-medium', isOffline ? 'text-red-400' : 'text-yellow-400']">
          {{ connectionStatusText }}
        </span> -->
      </div>
      <ThemeSwitcher />
      <div
        class="flex items-center gap-2 cursor-pointer rounded-lg hover:bg-hover-bg transition-colors"
        @click="$emit('show-profile')"
      >
        <img
          v-if="currentUser?.avatar_url"
          :src="currentUser.avatar_url"
          alt="avatar"
          class="w-10 h-10 rounded-xl object-cover"
          referrerpolicy="no-referrer"
          @error="(e) => console.error('[avatar] 侧边栏头像加载失败:', currentUser?.avatar_url, e)"
        />
        <div
          v-else
          class="w-10 h-10 rounded-xl flex items-center justify-center font-bold text-white"
          style="background: var(--theme-gradient)"
        >
          {{ currentUser?.username?.charAt(0) || 'U' }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { BsChatLeftDots, BsFillPersonLinesFill, BsRobot, BsCpu, BsGear } from 'vue-icons-plus/bs';
import ThemeSwitcher from '../ThemeSwitcher.vue';
import { usePanelController } from '../../controllers/panelController';
import { useRoute } from 'vue-router';
import type { User } from '../../models/types';
import { useConnectionStore } from '../../stores/connection';
import { useSidebarStore } from '../../stores/sidebar';

interface Props {
  currentUser: User | null;
}

defineProps<Props>();

const route = useRoute();
const { navigateToPanel } = usePanelController();
const connectionStore = useConnectionStore();
const sidebarStore = useSidebarStore();

// 根据当前路由确定activePanel
const activePanel = computed(() => {
  if (route.path === '/chat') return 'chat';
  if (route.path === '/friends') return 'friends';
  if (route.path === '/ai') return 'ai';
  if (route.path === '/bots' || route.path.startsWith('/bot-studio/')) return 'bots';
  if (route.path === '/settings') return 'settings';
  return 'chat';
});

// 连接状态
const isOnline = computed(() => connectionStore.isOnline);
const isOffline = computed(() => connectionStore.isOffline);
// const isConnecting = computed(() => connectionStore.isConnecting);
const connectionStatusText = computed(() => connectionStore.getConnectionStatusText());

const handlePanelClick = (panel: 'chat' | 'friends' | 'ai' | 'bots' | 'settings') => {
  if (panel === activePanel.value) {
    // 点击当前 panel → 切换折叠/展开
    sidebarStore.toggleSidebar(panel);
  } else {
    // 切换到其他 panel
    navigateToPanel(panel);
  }
};

defineEmits<{
  'show-profile': [];
}>();
</script>

<style scoped></style>
