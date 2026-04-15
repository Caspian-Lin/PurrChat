<template>
  <div class="flex flex-col h-full">
    <!-- 配置选择区域 -->
    <div class="border-b border-border-color flex-shrink-0">
      <div class="px-4 py-2 text-xs font-medium uppercase tracking-wider text-text-tertiary">
        AI 模型配置
      </div>
      <CustomScrollbar class="max-h-48">
        <div
          v-for="config in configs"
          :key="config.id"
          :class="[
            'flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors group',
            activeConfigId === config.id ? 'bg-selected-bg' : 'hover:bg-hover-bg',
          ]"
          @click="$emit('select-config', config.id)"
        >
          <div
            class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
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
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium truncate text-text-primary">{{ config.name }}</div>
            <div class="text-xs truncate text-text-tertiary">{{ config.model }}</div>
          </div>
          <!-- 操作按钮 -->
          <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              class="w-6 h-6 rounded flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
              title="编辑"
              @click.stop="$emit('edit-config', config)"
            >
              <BsPencil :size="12" />
            </button>
            <button
              class="w-6 h-6 rounded flex items-center justify-center hover:bg-red-500/20 transition-colors text-text-tertiary hover:text-red-500"
              title="删除"
              @click.stop="$emit('delete-config', config.id)"
            >
              <BsTrash :size="12" />
            </button>
          </div>
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
        <div
          v-for="conv in conversations"
          :key="conv.id"
          :class="[
            'flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors group',
            activeConversationId === conv.id ? 'bg-selected-bg' : 'hover:bg-hover-bg',
          ]"
          @click="$emit('select-conversation', conv.id)"
        >
          <div
            class="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
            style="background: var(--surface-secondary-color)"
          >
            <BsChatLeft :size="18" class="text-text-tertiary" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium truncate text-text-primary">{{ conv.title }}</div>
            <div class="text-xs text-text-tertiary">
              {{ formatTime(conv.updatedAt) }}
            </div>
          </div>
          <button
            class="w-6 h-6 rounded flex items-center justify-center opacity-0 group-hover:opacity-100 hover:bg-red-500/20 transition-all text-text-tertiary hover:text-red-500"
            title="删除对话"
            @click.stop="$emit('delete-conversation', conv.id)"
          >
            <BsX :size="14" />
          </button>
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
import type { AiConfig, AiConversation } from '../../models/types';
import { formatTime } from '../../utils/formatTime';

interface Props {
  configs: AiConfig[];
  activeConfigId: string | null;
  conversations: AiConversation[];
  activeConversationId: string | null;
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
