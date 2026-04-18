<template>
  <section id="settings-notifications" class="settings-section">
    <h2 class="settings-section__title">通知</h2>
    <p class="settings-section__desc">选择你需要接收通知的消息类型。</p>

    <div class="space-y-2">
      <div
        v-for="item in notificationItems"
        :key="item.key"
        class="flex items-center justify-between p-3 rounded-[var(--radius-sm,8px)] cursor-pointer transition-colors duration-200"
        style="background: transparent"
        onmouseenter="this.style.background = 'var(--hover-background)'"
        onmouseleave="this.style.background = 'transparent'"
        @click="toggleNotification(item.key)"
      >
        <div>
          <p class="text-sm text-text-primary">{{ item.label }}</p>
          <p class="text-xs text-text-tertiary mt-0.5">{{ item.desc }}</p>
        </div>
        <button
          class="w-10 h-6 rounded-full relative transition-colors duration-200 flex-shrink-0"
          :style="{
            backgroundColor: (notificationSettings as any)[item.key]
              ? 'var(--theme-primary)'
              : 'var(--border-color)',
          }"
        >
          <span
            class="absolute top-1 w-4 h-4 rounded-full bg-white transition-transform duration-200"
            :style="{
              left: (notificationSettings as any)[item.key] ? '20px' : '4px',
            }"
          />
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { NotificationSettings } from '../../../../models/types';

interface Props {
  notificationSettings: NotificationSettings;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [settings: Partial<NotificationSettings>];
}>();

const notificationItems = [
  { key: 'messageNotification' as const, label: '新消息', desc: '收到新聊天消息时通知' },
  { key: 'friendRequestNotification' as const, label: '好友请求', desc: '收到好友请求时通知' },
  { key: 'groupInviteNotification' as const, label: '群组邀请', desc: '被邀请加入群组时通知' },
  { key: 'systemNotification' as const, label: '系统通知', desc: '系统公告和更新通知' },
  { key: 'soundEnabled' as const, label: '通知音效', desc: '收到通知时播放提示音' },
  { key: 'desktopNotificationEnabled' as const, label: '桌面通知', desc: '在桌面显示弹窗通知' },
];

function toggleNotification(key: keyof NotificationSettings) {
  emit('update', { [key]: !(props.notificationSettings as any)[key] });
}
</script>
