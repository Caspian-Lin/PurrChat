<template>
  <div class="flex flex-col h-full">
    <!-- 配置选择区域 -->
    <div class="border-b border-border-color flex-shrink-0">
      <div class="px-4 py-2 text-xs font-medium uppercase tracking-wider text-text-tertiary">
        AI 模型配置
      </div>
      <CustomScrollbar class="max-h-48">
        <div class="px-2 pt-2 pb-0.5">
          <BaseListItem
            v-for="config in configs"
            :key="config.id"
            :selected="activeConfigId === config.id"
            @click="$emit('select-config', config.id)"
          >
            <template #avatar>
              <div
                class="w-9 h-9 rounded-[var(--radius-sm)] flex items-center justify-center"
                :style="{
                  background:
                    activeConfigId === config.id
                      ? 'var(--theme-primary)'
                      : 'var(--surface-secondary-color)',
                }"
              >
                <BsRobot
                  :size="16"
                  :class="activeConfigId === config.id ? 'text-white' : 'text-text-tertiary'"
                />
              </div>
            </template>

            <div class="text-sm font-medium truncate text-text-primary">{{ config.name }}</div>
            <div class="text-xs truncate text-text-tertiary">{{ config.model }}</div>

            <template #actions>
              <button
                class="w-6 h-6 rounded-[var(--radius-xs)] flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
                title="编辑"
                @click.stop="$emit('edit-config', config)"
              >
                <BsPencil :size="12" />
              </button>
              <button
                class="w-6 h-6 rounded-[var(--radius-xs)] flex items-center justify-center hover:bg-red-500/20 transition-colors text-text-tertiary hover:text-red-500"
                title="删除"
                @click.stop="$emit('delete-config', config.id)"
              >
                <BsTrash :size="12" />
              </button>
            </template>
          </BaseListItem>
        </div>
        <div v-if="configs.length === 0" class="px-4 py-6 text-center text-text-tertiary text-sm">
          暂无配置，请先添加
        </div>
      </CustomScrollbar>
    </div>

    <!-- 会话列表区域 -->
    <div class="flex-1 min-h-0">
      <div
        class="px-4 py-2 text-xs font-medium uppercase tracking-wider text-text-tertiary border-b border-border-color flex items-center justify-between"
      >
        <span>对话历史</span>
      </div>
      <CustomScrollbar class="h-[calc(100%-28px)]">
        <div class="px-2 pt-2 pb-0.5">
          <BaseListItem
            v-for="conv in conversations"
            :key="conv.id"
            :selected="activeConversationId === conv.id"
            @click="$emit('select-conversation', conv.id)"
          >
            <template #avatar>
              <div
                class="w-9 h-9 rounded-[var(--radius-sm)] flex items-center justify-center"
                style="background: var(--surface-secondary-color)"
              >
                <BsChatLeft :size="16" class="text-text-tertiary" />
              </div>
            </template>

            <div class="text-sm font-medium truncate text-text-primary">
              {{ conv.title }}
              <span
                v-if="streamingIds?.has(conv.id)"
                class="inline-block w-1.5 h-1.5 rounded-full bg-accent-color ml-1 align-middle streaming-dot"
              ></span>
            </div>
            <div class="text-xs text-text-tertiary">
              {{ formatTime(conv.updatedAt) }}
            </div>

            <template #actions>
              <button
                class="w-6 h-6 rounded-[var(--radius-xs)] flex items-center justify-center hover:bg-red-500/20 transition-all text-text-tertiary hover:text-red-500"
                title="删除对话"
                @click.stop="$emit('delete-conversation', conv.id)"
              >
                <BsX :size="14" />
              </button>
            </template>
          </BaseListItem>
        </div>
        <div
          v-if="conversations.length === 0"
          class="px-4 py-6 text-center text-text-tertiary text-sm"
        >
          暂无对话
        </div>
      </CustomScrollbar>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BsRobot, BsPencil, BsTrash, BsX, BsChatLeft } from 'vue-icons-plus/bs';
import CustomScrollbar from '../common/CustomScrollbar.vue';
import BaseListItem from '../common/BaseListItem.vue';
import type { AiConfig, AiConversation } from '../../models/types';
import { formatTime } from '../../utils/formatTime';

interface Props {
  configs: AiConfig[];
  activeConfigId: string | null;
  conversations: AiConversation[];
  activeConversationId: string | null;
  streamingIds?: Set<string>;
}

defineProps<Props>();

defineEmits<{
  'select-config': [configId: string];
  'edit-config': [config: AiConfig];
  'delete-config': [configId: string];
  'select-conversation': [conversationId: string];
  'delete-conversation': [conversationId: string];
}>();
</script>

<style scoped>
.streaming-dot {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 0.3;
  }
  50% {
    opacity: 1;
  }
}
</style>
