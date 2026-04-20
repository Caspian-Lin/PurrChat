<template>
  <div class="flex flex-col h-full">
    <!-- 列表 -->
    <div v-if="bots.length > 0" class="flex-1 overflow-y-auto">
      <div
        v-for="bot in bots"
        :key="bot.id"
        class="flex items-start gap-3 px-3 py-3 cursor-pointer transition-colors group"
        :class="[activeBotId === bot.id ? 'bg-hover-bg' : 'hover:bg-hover-bg']"
        @click="$emit('select', bot.id)"
      >
        <!-- 头像 -->
        <div
          class="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 text-white font-bold text-sm mt-0.5"
          style="background: var(--theme-primary)"
        >
          <BsCpu v-if="!bot.avatar_url" :size="20" />
          <img
            v-else
            :src="bot.avatar_url"
            :alt="bot.name"
            class="w-full h-full rounded-xl object-cover"
            referrerpolicy="no-referrer"
          />
        </div>

        <!-- 信息 -->
        <div class="flex-1 min-w-0">
          <!-- 名称行 -->
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-text-primary truncate">{{ bot.name }}</span>
            <span
              v-if="bot.status === 'disabled'"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-bg-quaternary text-text-quaternary flex-shrink-0"
            >
              已禁用
            </span>
          </div>

          <!-- 描述 -->
          <p class="text-xs text-text-tertiary truncate mt-0.5">
            {{ bot.description || (isSearch ? getVisibilityLabel(bot.visibility) : '无描述') }}
          </p>

          <!-- 搜索模式下的额外信息 -->
          <div
            v-if="isSearch && isPublicBotDetail(bot)"
            class="flex items-center gap-2 mt-1 flex-wrap"
          >
            <span
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-bg-quaternary text-text-tertiary"
            >
              {{ bot.trigger_summary }}
            </span>
            <span
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-bg-quaternary text-text-tertiary"
            >
              {{ bot.reply_type }}
            </span>
            <span
              v-if="bot.deployment_count > 0"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-bg-quaternary text-text-tertiary"
            >
              {{ bot.deployment_count }} 次部署
            </span>
            <span
              v-if="bot.owner_name"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-bg-quaternary text-text-tertiary"
            >
              {{ bot.owner_name }}
            </span>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div
          class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 mt-0.5"
        >
          <template v-if="isSearch">
            <!-- 搜索模式：添加好友 -->
            <button
              class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
              title="添加好友"
              @click.stop="$emit('create-conversation', bot.id)"
            >
              <BsChatDots :size="14" />
            </button>
          </template>
          <template v-else>
            <!-- 我的 Bot：对话 + 删除 -->
            <button
              class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
              title="开始对话"
              @click.stop="$emit('create-conversation', bot.id)"
            >
              <BsChatDots :size="14" />
            </button>
            <button
              class="p-1.5 rounded-lg hover:bg-red-500/10 text-text-tertiary hover:text-red-500 transition-colors"
              title="删除"
              @click.stop="$emit('delete', bot.id)"
            >
              <BsTrash :size="14" />
            </button>
          </template>
        </div>
      </div>
    </div>

    <!-- 加载更多按钮 -->
    <div v-if="isSearch && hasMore" class="px-3 py-2 border-t border-border-subtle flex-shrink-0">
      <button
        class="w-full py-1.5 text-xs text-text-tertiary hover:text-text-secondary transition-colors"
        :disabled="loading"
        @click="$emit('load-more')"
      >
        {{ loading ? '加载中...' : '加载更多' }}
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-else-if="loading" class="flex-1 flex items-center justify-center">
      <div
        class="w-6 h-6 border-2 border-text-quaternary border-t-[var(--theme-primary)] rounded-full animate-spin"
      />
    </div>

    <!-- 空状态 -->
    <div v-else class="flex-1 flex items-center justify-center px-6">
      <div class="text-center">
        <BsCpu :size="32" class="mx-auto text-text-quaternary mb-3" />
        <p class="text-sm text-text-tertiary">
          {{ isSearch ? '没有找到匹配的 Bot' : '还没有创建 Bot' }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BsCpu, BsChatDots, BsTrash } from 'vue-icons-plus/bs';
import type { Bot, PublicBotDetail } from '../../../../models/types';

interface Props {
  bots: Bot[];
  activeBotId: string | null;
  loading?: boolean;
  isSearch?: boolean;
  hasMore?: boolean;
}

defineProps<Props>();

defineEmits<{
  select: [botId: string];
  delete: [botId: string];
  'create-conversation': [botId: string];
  'load-more': [];
}>();

function isPublicBotDetail(bot: Bot): bot is PublicBotDetail {
  return 'deployment_count' in bot && 'trigger_summary' in bot;
}

function getVisibilityLabel(visibility: string): string {
  const labels: Record<string, string> = {
    private: '私有',
    public: '公开',
    global: '系统',
  };
  return labels[visibility] || visibility;
}
</script>
