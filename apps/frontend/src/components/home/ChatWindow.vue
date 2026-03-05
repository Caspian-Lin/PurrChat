<template>
  <div v-if="conversation" class="flex flex-col h-full bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-start px-4 py-4 gap-2 bg-bg-secondary border-b border-border-color h-[80px]"
    >
      <div class="font-semibold text-[28px] text-text-secondary leading-none">
        {{ getUserUsername(getOtherUser(conversation, currentUserId)) }}
      </div>
      <div class="flex items-center gap-2 mt-1">
        <div class="w-[12px] h-[12px] rounded-full bg-accent-color" />
        <div class="text-sm text-text-tertiary">
          UID: {{ getOtherUser(conversation, currentUserId)?.uid }}
        </div>
      </div>
    </div>

    <!-- 消息列表 -->
    <div class="flex-1 overflow-y-auto p-4 space-y-2 bg-bg-quaternary border-b border-border-color">
      <div
        v-for="message in messages"
        :key="message.id"
        :class="[
          'flex gap-2 max-w-[75%]',
          { 'flex-row-reverse ml-auto': message.sender_id === currentUserId },
        ]"
      >
        <div class="size-12 roundrect overflow-hidden flex-shrink-0">
          <img
            v-if="message.sender?.avatar_url"
            :src="message.sender.avatar_url"
            alt="avatar"
            class="w-full h-full object-cover"
          />
          <div
            v-else
            class="w-full h-full flex items-center justify-center font-bold text-white text-2xl"
            style="background: var(--theme-gradient)"
          >
            {{ message.sender?.username?.charAt(0) || '?' }}
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <!-- 对方的消息显示昵称，自己的消息不显示昵称 -->
          <div
            v-if="message.sender_id !== currentUserId"
            class="text-lg font-semibold text-text-tertiary"
          >
            {{ message.sender?.username }}
          </div>
          <div
            :class="[
              'px-4 py-2 rounded-2xl break-words',
              // message.sender_id === currentUserId ? 'rounded-br-none' : 'rounded-bl-none',
            ]"
            :style="{
              background:
                message.sender_id === currentUserId ? 'var(--theme-gradient)' : '#fffffffa',
              color: message.sender_id === currentUserId ? 'white' : '#000000',
            }"
          >
            {{ message.content }}
          </div>
          <div
            class="text-base text-text-tertiary"
            :title="formatTimeWithSeconds(message.created_at)"
          >
            {{ formatTime(message.created_at) }}
          </div>
        </div>
      </div>
    </div>

    <!-- 消息输入区 -->
    <div
      class="flex flex-col min-h-[300px] max-h-[800px] bg-bg-primary border-t border-border-color"
    >
      <!-- 文本选项 -->
      <div class="flex items-center gap-3 px-4 py-4">
        <button
          class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="表情"
        >
          <BsEmojiSmile class="text-2xl" />
        </button>
        <button
          class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="文件"
        >
          <BsPaperclip class="text-2xl" />
        </button>
        <button
          class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="截图"
        >
          <BsCamera class="text-2xl" />
        </button>
        <div class="h-[18px] w-px bg-border-color" />
        <button
          class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="视频通话"
        >
          <BsCameraVideo class="text-2xl" />
        </button>
      </div>

      <!-- 文本输入区 -->
      <div class="flex-1 px-4 overflow-y-auto">
        <textarea
          v-model="newMessage"
          placeholder="text here..."
          class="w-full h-full bg-transparent text-xl text-text-tertiary resize-none outline-none"
          @keydown.enter.prevent="handleSend"
        />
      </div>

      <!-- 发送按钮 -->
      <div class="flex justify-end px-8 py-8">
        <button
          class="px-4 py-1.5 bg-[var(--theme-primary)] hover:opacity-80 transition-opacity flex items-center justify-center text-white font-semibold text-xl"
          :disabled="!newMessage.trim()"
          @click="handleSend"
        >
          Send
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { getUserUsername, getOtherUser } from '../../utils/userHelpers';
import { formatTime, formatTimeWithSeconds } from '../../utils/formatTime';
import { BsEmojiSmile, BsPaperclip, BsCamera, BsCameraVideo } from 'vue-icons-plus/bs';
import type { Conversation, Message } from '../../models/types';

interface Props {
  conversation: Conversation | null;
  messages: Message[];
  currentUserId: string | undefined;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'send-message': [content: string];
  'export-messages': [];
  'show-user': [user: any];
  'update-conversation': [];
}>();

const newMessage = ref('');

const handleSend = () => {
  if (!props.conversation?.id || !newMessage.value.trim()) return;
  emit('send-message', newMessage.value);
  newMessage.value = '';
};
</script>

<style scoped>
textarea::-webkit-scrollbar {
  display: none;
}
</style>
