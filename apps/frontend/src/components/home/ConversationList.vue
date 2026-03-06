<template>
  <div class="flex-1 h-full">
    <!-- 统一的会话列表（按时间排序） -->
    <div
      v-for="conversation in sortedConversations"
      :key="conversation.id"
      :class="[
        'flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors',
        selectedId === conversation.id ? 'bg-selected-bg' : 'hover:bg-hover-bg',
      ]"
      @click="$emit('select', conversation)"
    >
      <!-- 头像 -->
      <div
        v-if="conversation.conversation_type === 'group'"
        class="w-12 h-12 rounded-lg overflow-hidden flex-shrink-0"
      >
        <div
          class="w-full h-full flex items-center justify-center font-bold text-white text-2xl"
          style="background: var(--theme-gradient)"
        >
          {{ conversation.name?.charAt(0) || 'G' }}
        </div>
      </div>
      <div
        v-else
        class="w-12 h-12 rounded-lg overflow-hidden flex-shrink-0 cursor-pointer"
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

      <!-- 内容区域 -->
      <div class="flex-1 min-w-0">
        <div class="flex justify-between items-center">
          <div class="flex items-center gap-2">
            <span class="font-semibold text-lg truncate text-text-primary">
              {{ getConversationName(conversation) }}
            </span>
            <!-- 群聊标签 -->
            <span
              v-if="conversation.conversation_type === 'group'"
              class="text-xs px-1 rounded bg-bg-secondary"
            >
              群聊
            </span>
            <!-- 好友状态标签（仅私聊） -->
            <span
              v-else-if="conversation.friendship_status"
              :class="[
                'text-xs px-1 rounded ',
                getFriendshipStatusColor(conversation.friendship_status),
              ]"
              class="bg-bg-secondary"
            >
              {{ formatFriendshipStatus(conversation.friendship_status) }}
            </span>
          </div>
          <!-- 最后一条消息的时间 -->
          <div
            v-if="conversation.last_message?.created_at"
            class="text-xs text-text-tertiary whitespace-nowrap ml-2"
          >
            {{ formatConversationTime(conversation.last_message.created_at) }}
          </div>
        </div>

        <div class="flex justify-between items-center">
          <div class="text-base truncate text-text-tertiary">
            {{ conversation.last_message?.content || '暂无消息' }}
          </div>
          <!-- 删除按钮 -->
          <button
            class="size-5 aspect-1 rounded-full flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary flex-shrink-0 ml-2"
            @click.stop="$emit('delete-conversation', conversation.id)"
            title="删除会话"
          >
            <BsX />
          </button>
        </div>
      </div>
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
import { computed } from 'vue';
import {
  getUserAvatar,
  getUserUsername,
  getOtherUser,
  formatFriendshipStatus,
  getFriendshipStatusColor,
} from '../../utils/userHelpers';
import { formatConversationTime } from '../../utils/formatTime';
import { BsX } from 'vue-icons-plus/bs';
import type { Conversation } from '../../models/types';

interface Props {
  conversations: Conversation[];
  selectedId: string | undefined;
  currentUserId: string | undefined;
}

const props = defineProps<Props>();

// 获取会话名称
const getConversationName = (conversation: Conversation): string => {
  if (conversation.conversation_type === 'group') {
    return conversation.name || '群聊';
  }
  return conversation.name || getUserUsername(getOtherUser(conversation, props.currentUserId));
};

// 按最后消息时间排序所有会话
const sortedConversations = computed(() => {
  console.log(
    '[ConversationList] sortedConversations recomputed, conversations count:',
    props.conversations.length
  );
  const sorted = [...props.conversations].sort((a, b) => {
    const timeA = a.last_message?.created_at || a.updated_at;
    const timeB = b.last_message?.created_at || b.updated_at;
    const timestampA = new Date(timeA).getTime();
    const timestampB = new Date(timeB).getTime();
    console.log(`[ConversationList] Comparing ${a.id} (${timestampA}) and ${b.id} (${timestampB}), diff: ${timestampB - timestampA}`);
    return timestampB - timestampA;
  });
  console.log(
    '[ConversationList] Sorted conversations:',
    sorted.map((c) => ({
      id: c.id,
      name: c.name,
      lastMessage: c.last_message?.content,
      lastMessageTime: c.last_message?.created_at || c.updated_at,
    }))
  );
  return sorted;
});

defineEmits<{
  select: [conversation: Conversation];
  'show-user': [user: any];
  'delete-conversation': [conversationId: string];
  'show-group-detail': [conversation: Conversation];
}>();
</script>

<style scoped>
.roundrect {
  border-radius: 8px;
}
</style>
