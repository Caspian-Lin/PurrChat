<template>
  <div v-if="conversation" class="flex flex-col h-full bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-between px-4 py-4 gap-2 bg-bg-secondary border-b border-border-color h-[80px]"
    >
      <div class="flex items-center gap-2">
        <div class="font-semibold text-[28px] text-text-secondary leading-none">
          {{
            conversation.conversation_type === 'group'
              ? conversation.name
              : getUserUsername(getOtherUser(conversation, currentUserId))
          }}
        </div>
        <div class="flex items-center gap-2 mt-1">
          <div class="w-[12px] h-[12px] rounded-full bg-accent-color" />
          <div class="text-sm text-text-tertiary">
            <template v-if="conversation.conversation_type === 'direct'">
              UID: {{ getOtherUser(conversation, currentUserId)?.uid }}
            </template>
            <template v-else> GID: {{ conversation.id }} </template>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <!-- 群聊创建按钮
        <button
          class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="创建群聊"
          @click="$emit('create-group')"
        >
          <BsPeopleFill class="text-2xl" />
        </button> -->
        <!-- 会话详情按钮 -->
        <button
          class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="会话详情"
          @click="handleShowDetail"
        >
          <BsInfoCircle class="text-2xl" />
        </button>
      </div>
    </div>

    <!-- 消息列表 -->
    <div
      ref="messagesContainer"
      class="flex-1 overflow-y-auto p-4 space-y-2 bg-bg-quaternary border-b border-border-color"
    >
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
              background: 'var(--strong-background-color)',
              color: 'var(--text-color)',
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

    <!-- 分割器 -->
    <ResizableSplitter
      direction="vertical"
      :initial-position="inputAreaHeight"
      :min-position="200"
      :max-position="600"
      storage-key="chat-input-height"
      @resize="handleSplitterResize"
    />

    <!-- 消息输入区 -->
    <div
      class="flex flex-col bg-bg-primary border-t border-border-color"
      :style="{ height: `${inputAreaHeight}px` }"
    >
      <!-- 文本选项 -->
      <div class="flex items-center gap-3 px-4 py-3">
        <button
          class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="表情"
        >
          <BsEmojiSmile class="text-2xl" />
        </button>
        <button
          class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="文件"
        >
          <BsPaperclip class="text-2xl" />
        </button>
        <button
          class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
          title="截图"
        >
          <BsCamera class="text-2xl" />
        </button>
        <div class="h-[18px] w-px bg-border-color" />
        <button
          class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
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
      <div class="flex justify-end pb-8 pr-8">
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
import { ref, onMounted, nextTick, watch } from 'vue';
import { getUserUsername, getOtherUser } from '../../utils/userHelpers';
import { formatTime, formatTimeWithSeconds } from '../../utils/formatTime';
import {
  BsEmojiSmile,
  BsPaperclip,
  BsCamera,
  BsCameraVideo,
  BsInfoCircle,
} from 'vue-icons-plus/bs';
import ResizableSplitter from '../common/Splitter.vue';
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
  'create-group': [];
  'show-detail': [];
}>();

const newMessage = ref('');
const inputAreaHeight = ref(300);
const messagesContainer = ref<HTMLElement | null>(null);

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
  }
};

const handleSend = () => {
  console.log(
    '[ChatWindow] handleSend called, conversation:',
    props.conversation?.id,
    'newMessage:',
    newMessage.value
  );
  if (!props.conversation?.id || !newMessage.value.trim()) {
    console.log('[ChatWindow] handleSend returning early, no conversation or empty message');
    return;
  }
  console.log('[ChatWindow] Emitting send-message event with content:', newMessage.value);
  emit('send-message', newMessage.value);
  newMessage.value = '';
};

const handleShowDetail = () => {
  if (!props.conversation) return;

  if (props.conversation.conversation_type === 'group') {
    emit('show-detail');
  } else {
    const otherUser = getOtherUser(props.conversation, props.currentUserId);
    if (otherUser) {
      emit('show-user', otherUser);
    }
  }
};

const handleSplitterResize = (height: number) => {
  inputAreaHeight.value = height;
};

// 监听消息变化，自动滚动到底部
watch(
  () => props.messages,
  async () => {
    await scrollToBottom();
  },
  { deep: true }
);

onMounted(() => {
  // 从localStorage恢复输入区高度
  const savedHeight = localStorage.getItem('chat-input-height');
  if (savedHeight) {
    const height = parseInt(savedHeight, 10);
    if (!isNaN(height) && height >= 200 && height <= 600) {
      inputAreaHeight.value = height;
    }
  }
  // 组件挂载后滚动到底部
  scrollToBottom();
});

// 暴露方法给父组件
defineExpose({
  scrollToBottom,
});
</script>

<style scoped>
textarea::-webkit-scrollbar {
  display: none;
}
</style>
