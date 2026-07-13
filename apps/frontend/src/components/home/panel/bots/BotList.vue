<template>
  <div class="flex flex-col h-full">
    <!-- Bot 列表 -->
    <div v-if="bots.length > 0" class="flex-1 min-h-0 overflow-y-auto">
      <div class="px-2 pt-2 pb-0.5">
        <BaseListItem
          v-for="bot in bots"
          :key="bot.id"
          :selected="activeBotId === bot.id"
          @click="$emit('select', bot.id)"
        >
          <template #avatar>
            <div
              class="w-10 h-10 rounded-[var(--radius-md)] flex items-center justify-center flex-shrink-0 text-white font-bold text-sm"
              style="background: var(--theme-primary)"
            >
              <BsCpu v-if="!bot.avatar_url" :size="20" />
              <img
                v-else
                :src="bot.avatar_url"
                :alt="bot.name"
                class="w-full h-full rounded-[var(--radius-md)] object-cover"
                referrerpolicy="no-referrer"
              />
            </div>
          </template>

          <!-- 名称行 -->
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-text-primary truncate">{{ bot.name }}</span>
            <span
              v-if="bot.bot_type === 'external'"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--theme-primary)]/10 text-[var(--theme-primary)] flex-shrink-0"
            >
              OneBot
            </span>
            <span
              v-if="bot.status === 'disabled'"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-bg-quaternary text-text-tertiary flex-shrink-0"
            >
              已禁用
            </span>
            <!-- 来源标记 -->
            <span
              v-if="!isSearch && !isOwned(bot)"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--theme-primary)]/10 text-[var(--theme-primary)] flex-shrink-0"
            >
              已安装
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

          <template #actions>
            <template v-if="isSearch">
              <!-- 搜索模式：添加好友 -->
              <button
                class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
                title="添加好友"
                aria-label="添加好友"
                @click.stop="$emit('create-conversation', bot.id)"
              >
                <BsChatDots :size="14" />
              </button>
            </template>
            <template v-else-if="isOwned(bot)">
              <!-- 我的 Bot：对话、安装到群聊、删除 -->
              <button
                class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
                title="开始对话"
                aria-label="开始对话"
                @click.stop="$emit('create-conversation', bot.id)"
              >
                <BsChatDots :size="14" />
              </button>
              <button
                class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
                title="安装到群聊"
                aria-label="安装到群聊"
                @click.stop="$emit('deploy', bot.id)"
              >
                <BsBoxArrowUpRight :size="14" />
              </button>
              <button
                class="p-1.5 rounded-lg hover:bg-red-500/10 text-text-tertiary hover:text-red-500 transition-colors"
                title="删除"
                aria-label="删除 Bot"
                @click.stop="$emit('delete', bot.id)"
              >
                <BsTrash :size="14" />
              </button>
            </template>
            <template v-else>
              <!-- 已安装的公开 Bot：对话、安装到群聊 -->
              <button
                class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
                title="开始对话"
                aria-label="开始对话"
                @click.stop="$emit('create-conversation', bot.id)"
              >
                <BsChatDots :size="14" />
              </button>
              <button
                class="p-1.5 rounded-lg hover:bg-bg-quaternary text-text-tertiary hover:text-text-primary transition-colors"
                title="安装到群聊"
                aria-label="安装到群聊"
                @click.stop="$emit('deploy', bot.id)"
              >
                <BsBoxArrowUpRight :size="14" />
              </button>
            </template>
          </template>
        </BaseListItem>
      </div>
    </div>

    <!-- 加载状态（初始加载，尚无数据） -->
    <div v-else-if="loading" class="flex-1 flex items-center justify-center">
      <div
        class="w-6 h-6 border-2 border-text-tertiary border-t-[var(--theme-primary)] rounded-full animate-spin"
      />
    </div>

    <!-- 空状态 -->
    <div v-else class="flex-1 flex flex-col items-center justify-center text-text-tertiary">
      <div
        class="w-20 h-20 rounded-full flex items-center justify-center mb-6"
        style="background: var(--surface-color)"
      >
        <BsCpu :size="36" style="color: var(--text-tertiary-color)" />
      </div>
      <p class="text-sm">
        {{ isSearch ? '没有找到匹配的 Bot' : '还没有创建 Bot' }}
      </p>
    </div>

    <!-- 加载更多按钮（独立于列表/空状态/加载状态） -->
    <div v-if="isSearch && hasMore" class="px-3 py-2 border-t border-border-subtle flex-shrink-0">
      <button
        class="w-full py-1.5 text-xs text-text-tertiary hover:text-text-secondary transition-colors"
        :disabled="loading"
        @click="$emit('load-more')"
      >
        {{ loading ? '加载中...' : '加载更多' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BsCpu, BsChatDots, BsTrash, BsBoxArrowUpRight } from 'vue-icons-plus/bs';
import BaseListItem from '../../../common/BaseListItem.vue';
import type { Bot, PublicBotDetail } from '../../../../models/types';

interface Props {
  bots: Bot[];
  activeBotId: string | null;
  loading?: boolean;
  isSearch?: boolean;
  hasMore?: boolean;
  currentUserId?: string;
}

const props = defineProps<Props>();

defineEmits<{
  select: [botId: string];
  delete: [botId: string];
  'create-conversation': [botId: string];
  deploy: [botId: string];
  'load-more': [];
}>();

function isOwned(bot: Bot): boolean {
  return bot.owner_id === props.currentUserId;
}

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
