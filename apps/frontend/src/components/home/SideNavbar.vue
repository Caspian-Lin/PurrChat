<template>
  <div
    class="flex flex-col items-center justify-between px-4 py-5 gap-4 bg-bg-primary border-r border-border-color relative overflow-hidden"
  >
    <!-- 标题 -->
    <div class="flex-0 text-xl font-bold text-text-primary">Purr <br />Chat</div>

    <!-- 会话和好友按钮 -->
    <div class="flex flex-col items-center gap-2">
      <button
        :class="[
          'w-12 h-12 p-0 roundrect flex items-center justify-center transition-all',
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
          'w-12 h-12 p-0 roundrect flex items-center justify-center transition-all',
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
    </div>

    <!-- 底部区域 - 主题切换、个人资料 -->
    <div class="w-12 flex flex-col items-center gap-4 mt-auto">
      <!-- 在线状态指示器（只在离线或连接中时显示） -->
      <div
        v-if="!isOnline"
        :class="[
          'flex items-center gap-2 px-3 py-1.5 rounded-lg transition-all',
          isOffline
            ? 'bg-red-500/20 border border-red-500/50'
            : 'bg-yellow-500/20 border border-yellow-500/50',
        ]"
        :title="connectionStatusText"
      >
        <div
          :class="[
            'w-2 h-2 rounded-full animate-pulse',
            isOffline ? 'bg-red-500' : 'bg-yellow-500',
          ]"
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
          class="w-10 h-10 roundrect object-cover"
        />
        <div
          v-else
          class="w-10 h-10 roundrect flex items-center justify-center font-bold text-white"
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
import { BsChatLeftDots, BsFillPersonLinesFill } from 'vue-icons-plus/bs';
import ThemeSwitcher from '../ThemeSwitcher.vue';
import { usePanelController } from '../../controllers/panelController';
import { useRoute } from 'vue-router';
import type { User } from '../../models/types';
import { useConnectionStore } from '../../stores/connection';

interface Props {
  currentUser: User | null;
}

defineProps<Props>();

const route = useRoute();
const { navigateToPanel } = usePanelController();
const connectionStore = useConnectionStore();

// 根据当前路由确定activePanel
const activePanel = computed(() => {
  if (route.path === '/chat') return 'chat';
  if (route.path === '/friends') return 'friends';
  return 'chat'; // 默认
});

// 连接状态
const isOnline = computed(() => connectionStore.isOnline);
const isOffline = computed(() => connectionStore.isOffline);
// const isConnecting = computed(() => connectionStore.isConnecting);
const connectionStatusText = computed(() => connectionStore.getConnectionStatusText());

const handlePanelClick = (panel: 'chat' | 'friends') => {
  navigateToPanel(panel);
};

defineEmits<{
  'show-profile': [];
}>();
</script>

<style scoped></style>
