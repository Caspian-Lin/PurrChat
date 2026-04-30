<template>
  <div class="flex-1 min-h-0 overflow-y-auto">
    <!-- 统一的会话列表（按时间排序） -->
    <div class="px-2 pt-2 pb-0.5">
      <BaseListItem
        v-for="conversation in sortedConversations"
        :key="conversation.id"
        :selected="selectedId === conversation.id"
        @click="$emit('select', conversation)"
      >
        <template #avatar>
          <div
            v-if="conversation.conversation_type === 'group'"
            class="w-11 h-11 rounded-[var(--radius-md)] overflow-hidden"
          >
            <img
              v-if="conversation.avatar_url"
              :src="conversation.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
              referrerpolicy="no-referrer"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white text-lg"
              style="background: var(--theme-gradient)"
            >
              {{ conversation.name?.charAt(0) || 'G' }}
            </div>
          </div>
          <div
            v-else
            class="w-11 h-11 rounded-[var(--radius-md)] overflow-hidden cursor-pointer"
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
              class="w-full h-full flex items-center justify-center font-bold text-white text-lg"
              style="background: var(--theme-gradient)"
            >
              {{ getUserUsername(getOtherUser(conversation, currentUserId)).charAt(0) }}
            </div>
          </div>
        </template>

        <!-- 内容 -->
        <div class="flex justify-between items-center">
          <div class="flex items-center gap-2">
            <span class="font-semibold text-[15px] truncate text-text-primary">
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
            {{ formatLastMessageContent(conversation.last_message) }}
          </div>
          <div class="flex items-center gap-2">
            <!-- 未读消息提示气泡 -->
            <div
              v-if="conversation.unread_count && conversation.unread_count > 0"
              class="min-w-[20px] h-5 px-1.5 rounded-full flex items-center justify-center text-xs font-bold text-white flex-shrink-0"
              style="background: var(--theme-primary)"
            >
              {{ conversation.unread_count > 99 ? '99+' : conversation.unread_count }}
            </div>
            <!-- 删除按钮（始终可见，因为未读徽标也需要可见性） -->
            <button
              class="size-5 aspect-1 rounded-full flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary flex-shrink-0"
              @click.stop="$emit('delete-conversation', conversation.id)"
              title="删除会话"
            >
              <BsX />
            </button>
          </div>
        </div>
      </BaseListItem>
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
import type { Conversation, Message } from '../../models/types';
import BaseListItem from '../common/BaseListItem.vue';

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

// 格式化最后一条消息内容（文件消息显示文件名）
const formatLastMessageContent = (message: Message | undefined): string => {
  if (!message) return '暂无消息';
  if (message.msg_type === 'file') {
    try {
      const fileContent = JSON.parse(message.content);
      return `[文件] ${fileContent.file_name}`;
    } catch {
      return '[文件]';
    }
  }
  return message.content || '暂无消息';
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
    console.log(
      `[ConversationList] Comparing ${a.id} (${timestampA}) and ${b.id} (${timestampB}), diff: ${timestampB - timestampA}`
    );
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
