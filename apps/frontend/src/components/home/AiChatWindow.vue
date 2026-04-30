<template>
  <div class="flex flex-col flex-1 min-h-0 bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-between px-4 py-3 gap-2 bg-bg-secondary border-b border-border-color flex-shrink-0"
    >
      <div class="flex items-top gap-3 pt-2">
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
      <div
        ref="messagesContainer"
        class="flex-1 overflow-y-auto bg-bg-quaternary border-b border-border-color min-h-0"
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
                class="group relative w-fit"
                style="max-width: var(--msg-bubble-max-width, 90%)"
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
                      class="mt-1 p-4 rounded-[var(--radius-md)] text-sm text-text-quaternary overflow-hidden"
                      :class="message.isThinking ? 'thinking-active' : ''"
                      style="
                        background: var(--background-color);
                        max-height: 400px;
                        overflow-y: auto;
                      "
                      @wheel.stop
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

                <!-- ===== 用户消息：内联编辑模式 ===== -->
                <div v-if="editingMessageId === message.id" class="w-full">
                  <textarea
                    ref="editTextareaRef"
                    v-model="editingContent"
                    class="w-full p-3 rounded-2xl bg-[var(--theme-primary)] text-white text-base resize-none outline-none"
                    style="min-height: 60px"
                    @keydown="handleEditKeyDown"
                  />
                  <div class="flex items-center gap-2 mt-1">
                    <button
                      class="px-3 py-1 text-xs rounded-[var(--radius-sm)] bg-[var(--theme-primary)] text-white hover:opacity-80 transition-opacity cursor-pointer"
                      :disabled="!editingContent.trim() || isStreaming"
                      @click="submitEdit(message.id, false)"
                    >
                      发送并覆盖
                    </button>
                    <button
                      class="px-3 py-1 text-xs rounded-[var(--radius-sm)] text-text-quaternary hover:text-text-secondary transition-colors cursor-pointer"
                      style="border: 1px solid var(--border-color)"
                      :disabled="!editingContent.trim() || isStreaming"
                      @click="submitEdit(message.id, true)"
                    >
                      发送并分支
                    </button>
                    <button
                      class="px-3 py-1 text-xs rounded-[var(--radius-sm)] text-text-quaternary hover:text-text-secondary transition-colors cursor-pointer"
                      @click="cancelEdit"
                    >
                      取消
                    </button>
                  </div>
                </div>

                <!-- ===== 消息气泡 ===== -->
                <div
                  v-else-if="message.role === 'user' || message.content"
                  class="relative px-3.5 py-2.5 rounded-2xl cursor-default"
                  :style="{
                    background:
                      message.role === 'user'
                        ? 'var(--theme-primary)'
                        : message.isError
                          ? 'rgba(220, 38, 38, 0.08)'
                          : 'var(--strong-background-color)',
                    color:
                      message.role === 'user'
                        ? '#ffffff'
                        : message.isError
                          ? 'var(--color-error, #dc2626)'
                          : 'var(--text-color)',
                    wordBreak: 'break-word',
                    overflowWrap: 'break-word',
                  }"
                >
                  <!-- AI 消息：Markdown / 纯文本 渲染 -->
                  <template v-if="message.role === 'assistant'">
                    <div
                      v-if="isMarkdownMode(message.id)"
                      class="markdown-body"
                      v-html="renderMarkdownContent(message.content)"
                    ></div>
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

                <!-- ===== AI 消息操作栏 ===== -->
                <div
                  v-if="message.role === 'assistant' && !message.isStreaming && message.content"
                  class="mt-1 flex flex-wrap items-center gap-x-1 gap-y-0.5"
                >
                  <!-- 错误 / 中断状态 -->
                  <template v-if="message.isError || message.isInterrupted">
                    <BsExclamationTriangle :size="11" class="opacity-50" />
                    <span class="text-xs opacity-50">
                      {{ message.isError ? '生成失败' : '响应被中断' }}
                    </span>
                    <button
                      class="msg-action"
                      :disabled="isStreaming"
                      @click.stop="emit('retry', message.id)"
                    >
                      重试
                    </button>
                    <span class="msg-action-sep">·</span>
                    <button
                      class="msg-action"
                      :disabled="isStreaming"
                      @click.stop="emit('regenerate', message.id)"
                    >
                      重新生成
                    </button>
                    <button
                      class="msg-action"
                      :disabled="isStreaming"
                      @click.stop="emit('branch-regenerate', message.id)"
                    >
                      分支重新生成
                    </button>
                  </template>

                  <!-- 正常状态 -->
                  <template v-else>
                    <!-- Markdown / 纯文本 切换 -->
                    <button
                      class="msg-action msg-action--active"
                      @click.stop="toggleMarkdownMode(message.id)"
                    >
                      {{ isMarkdownMode(message.id) ? 'Markdown' : '纯文本' }}
                    </button>

                    <!-- 版本切换 -->
                    <template v-if="message.alternatives && message.alternatives.length > 0">
                      <span class="msg-action-sep">·</span>
                      <button class="msg-action" @click.stop="switchVersion(message.id, -1)">
                        &lt;
                      </button>
                      <span class="text-xs text-text-quaternary tabular-nums">
                        {{ currentVersionIndex(message) + 1 }}/{{ message.alternatives.length + 1 }}
                      </span>
                      <button class="msg-action" @click.stop="switchVersion(message.id, 1)">
                        &gt;
                      </button>
                    </template>

                    <span class="msg-action-sep">·</span>
                    <button class="msg-action" @click.stop="copyMessage(message)">
                      {{ copiedMessageId === message.id ? '已复制' : '复制回复' }}
                    </button>
                    <button class="msg-action" @click.stop="copyParentPrompt(message.id)">
                      {{ copiedMessageId === `prompt-${message.id}` ? '已复制' : '复制Prompt' }}
                    </button>

                    <span class="msg-action-sep">·</span>
                    <button
                      class="msg-action"
                      :disabled="isStreaming"
                      @click.stop="emit('regenerate', message.id)"
                    >
                      删除并重新生成
                    </button>
                    <button
                      class="msg-action"
                      :disabled="isStreaming"
                      @click.stop="emit('branch-regenerate', message.id)"
                    >
                      分支重新生成
                    </button>
                  </template>
                </div>

                <!-- ===== 用户消息操作栏 ===== -->
                <div
                  v-if="message.role === 'user' && !message.isStreaming"
                  class="mt-1 flex items-center gap-1"
                >
                  <button
                    class="msg-action"
                    :disabled="isStreaming"
                    @click.stop="startEdit(message)"
                  >
                    编辑
                  </button>
                  <span class="msg-action-sep">·</span>
                  <button class="msg-action" @click.stop="copyMessage(message)">
                    {{ copiedMessageId === message.id ? '已复制' : '复制' }}
                  </button>
                  <span class="msg-action-sep">·</span>
                  <button
                    class="msg-action text-red-500/70 hover:!text-red-500"
                    :disabled="isStreaming"
                    @click.stop="emit('delete-message', message.id)"
                  >
                    删除
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
      </div>

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
        <!-- 推理控件 -->
        <div class="flex items-center gap-3 pb-1.5 text-xs text-text-quaternary select-none">
          <button
            class="flex items-center gap-1.5 px-2 py-0.5 rounded-[var(--radius-xs)] transition-colors cursor-pointer"
            :class="reasoningEnabled ? 'text-primary bg-primary/10' : 'hover:text-text-secondary'"
            @click="toggleReasoning"
          >
            <BsLightbulb :size="12" />
            <span>推理</span>
          </button>
          <template v-if="reasoningEnabled">
            <span class="opacity-50">强度</span>
            <div class="flex gap-0.5">
              <button
                v-for="level in ['low', 'medium', 'high'] as const"
                :key="level"
                class="px-2 py-0.5 rounded-[var(--radius-xs)] transition-colors cursor-pointer"
                :class="
                  reasoningEffort === level
                    ? 'text-primary bg-primary/10'
                    : 'hover:text-text-secondary'
                "
                @click="setReasoningEffort(level)"
              >
                {{ level === 'low' ? '低' : level === 'medium' ? '中' : '高' }}
              </button>
            </div>
          </template>
        </div>

        <div class="flex-1 min-h-0">
          <textarea
            ref="textareaRef"
            v-model="newMessage"
            placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
            class="w-full h-full bg-transparent text-base text-text-primary resize-none outline-none placeholder:text-text-tertiary"
            @keydown="handleKeyDown"
            @paste="onPaste"
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
  BsChevronDown,
  BsLightbulb,
} from 'vue-icons-plus/bs';
import ResizableSplitter from '../common/Splitter.vue';
import { formatTimeWithSeconds, computeTimeDividers } from '../../utils/formatTime';
import { renderMarkdown } from '../../utils/markdown';
import { useAiStore } from '../../stores/ai';
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
  retry: [messageId: string];
  regenerate: [messageId: string];
  'branch-regenerate': [messageId: string];
  'edit-resend': [payload: { messageId: string; content: string; branch: boolean }];
  'delete-message': [messageId: string];
}>();

const store = useAiStore();

const newMessage = ref('');
const inputAreaHeight = ref(200);
const messagesContainer = ref<HTMLElement | null>(null);
const textareaRef = ref<HTMLTextAreaElement | null>(null);

// ===== 时间分割线 =====
const timeDividers = computed(() => computeTimeDividers(props.messages));

// ===== 精确时间提示 =====
const activeTooltipId = ref<string | null>(null);
let hoverTimer: ReturnType<typeof setTimeout> | null = null;

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
const markdownCache = new Map<string, string>();

const renderMarkdownContent = (text: string): string => {
  if (!text) return '';
  const cached = markdownCache.get(text);
  if (cached) return cached;
  const html = renderMarkdown(text);
  markdownCache.set(text, html);
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

const copyToClipboard = async (text: string, id: string) => {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
  }
  copiedMessageId.value = id;
  if (copiedTimer) clearTimeout(copiedTimer);
  copiedTimer = setTimeout(() => {
    copiedMessageId.value = null;
  }, 1500);
};

const copyMessage = (message: AiMessage) => {
  copyToClipboard(message.content, message.id);
};

// 复制 AI 消息对应的用户 prompt
const copyParentPrompt = (messageId: string) => {
  const idx = props.messages.findIndex((m) => m.id === messageId);
  if (idx < 0) return;
  for (let i = idx - 1; i >= 0; i--) {
    if (props.messages[i]!.role === 'user') {
      copyToClipboard(props.messages[i]!.content, `prompt-${messageId}`);
      return;
    }
  }
};

// ===== Markdown / 纯文本 切换 =====
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

// ===== 版本切换 =====
const currentVersionIndex = (message: AiMessage): number => {
  return message.alternatives?.length ?? 0;
};

const switchVersion = (messageId: string, direction: number) => {
  const convId = props.conversation.id;
  const msg = props.messages.find((m) => m.id === messageId);
  if (!msg || !msg.alternatives || msg.alternatives.length === 0) return;
  // direction: -1 = 向前（更旧的版本），+1 = 向后（更新的版本）
  // alternatives[0] 是最旧的，alternatives[length-1] 是最新的
  // 当前显示的是"主版本"（不在 alternatives 中）
  // 点击 < ：将主版本推入 alternatives，弹出最新替代版本
  // 点击 > ：将主版本推入 alternatives 末端，弹出最早的替代版本
  if (direction < 0) {
    // 回到上一个版本：弹出 alternatives 末尾（最新保存的）
    store.switchAlternative(convId, messageId, msg.alternatives.length - 1);
  } else {
    // 前进到下一个版本：弹出 alternatives 开头（最早保存的）
    store.switchAlternative(convId, messageId, 0);
  }
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

// ===== 用户消息内联编辑 =====
const editingMessageId = ref<string | null>(null);
const editingContent = ref('');
const editTextareaRef = ref<HTMLTextAreaElement | null>(null);

const startEdit = (message: AiMessage) => {
  editingMessageId.value = message.id;
  editingContent.value = message.content;
  nextTick(() => {
    if (editTextareaRef.value) {
      editTextareaRef.value.focus();
    }
  });
};

const cancelEdit = () => {
  editingMessageId.value = null;
  editingContent.value = '';
};

const submitEdit = (messageId: string, branch: boolean) => {
  if (!editingContent.value.trim()) return;
  emit('edit-resend', {
    messageId,
    content: editingContent.value.trim(),
    branch,
  });
  editingMessageId.value = null;
  editingContent.value = '';
};

const handleEditKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    if (editingMessageId.value && editingContent.value.trim()) {
      submitEdit(editingMessageId.value, false);
    }
  }
  if (event.key === 'Escape') {
    cancelEdit();
  }
};

// ===== 推理控件 =====
const reasoningEnabled = computed(() => props.conversation.reasoningEnabled !== false);
const reasoningEffort = computed(() => props.conversation.reasoningEffort || 'medium');

const toggleReasoning = () => {
  store.setReasoningSettings(props.conversation.id, !reasoningEnabled.value, reasoningEffort.value);
};

const setReasoningEffort = (level: string) => {
  store.setReasoningSettings(props.conversation.id, reasoningEnabled.value, level);
};

// ===== 滚动到底部 =====
const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
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

const onPaste = (event: ClipboardEvent) => {
  event.preventDefault();
  const text = event.clipboardData?.getData('text/plain') ?? '';
  const textarea = event.target as HTMLTextAreaElement;
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  textarea.value = textarea.value.substring(0, start) + text + textarea.value.substring(end);
  const cursorPos = start + text.length;
  textarea.selectionStart = textarea.selectionEnd = cursorPos;
  newMessage.value = textarea.value;
};

const handleSplitterResize = async (height: number) => {
  inputAreaHeight.value = height;
  await nextTick();
  if (messagesContainer.value) {
    // Native scrollbar updates automatically
  }
};

// 监听消息列表变化，自动滚动到底部
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
.msg-action {
  font-size: 11px;
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
  color: var(--text-quaternary-color, var(--color-text-quaternary));
  background: transparent;
  border: none;
  cursor: pointer;
  transition:
    color 0.15s,
    background 0.15s;
  white-space: nowrap;
  line-height: 1.6;
}
.msg-action:hover:not(:disabled) {
  color: var(--color-text-secondary, var(--text-secondary-color));
}
.msg-action:disabled {
  opacity: 0.35;
  cursor: default;
}
.msg-action--active {
  color: var(--color-primary, var(--theme-primary));
  opacity: 0.8;
}
.msg-action--active:hover {
  opacity: 1;
}
.msg-action-sep {
  font-size: 11px;
  color: var(--border-color);
  opacity: 0.5;
  user-select: none;
}

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

<!-- Markdown 渲染样式（非 scoped） -->
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
