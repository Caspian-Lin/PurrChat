<template>
  <div class="flex-1 overflow-y-auto">
    <div
      v-for="conversation in conversations"
      :key="conversation.id"
      :class="[
        'flex items-center gap-3 p-4 cursor-pointer transition-colors border-b border-border-color',
        selectedId === conversation.id ? 'bg-selected-bg' : 'hover:bg-hover-bg',
      ]"
      @click="$emit('select', conversation)"
    >
      <div
        class="w-12 h-12 roundrect overflow-hidden flex-shrink-0 cursor-pointer"
        @click.stop="$emit('show-user', getOtherUser(conversation, currentUserId)!)"
      >
        <img
          v-if="getUserAvatar(getOtherUser(conversation, currentUserId))"
          :src="getUserAvatar(getOtherUser(conversation, currentUserId))"
          alt="avatar"
          class="w-full h-full object-cover"
        />
        <div
          v-else
          class="w-full h-full flex items-center justify-center font-bold text-white text-2xl"
          style="background: var(--theme-gradient)"
        >
          {{ getUserUsername(getOtherUser(conversation, currentUserId)).charAt(0) }}
        </div>
      </div>
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <span class="font-semibold text-lg truncate text-text-primary">
            {{ conversation.name || getUserUsername(getOtherUser(conversation, currentUserId)) }}
          </span>
          <span
            v-if="conversation.friendship_status"
            :class="[
              'text-xs px-2 py-1 rounded',
              getFriendshipStatusColor(conversation.friendship_status),
            ]"
            class="bg-bg-secondary"
          >
            {{ formatFriendshipStatus(conversation.friendship_status) }}
          </span>
        </div>
        <div class="text-base truncate text-text-tertiary">
          {{ conversation.last_message?.content || '暂无消息' }}
        </div>
      </div>
      <button
        class="size-2 aspect-1 rounded-sm flex items-center justify-center hover:bg-bg-quaternary transition-colors text-text-tertiary hover:text-text-primary"
        @click.stop="$emit('delete-conversation', conversation.id)"
        title="删除会话"
      >
        <BsTrash3 class="flex-1" />
      </button>
    </div>
    <div
      v-if="conversations.length === 0"
      class="flex flex-col items-center justify-center h-full text-center p-8 text-text-tertiary"
    >
      <p>暂无会话</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  getUserAvatar,
  getUserUsername,
  getOtherUser,
  formatFriendshipStatus,
  getFriendshipStatusColor,
} from '../../utils/userHelpers';
import { BsTrash3 } from 'vue-icons-plus/bs';
import type { Conversation } from '../../models/types';

interface Props {
  conversations: Conversation[];
  selectedId: string | undefined;
  currentUserId: string | undefined;
}

defineProps<Props>();

defineEmits<{
  select: [conversation: Conversation];
  'show-user': [user: any];
  'delete-conversation': [conversationId: string];
}>();
</script>

<style scoped></style>
