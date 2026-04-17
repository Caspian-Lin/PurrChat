<template>
  <div class="flex flex-col h-full bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-between px-4 py-3 gap-2 bg-bg-secondary border-b border-border-color flex-shrink-0"
    >
      <div class="flex items-top gap-2 pt-2">
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
      <CustomScrollbar
        ref="messagesContainer"
        class="flex-1 bg-bg-quaternary border-b border-border-color min-h-0"
      >
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
            <div :class="['flex gap-3', { 'flex-row-reverse': message.role === 'user' }]">
              <!-- 头像 -->
              <div class="w-10 h-10 rounded-xl overflow-hidden flex-shrink-0">
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
              <div
                class="group relative w-fit max-w-[75%]"
                @mouseenter="hoveredMessageId = message.id"
                @mouseleave="onBubbleMouseLeave"
                @dblclick="onBubbleDoubleClick(message.id)"
              >
                <!-- 思维链区域（仅 AI 消息，有 thinking 内容时显示） -->
                <div v-if="message.role === 'assistant' && message.thinking" class="mb-1">
                  <button
                    class="flex items-center gap-1.5 text-xs text-text-quaternary hover:text-text-secondary transition-colors cursor-pointer select-none"
                    @click.stop="toggleThinkingExpand(message.id)"
                  >
                    <BsChevronDown
                      :size="12"
                      class="transition-transform duration-200"
                      :class="{ 'rotate-[-90deg]': !expandedThinking.has(message.id) }"
                    />
                    <span>{{ message.isThinking ? '正在思考...' : '思维链' }}</span>
                    <span
                      v-if="message.isThinking"
                      class="inline-block w-1.5 h-1.5 rounded-full bg-accent-color thinking-dot"
                    ></span>
                  </button>
                  <Transition name="thinking-expand">
                    <div
                      v-show="message.isThinking || expandedThinking.has(message.id)"
                      class="mt-1 px-3 py-2 rounded-[var(--radius-md)] text-sm text-text-quaternary overflow-hidden"
                      :class="message.isThinking ? 'thinking-active' : ''"
                      style="
                        background: var(--strong-background-color);
                        border: 1px solid var(--border-color);
                        max-height: 300px;
                        overflow-y: auto;
                      "
                    >
                      <div style="white-space: pre-wrap; opacity: 0.7">{{ message.thinking }}</div>
                      <span
                        v-if="message.isThinking"
                        class="inline-block w-1.5 h-4 ml-0.5 align-middle bg-current opacity-50 streaming-cursor"
                      ></span>
                    </div>
                  </Transition>
                </div>

                <!-- 思考阶段指示器（只有 thinking 但还没有 content 时） -->
                <div
                  v-if="message.role === 'assistant' && message.isThinking && !message.content"
                  class="flex items-center gap-2 px-3 py-2 text-sm text-text-quaternary"
                  style="opacity: 0.6"
                >
                  <BsLightbulb :size="14" />
                  <span>正在整理回复...</span>
                  <span
                    class="inline-block w-1.5 h-4 ml-0.5 align-middle bg-current opacity-50 streaming-cursor"
                  ></span>
                </div>

                <div
                  v-if="message.role === 'user' || message.content"
                  class="relative px-3.5 py-2.5 rounded-2xl cursor-default"
                  :style="{
                    background:
                      message.role === 'user'
                        ? 'var(--theme-primary)'
                        : 'var(--strong-background-color)',
                    color: message.role === 'user' ? '#ffffff' : 'var(--text-color)',
                    wordBreak: 'break-word',
                    overflowWrap: 'break-word',
                  }"
                >
                  <!-- AI 消息：Markdown / 纯文本 渲染 -->
                  <template v-if="message.role === 'assistant'">
                    <!-- Markdown 模式 -->
                    <div
                      v-if="isMarkdownMode(message.id)"
                      class="markdown-body"
                      v-html="renderMarkdownContent(message.content)"
                    ></div>
                    <!-- 纯文本模式 -->
                    <div v-else style="white-space: pre-wrap">{{ message.content }}</div>
                  </template>
                  <!-- 用户消息：纯文本 -->
                  <div v-else style="white-space: pre-wrap">{{ message.content }}</div>

                  <!-- 流式输出光标 -->
                  <span
                    v-if="message.isStreaming && !message.isThinking"
                    class="inline-block w-2 h-5 ml-0.5 align-middle bg-current opacity-70 streaming-cursor"
                  ></span>

                  <!-- 精确时间提示 -->
                  <Transition name="tooltip">
                    <div
                      v-if="activeTooltipId === message.id"
                      class="absolute -bottom-7 left-1/2 -translate-x-1/2 text-xs text-text-tertiary whitespace-nowrap px-2 py-0.5 rounded-[var(--radius-xs)] z-10 pointer-events-none"
                      style="
                        background: var(--surface-color);
                        border: 1px solid var(--border-color);
                      "
                    >
                      {{ formatTimeWithSeconds(message.createdAt) }}
                    </div>
                  </Transition>
                </div>

                <!-- 复制按钮（hover 时显示） -->
                <Transition name="fade">
                  <button
                    v-if="hoveredMessageId === message.id && message.content"
                    class="absolute bottom-1 right-1 p-1.5 rounded-md transition-colors z-20"
                    :class="
                      message.role === 'user'
                        ? 'hover:bg-white/20 text-white/70'
                        : 'hover:bg-black/10 text-text-quaternary'
                    "
                    :title="copiedMessageId === message.id ? '已复制' : '复制'"
                    @click.stop="copyMessage(message)"
                  >
                    <BsClipboardCheck v-if="copiedMessageId === message.id" :size="14" />
                    <BsClipboard v-else :size="14" />
                  </button>
                </Transition>

                <!-- Markdown / 纯文本 切换标签（仅 AI 消息，有 content，非流式） -->
                <div
                  v-if="message.role === 'assistant' && message.content && !message.isStreaming"
                  class="mt-1 flex justify-start"
                >
                  <button
                    class="text-xs px-2 py-0.5 rounded transition-colors cursor-pointer"
                    :class="
                      isMarkdownMode(message.id)
                        ? 'text-primary hover:opacity-70'
                        : 'text-text-quaternary hover:text-text-secondary'
                    "
                    @click.stop="toggleMarkdownMode(message.id)"
                  >
                    {{ isMarkdownMode(message.id) ? 'Markdown' : '纯文本' }}
                  </button>
                </div>
              </div>
            </div>
          </template>

          <!-- 空状态 -->
          <div
            v-if="messages.length === 0"
            class="flex flex-col items-center justify-center py-20 text-text-tertiary"
          >
            <div
              class="w-20 h-20 rounded-full flex items-center justify-center mb-6"
              style="background: var(--message-sent-background)"
            >
              <BsRobot :size="36" style="color: var(--theme-primary)" />
            </div>
            <p class="text-sm">开始和 AI 对话吧</p>
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
        class="flex flex-col px-4 pt-2 bg-bg-primary border-t border-border-subtle flex-shrink-0"
        :style="{ height: `${inputAreaHeight}px` }"
      >
        <div class="flex-1 min-h-0">
          <textarea
            ref="textareaRef"
            v-model="newMessage"
            placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
            class="w-full h-full bg-transparent text-base text-text-primary resize-none outline-none placeholder:text-text-tertiary"
            @keydown="handleKeyDown"
          />
        </div>

        <div class="flex justify-end pb-4 pr-4">
          <button
            v-if="isStreaming"
            class="px-4 py-1.5 bg-red-500 hover:bg-red-600 transition-colors flex items-center justify-center text-white font-semibold text-base gap-2"
            @click="$emit('stop-generation')"
          >
            <BsStopFill :size="16" />
            停止
          </button>
          <button
            v-else
            class="px-4 py-1.5 bg-[var(--theme-primary)] hover:opacity-80 transition-opacity flex items-center justify-center text-white font-semibold text-base"
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
import {
  BsRobot,
  BsStars,
  BsStopFill,
  BsExclamationTriangle,
  BsClipboard,
  BsClipboardCheck,
  BsChevronDown,
  BsLightbulb,
} from 'vue-icons-plus/bs';
import ResizableSplitter from '../common/Splitter.vue';
import CustomScrollbar from '../common/CustomScrollbar.vue';
import { formatTimeWithSeconds, computeTimeDividers } from '../../utils/formatTime';
import { renderMarkdown } from '../../utils/markdown';
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
  hoveredMessageId.value = null;
};

const onBubbleDoubleClick = (messageId: string) => {
  if (hoverTimer) {
    clearTimeout(hoverTimer);
    hoverTimer = null;
  }
  activeTooltipId.value = activeTooltipId.value === messageId ? null : messageId;
};

// ===== Markdown 渲染 =====
// 缓存已渲染的 markdown HTML，避免重复解析
const markdownCache = new Map<string, string>();

const renderMarkdownContent = (text: string): string => {
  if (!text) return '';
  const cached = markdownCache.get(text);
  if (cached) return cached;
  const html = renderMarkdown(text);
  markdownCache.set(text, html);
  // 限制缓存大小
  if (markdownCache.size > 200) {
    const firstKey = markdownCache.keys().next().value;
    if (firstKey) markdownCache.delete(firstKey);
  }
  return html;
};

// ===== 消息 hover 状态 =====
const hoveredMessageId = ref<string | null>(null);

// ===== 复制功能 =====
const copiedMessageId = ref<string | null>(null);
let copiedTimer: ReturnType<typeof setTimeout> | null = null;

const copyMessage = async (message: AiMessage) => {
  try {
    await navigator.clipboard.writeText(message.content);
    copiedMessageId.value = message.id;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copiedMessageId.value = null;
    }, 1500);
  } catch {
    // fallback for non-HTTPS contexts
    const textarea = document.createElement('textarea');
    textarea.value = message.content;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    copiedMessageId.value = message.id;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copiedMessageId.value = null;
    }, 1500);
  }
};

// ===== Markdown / 纯文本 切换 =====
// true = markdown 模式（默认），false = 纯文本模式
const plainTextMessages = ref<Set<string>>(new Set());

const isMarkdownMode = (messageId: string): boolean => {
  return !plainTextMessages.value.has(messageId);
};

const toggleMarkdownMode = (messageId: string) => {
  const newSet = new Set(plainTextMessages.value);
  if (newSet.has(messageId)) {
    newSet.delete(messageId);
  } else {
    newSet.add(messageId);
  }
  plainTextMessages.value = newSet;
};

// ===== 思维链展开/折叠 =====
const expandedThinking = ref<Set<string>>(new Set());

const toggleThinkingExpand = (messageId: string) => {
  const newSet = new Set(expandedThinking.value);
  if (newSet.has(messageId)) {
    newSet.delete(messageId);
  } else {
    newSet.add(messageId);
  }
  expandedThinking.value = newSet;
};

// ===== 滚动到底部 =====
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

// 监听消息列表变化（新消息添加或流式内容更新），自动滚动到底部
watch(
  () => props.messages,
  () => {
    if (props.messages.length > 0) {
      const lastMsg = props.messages[props.messages.length - 1];
      if (lastMsg?.isStreaming || lastMsg?.isThinking) {
        scrollToBottom();
      }
    }
  }
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
  if (copiedTimer) clearTimeout(copiedTimer);
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

.thinking-dot {
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

.thinking-active {
  animation: thinking-glow 2s ease-in-out infinite;
}

@keyframes thinking-glow {
  0%,
  100% {
    border-color: var(--border-color);
  }
  50% {
    border-color: var(--theme-primary);
    opacity: 0.8;
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

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.thinking-expand-enter-active,
.thinking-expand-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}
.thinking-expand-enter-from,
.thinking-expand-leave-to {
  opacity: 0;
  max-height: 0;
  margin-top: 0;
  padding-top: 0;
  padding-bottom: 0;
}
</style>

<!-- Markdown 渲染样式（非 scoped，因为 v-html 内容不受 scoped 样式影响） -->
<style>
.markdown-body {
  font-size: 0.9375rem;
  line-height: 1.6;
}

.markdown-body p {
  margin: 0.25em 0;
}

.markdown-body p:first-child {
  margin-top: 0;
}

.markdown-body p:last-child {
  margin-bottom: 0;
}

.markdown-body h1,
.markdown-body h2,
.markdown-body h3,
.markdown-body h4,
.markdown-body h5,
.markdown-body h6 {
  margin: 0.75em 0 0.25em;
  font-weight: 600;
  line-height: 1.4;
}

.markdown-body h1 {
  font-size: 1.5em;
}

.markdown-body h2 {
  font-size: 1.3em;
}

.markdown-body h3 {
  font-size: 1.15em;
}

.markdown-body ul,
.markdown-body ol {
  margin: 0.25em 0;
  padding-left: 1.5em;
}

.markdown-body li {
  margin: 0.125em 0;
}

.markdown-body blockquote {
  margin: 0.5em 0;
  padding: 0.25em 0.75em;
  border-left: 3px solid currentColor;
  opacity: 0.8;
}

.markdown-body code {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
  font-size: 0.875em;
}

.markdown-body :not(pre) > code {
  padding: 0.15em 0.4em;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.08);
}

.markdown-body pre {
  margin: 0.5em 0;
  padding: 0.75em 1em;
  border-radius: 8px;
  overflow-x: auto;
  background: rgba(0, 0, 0, 0.06);
}

.markdown-body pre code {
  padding: 0;
  background: none;
  font-size: 0.85em;
}

.markdown-body table {
  margin: 0.5em 0;
  border-collapse: collapse;
  font-size: 0.9em;
}

.markdown-body th,
.markdown-body td {
  padding: 0.35em 0.65em;
  border: 1px solid currentColor;
  opacity: 0.7;
}

.markdown-body th {
  font-weight: 600;
}

.markdown-body a {
  text-decoration: underline;
  text-underline-offset: 2px;
}

.markdown-body img {
  max-width: 100%;
  border-radius: 8px;
}

.markdown-body hr {
  margin: 0.75em 0;
  border: none;
  border-top: 1px solid currentColor;
  opacity: 0.2;
}

.markdown-body del {
  opacity: 0.6;
}
</style>
