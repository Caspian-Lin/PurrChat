<template>
  <div v-if="conversation" class="flex flex-col h-full bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-between p-3 pt-5 gap-2 bg-bg-secondary border-b border-border-color flex-shrink-0"
    >
      <div class="flex items-center gap-2">
        <div class="font-semibold text-2xl text-text-secondary leading-none">
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

    <!-- 可调整大小的容器：包含消息列表和输入区 -->
    <div class="flex flex-col flex-1 overflow-hidden">
      <!-- 消息列表 -->
      <CustomScrollbar
        ref="messagesContainer"
        class="flex-1 bg-bg-quaternary border-b border-border-color min-h-0"
      >
        <div class="p-4 space-y-2">
          <template v-for="(message, index) in messages" :key="message.id">
            <!-- 时间分割线 -->
            <div v-if="timeDividers.has(index)" class="flex justify-center py-2">
              <!-- <div class="flex-1 h-px" style="background: var(--border-color)"></div> -->
              <span class="px-3 text-xs text-text-tertiary whitespace-nowrap">
                {{ timeDividers.get(index) }}
              </span>
              <!-- <div class="flex-1 h-px" style="background: var(--border-color)"></div> -->
            </div>

            <!-- 消息行 -->
            <div
              :class="['flex gap-3', { 'flex-row-reverse': message.sender_id === currentUserId }]"
            >
              <!-- 头像 -->
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

              <!-- 消息内容 -->
              <div class="w-fit max-w-[75%]">
                <!-- 对方的消息显示昵称 -->
                <div
                  v-if="message.sender_id !== currentUserId"
                  class="text-lg font-semibold text-text-tertiary mb-0.5"
                >
                  {{ message.sender?.username }}
                </div>
                <div
                  class="relative px-4 py-2 rounded-2xl cursor-default"
                  :style="{
                    background: 'var(--strong-background-color)',
                    color: 'var(--text-color)',
                    wordBreak: 'break-word',
                    overflowWrap: 'break-word',
                    whiteSpace: 'pre-wrap',
                  }"
                  @mouseenter="onBubbleMouseEnter(message.id)"
                  @mouseleave="onBubbleMouseLeave"
                  @dblclick="onBubbleDoubleClick(message.id)"
                >
                  {{ message.content }}
                  <!-- 消息发送状态指示器（仅显示自己的消息） -->
                  <div
                    v-if="message.sender_id === currentUserId && message.sendStatus"
                    :class="[
                      'absolute top-2 right-2 w-2 h-2 rounded-full',
                      {
                        'bg-yellow-500': message.sendStatus === 'sending',
                        'bg-green-500': message.sendStatus === 'sent',
                        'bg-red-500': message.sendStatus === 'failed',
                      },
                    ]"
                    :title="
                      {
                        sending: '发送中',
                        sent: '已发送',
                        failed: '发送失败',
                      }[message.sendStatus]
                    "
                  />
                  <!-- 精确时间提示 -->
                  <Transition name="tooltip">
                    <div
                      v-if="activeTooltipId === message.id"
                      class="absolute -bottom-7 left-1/2 -translate-x-1/2 text-xs text-text-tertiary whitespace-nowrap px-2 py-0.5 rounded-md z-10 pointer-events-none"
                      style="background: var(--surface-color); border: 1px solid var(--border-color)"
                    >
                      {{ formatTimeWithSeconds(message.created_at) }}
                    </div>
                  </Transition>
                </div>
              </div>
            </div>
          </template>

          <!-- 空状态 -->
          <div v-if="messages.length === 0" class="flex flex-col items-center justify-center text-text-tertiary">
            <div class="text-6xl mb-4">💬</div>
            <h3 class="text-2xl font-semibold mb-2 text-text-primary">欢迎来到 PurrChat</h3>
            <p>选择一个会话开始聊天</p>
          </div>
        </div>
      </CustomScrollbar>

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
        class="flex flex-col bg-bg-primary border-t border-border-color flex-shrink-0"
        :style="{ height: `${inputAreaHeight}px` }"
      >
        <!-- 文本选项 -->
        <div class="flex items-center gap-3 px-4 py-3">
          <EmojiPicker v-model="newMessage" @select="handleEmojiSelect" />
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
        <div class="flex-1 px-4 min-h-0">
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue';
import { getUserUsername, getOtherUser } from '../../utils/userHelpers';
import { formatTimeWithSeconds, computeTimeDividers } from '../../utils/formatTime';
import { BsPaperclip, BsCamera, BsCameraVideo, BsInfoCircle } from 'vue-icons-plus/bs';
import ResizableSplitter from '../common/Splitter.vue';
import CustomScrollbar from '../common/CustomScrollbar.vue';
import EmojiPicker from '../common/EmojiPicker.vue';
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
const messagesContainer = ref<InstanceType<typeof CustomScrollbar> | null>(null);

// ===== 时间分割线 =====
const timeDividers = computed(() => computeTimeDividers(props.messages));

// ===== 精确时间提示 =====
const activeTooltipId = ref<string | null>(null);
let hoverTimer: ReturnType<typeof setTimeout> | null = null;

const onBubbleMouseEnter = (messageId: string) => {
  hoverTimer = setTimeout(() => {
    activeTooltipId.value = messageId;
  }, 2000);
};

const onBubbleMouseLeave = () => {
  if (hoverTimer) {
    clearTimeout(hoverTimer);
    hoverTimer = null;
  }
  activeTooltipId.value = null;
};

const onBubbleDoubleClick = (messageId: string) => {
  if (hoverTimer) {
    clearTimeout(hoverTimer);
    hoverTimer = null;
  }
  activeTooltipId.value = activeTooltipId.value === messageId ? null : messageId;
};

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.scrollToBottom();
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

const handleEmojiSelect = (emoji: string) => {
  console.log('[ChatWindow] Emoji selected:', emoji);
};

const handleShowDetail = () => {
  if (!props.conversation) return;
  emit('show-detail');
};

const handleSplitterResize = async (height: number) => {
  inputAreaHeight.value = height;
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.updateScrollbar();
  }
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
  const savedHeight = localStorage.getItem('chat-input-height');
  if (savedHeight) {
    const height = parseInt(savedHeight, 10);
    if (!isNaN(height) && height >= 200 && height <= 600) {
      inputAreaHeight.value = height;
    }
  }
  scrollToBottom();
});

onUnmounted(() => {
  if (hoverTimer) clearTimeout(hoverTimer);
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

.tooltip-enter-active,
.tooltip-leave-active {
  transition: opacity 0.15s ease;
}
.tooltip-enter-from,
.tooltip-leave-to {
  opacity: 0;
}
</style>
