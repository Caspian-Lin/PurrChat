<template>
  <div class="flex flex-col h-full bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-between p-3 pt-5 gap-2 bg-bg-secondary border-b border-border-color flex-shrink-0"
    >
      <div class="flex items-center gap-2">
        <div
          class="w-8 h-8 rounded-lg flex items-center justify-center"
          style="background: var(--theme-primary)"
        >
          <BsStars class="text-white" :size="16" />
        </div>
        <div>
          <div class="font-semibold text-xl text-text-primary leading-none">
            {{ conversation.title }}
          </div>
          <div class="text-sm text-text-tertiary mt-1">{{ config.model }}</div>
        </div>
      </div>
    </div>

    <!-- 可调整大小的容器 -->
    <div class="flex flex-col flex-1 overflow-hidden">
      <!-- 消息列表 -->
      <CustomScrollbar ref="messagesContainer" class="flex-1 bg-bg-quaternary border-b border-border-color min-h-0">
        <div class="p-4 space-y-2">
          <template v-for="(message, index) in messages" :key="message.id">
            <!-- 时间分割线 -->
            <div v-if="timeDividers.has(index)" class="flex items-center py-2">
              <div class="flex-1 h-px" style="background: var(--border-color)"></div>
              <span class="px-3 text-xs text-text-tertiary whitespace-nowrap">
                {{ timeDividers.get(index) }}
              </span>
              <div class="flex-1 h-px" style="background: var(--border-color)"></div>
            </div>

            <!-- 消息行 -->
            <div
              :class="['flex gap-3', { 'flex-row-reverse': message.role === 'user' }]"
            >
              <!-- 头像 -->
              <div class="w-10 h-10 rounded-lg overflow-hidden flex-shrink-0">
                <div
                  v-if="message.role === 'assistant'"
                  class="w-full h-full flex items-center justify-center"
                  style="background: var(--theme-gradient)"
                >
                  <BsRobot class="text-white" :size="20" />
                </div>
                <div
                  v-else
                  class="w-full h-full flex items-center justify-center font-bold text-white"
                  style="background: var(--theme-secondary, var(--theme-primary))"
                >
                  U
                </div>
              </div>

              <!-- 消息内容 -->
              <div class="w-fit max-w-[75%]">
                <div
                  class="relative px-4 py-2.5 rounded-2xl cursor-default"
                  :style="{
                    background: message.role === 'user' ? 'var(--theme-primary)' : 'var(--strong-background-color)',
                    color: message.role === 'user' ? '#ffffff' : 'var(--text-color)',
                    wordBreak: 'break-word',
                    overflowWrap: 'break-word',
                    whiteSpace: 'pre-wrap',
                  }"
                  @mouseenter="onBubbleMouseEnter(message.id)"
                  @mouseleave="onBubbleMouseLeave"
                  @dblclick="onBubbleDoubleClick(message.id)"
                >
                  {{ message.content }}
                  <!-- 流式输出光标 -->
                  <span
                    v-if="message.isStreaming"
                    class="inline-block w-2 h-5 ml-0.5 align-middle bg-current opacity-70 streaming-cursor"
                  ></span>
                  <!-- 精确时间提示 -->
                  <Transition name="tooltip">
                    <div
                      v-if="activeTooltipId === message.id"
                      class="absolute -bottom-7 left-1/2 -translate-x-1/2 text-xs text-text-tertiary whitespace-nowrap px-2 py-0.5 rounded-md z-10 pointer-events-none"
                      style="background: var(--surface-color); border: 1px solid var(--border-color)"
                    >
                      {{ formatTimeWithSeconds(message.createdAt) }}
                    </div>
                  </Transition>
                </div>
              </div>
            </div>
          </template>

          <!-- 空状态 -->
          <div v-if="messages.length === 0" class="flex flex-col items-center justify-center py-16 text-text-tertiary">
            <BsRobot :size="48" class="mb-4 opacity-30" />
            <p>开始和 AI 对话吧</p>
          </div>
        </div>
      </CustomScrollbar>

      <!-- 分割器 -->
      <ResizableSplitter
        direction="vertical"
        :initial-position="inputAreaHeight"
        :min-position="120"
        :max-position="500"
        storage-key="ai-input-height"
        @resize="handleSplitterResize"
      />

      <!-- 错误提示 -->
      <div
        v-if="error"
        class="px-4 py-2 bg-red-500/10 border-b border-red-500/20 text-red-500 text-sm flex items-center gap-2"
      >
        <BsExclamationTriangle :size="14" />
        {{ error }}
        <button class="ml-auto hover:underline" @click="$emit('clear-error')">关闭</button>
      </div>

      <!-- 消息输入区 -->
      <div
        class="flex flex-col bg-bg-primary border-t border-border-color flex-shrink-0"
        :style="{ height: `${inputAreaHeight}px` }"
      >
        <div class="flex-1 px-4 min-h-0">
          <textarea
            ref="textareaRef"
            v-model="newMessage"
            placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
            class="w-full h-full bg-transparent text-lg text-text-tertiary resize-none outline-none"
            @keydown="handleKeyDown"
          />
        </div>

        <div class="flex justify-end pb-8 pr-8">
          <button
            v-if="isStreaming"
            class="px-4 py-1.5 bg-red-500 hover:bg-red-600 transition-colors flex items-center justify-center text-white font-semibold text-xl gap-2"
            @click="$emit('stop-generation')"
          >
            <BsStopFill :size="16" />
            停止
          </button>
          <button
            v-else
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
import { BsRobot, BsStars, BsStopFill, BsExclamationTriangle } from 'vue-icons-plus/bs';
import ResizableSplitter from '../common/Splitter.vue';
import CustomScrollbar from '../common/CustomScrollbar.vue';
import { formatTimeWithSeconds, computeTimeDividers } from '../../utils/formatTime';
import type { AiConfig, AiConversation, AiMessage } from '../../models/types';

interface Props {
  config: AiConfig;
  conversation: AiConversation;
  messages: AiMessage[];
  isStreaming: boolean;
  error: string | null;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'send-message': [content: string];
  'stop-generation': [];
  'clear-error': [];
}>();

const newMessage = ref('');
const inputAreaHeight = ref(200);
const messagesContainer = ref<InstanceType<typeof CustomScrollbar> | null>(null);
const textareaRef = ref<HTMLTextAreaElement | null>(null);

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

const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.scrollToBottom();
  }
};

const handleSend = () => {
  if (!newMessage.value.trim() || props.isStreaming) return;
  emit('send-message', newMessage.value);
  newMessage.value = '';
  nextTick(() => {
    if (textareaRef.value) {
      textareaRef.value.style.height = 'auto';
    }
  });
};

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    handleSend();
  }
};

const handleSplitterResize = async (height: number) => {
  inputAreaHeight.value = height;
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.updateScrollbar();
  }
};

watch(
  () => props.messages.length,
  () => scrollToBottom()
);

watch(
  () => props.messages,
  () => {
    if (props.messages.length > 0) {
      const lastMsg = props.messages[props.messages.length - 1];
      if (lastMsg?.isStreaming) {
        scrollToBottom();
      }
    }
  },
  { deep: true }
);

onMounted(() => {
  const savedHeight = localStorage.getItem('ai-input-height');
  if (savedHeight) {
    const height = parseInt(savedHeight, 10);
    if (!isNaN(height) && height >= 120 && height <= 500) {
      inputAreaHeight.value = height;
    }
  }
  scrollToBottom();
});

onUnmounted(() => {
  if (hoverTimer) clearTimeout(hoverTimer);
});

defineExpose({ scrollToBottom });
</script>

<style scoped>
.streaming-cursor {
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0;
  }
}

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
