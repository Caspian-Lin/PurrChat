<template>
  <div v-if="conversation" class="flex flex-col flex-1 min-h-0 bg-bg-tertiary">
    <!-- 聊天头部 -->
    <div
      class="flex items-center justify-between px-4 py-3 gap-2 bg-bg-secondary border-b border-border-color flex-shrink-0"
    >
      <div class="flex items-center gap-2">
        <div class="font-semibold text-lg text-text-secondary leading-none">
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

    <!-- 工作流状态条 -->
    <div v-if="activeWorkflow" class="workflow-banner">
      <BsCpu :size="14" class="workflow-banner__icon" />
      <span class="workflow-banner__text"> {{ activeWorkflow.bot_name }} · Agent 模式运行中 </span>
      <span class="workflow-banner__dot" />
      <button class="workflow-banner__stop" @click="handleDeactivateWorkflow">结束</button>
    </div>

    <!-- 可调整大小的容器：包含消息列表和输入区 -->
    <div class="flex flex-col flex-1 overflow-hidden">
      <!-- 消息列表 -->
      <div
        ref="messagesContainer"
        class="flex-1 overflow-y-auto bg-bg-quaternary border-b border-border-color min-h-0"
      >
        <div class="p-4 space-y-4">
          <template v-for="(message, index) in messages" :key="message.id">
            <!-- 时间分割线 -->
            <div v-if="timeDividers.has(index)" class="flex justify-center py-2">
              <span class="px-3 text-xs text-text-tertiary whitespace-nowrap">
                {{ timeDividers.get(index) }}
              </span>
            </div>

            <!-- 系统消息（居中，无头像） -->
            <div
              v-if="message.msg_type === 'system'"
              class="flex justify-center py-1.5"
              :class="{ 'poke-message': isPokeMessage(message) }"
            >
              <span
                class="px-3 py-1 text-xs rounded-full whitespace-nowrap"
                :class="
                  isPokeMessage(message)
                    ? 'text-[var(--theme-primary)] bg-[color-mix(in_srgb,var(--theme-primary)_8%,transparent)]'
                    : 'text-text-tertiary bg-bg-quaternary'
                "
              >
                {{ getSystemMessageText(message) }}
              </span>
            </div>

            <!-- 消息行 -->
            <div
              v-else
              :class="['flex gap-3', { 'flex-row-reverse': message.sender_id === currentUserId }]"
              @touchstart.passive="onMessageTouchStart($event, message)"
              @touchend="onMessageTouchEnd($event)"
              @touchmove="onMessageTouchMove($event)"
            >
              <!-- 头像 -->
              <div
                class="size-10 rounded-xl overflow-hidden flex-shrink-0 cursor-pointer"
                @contextmenu.prevent="onAvatarContextMenu($event, message)"
              >
                <!-- Bot 消息头像 -->
                <template v-if="message.sender?.is_bot || message.bot_id">
                  <img
                    v-if="message.sender?.avatar_url"
                    :src="message.sender.avatar_url"
                    alt="bot avatar"
                    class="w-full h-full object-cover"
                  />
                  <div
                    v-else
                    class="w-full h-full flex items-center justify-center text-white"
                    style="background: var(--theme-primary)"
                  >
                    <BsCpu :size="20" />
                  </div>
                </template>
                <!-- 普通消息头像 -->
                <template v-else>
                  <img
                    v-if="message.sender?.avatar_url"
                    :src="message.sender.avatar_url"
                    alt="avatar"
                    class="w-full h-full object-cover"
                  />
                  <div
                    v-else
                    class="w-full h-full flex items-center justify-center font-bold text-white text-lg"
                    style="background: var(--theme-gradient)"
                  >
                    {{ message.sender?.username?.charAt(0) || '?' }}
                  </div>
                </template>
              </div>

              <!-- 消息内容 -->
              <div class="w-fit" style="max-width: var(--msg-bubble-max-width, 75%)">
                <!-- 对方的消息显示昵称 -->
                <div
                  v-if="message.sender_id !== currentUserId"
                  class="text-md font-semibold text-text-tertiary mb-0.5"
                >
                  {{ message.sender?.username || message.bot_name }}
                  <span
                    v-if="message.sender?.is_bot || message.bot_id"
                    class="inline-flex items-center gap-0.5 ml-1.5 text-[10px] font-normal px-1.5 py-0.5 rounded-full"
                    style="background: var(--theme-primary); color: white"
                  >
                    <BsCpu :size="10" />
                    Bot
                  </span>
                </div>
                <div
                  class="relative px-3.5 py-2.5 rounded-2xl cursor-default"
                  :style="{
                    background:
                      message.sender?.is_bot || message.bot_id
                        ? 'var(--message-bot-background, rgba(90, 143, 78, 0.08))'
                        : message.sender_id === currentUserId
                          ? 'var(--message-sent-background)'
                          : 'var(--message-received-background)',
                    color: 'var(--text-color)',
                    wordBreak: 'break-word',
                    overflowWrap: 'break-word',
                    whiteSpace: 'pre-wrap',
                  }"
                  @mouseenter="onBubbleMouseEnter(message.id)"
                  @mouseleave="onBubbleMouseLeave"
                  @dblclick="onBubbleDoubleClick(message.id)"
                  @contextmenu.prevent="onBubbleContextMenu($event, message)"
                >
                  <!-- 文件消息：图片 -->
                  <template v-if="isFileMessage(message) && getFileContent(message)?.thumbnail_url">
                    <img
                      :src="getFileContent(message)!.thumbnail_url"
                      :alt="getFileContent(message)!.file_name"
                      class="max-w-[300px] max-h-[300px] rounded-[var(--radius-md)] object-cover cursor-pointer"
                      loading="lazy"
                      @click="openImagePreview(message)"
                    />
                  </template>
                  <!-- 文件消息：非图片文件 -->
                  <template v-else-if="isFileMessage(message)">
                    <div
                      class="flex items-center gap-3 p-1 min-w-[200px] cursor-pointer"
                      @click="handleFileDownload(message)"
                    >
                      <BsFileEarmark class="text-3xl text-text-tertiary flex-shrink-0" />
                      <div class="flex flex-col gap-0.5 flex-1 min-w-0">
                        <span class="text-sm font-medium truncate">{{
                          getFileContent(message)?.file_name
                        }}</span>
                        <span class="text-xs text-text-tertiary">{{
                          formatFileSize(getFileContent(message)?.file_size || 0)
                        }}</span>
                      </div>
                      <BsDownload class="text-lg text-text-tertiary flex-shrink-0" />
                    </div>
                  </template>
                  <!-- 文本/图片消息：原有逻辑 -->
                  <template v-else>
                    {{ message.content }}
                  </template>

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
                      class="absolute -bottom-7 left-1/2 -translate-x-1/2 text-xs text-text-tertiary whitespace-nowrap px-2 py-0.5 rounded-[var(--radius-xs)] z-10 pointer-events-none"
                      style="
                        background: var(--surface-color);
                        border: 1px solid var(--border-color);
                      "
                    >
                      {{ formatTimeWithSeconds(message.created_at) }}
                    </div>
                  </Transition>
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
              <svg
                class="w-10 h-10"
                style="color: var(--theme-primary)"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="1.5"
                  d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                />
              </svg>
            </div>
            <h3 class="text-lg font-semibold mb-1 text-text-primary">开始聊天吧</h3>
            <p class="text-sm">选择一个会话开始聊天</p>
          </div>
        </div>
      </div>

      <!-- 分割器（桌面端） -->
      <ResizableSplitter
        v-if="!isMobile"
        direction="vertical"
        :initial-position="inputAreaHeight"
        :min-position="200"
        :max-position="600"
        storage-key="chat-input-height"
        @resize="handleSplitterResize"
      />
      <!-- 消息输入区 -->
      <div
        class="flex flex-col bg-bg-primary border-t border-border-subtle flex-shrink-0"
        :class="{ 'border-dashed border-2 border-[var(--theme-primary)]': !isMobile && isDragOver }"
        :style="{ height: isMobile ? '180px' : `${inputAreaHeight}px` }"
        @dragover.prevent="!isMobile && (isDragOver = true)"
        @dragleave.prevent="!isMobile && (isDragOver = false)"
        @drop.prevent="!isMobile && handleDrop($event)"
      >
        <!-- 文件预览卡片（上传完成后显示） -->
        <div v-if="fileData || fileUploading" class="px-4 pt-2">
          <div
            class="flex items-center gap-3 p-2 rounded-lg"
            style="background: var(--surface-color); border: 1px solid var(--border-color)"
          >
            <!-- 图片预览 -->
            <div v-if="thumbnailDataUrl" class="w-12 h-12 rounded overflow-hidden flex-shrink-0">
              <img :src="thumbnailDataUrl" alt="preview" class="w-full h-full object-cover" />
            </div>
            <!-- 文件图标 -->
            <div
              v-else
              class="w-12 h-12 rounded flex items-center justify-center flex-shrink-0"
              style="background: var(--bg-quaternary)"
            >
              <BsFileEarmark class="text-2xl text-text-tertiary" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium truncate">
                {{ fileData?.file_name || '上传中...' }}
              </div>
              <div class="text-xs text-text-tertiary">
                {{ fileData ? formatFileSize(fileData.file_size) : '正在上传...' }}
              </div>
            </div>
            <!-- 上传进度条 -->
            <div v-if="fileUploading" class="w-20">
              <div class="h-1 rounded-full overflow-hidden" style="background: var(--border-color)">
                <div
                  class="h-1 rounded-full bg-[var(--theme-primary)] transition-all duration-300"
                  :style="{ width: `${fileUploadProgress}%` }"
                />
              </div>
            </div>
            <!-- 移除按钮 -->
            <button v-else class="p-1 hover:bg-hover-bg rounded" @click="removePendingFile">
              <BsX class="text-lg text-text-tertiary" />
            </button>
          </div>
        </div>

        <!-- 文本选项 -->
        <div class="flex items-center gap-2 px-3 py-2">
          <EmojiPicker v-model="newMessage" @select="handleEmojiSelect" />
          <button
            class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm)]"
            title="文件"
            @click="handleFileSelect"
          >
            <BsPaperclip class="text-xl" />
          </button>
          <!-- 桌面端专属按钮 -->
          <template v-if="!isMobile">
            <button
              class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm)]"
              title="截图"
            >
              <BsCamera class="text-xl" />
            </button>
            <div class="h-[18px] w-px bg-border-color" />
            <button
              class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm)]"
              title="视频通话"
            >
              <BsCameraVideo class="text-xl" />
            </button>
          </template>
          <!-- 隐藏的文件输入 -->
          <input ref="fileInputRef" type="file" class="hidden" @change="handleFileChange" />
        </div>

        <!-- 文本输入区 -->
        <div class="flex-1 px-3 min-h-0">
          <textarea
            v-model="newMessage"
            :placeholder="isMobile ? '输入消息...' : '输入消息... (Enter 发送)'"
            class="w-full h-full bg-transparent text-base text-text-primary resize-none outline-none placeholder:text-text-tertiary"
            @keydown="handleKeyDown"
            @paste="onPaste"
          />
        </div>

        <!-- 发送按钮 -->
        <div class="flex justify-end" :class="isMobile ? 'pb-2 pr-3' : 'pb-4 pr-4'">
          <button
            class="px-4 py-1.5 bg-[var(--theme-primary)] hover:opacity-80 transition-opacity flex items-center justify-center text-white font-semibold text-base disabled:opacity-50 disabled:cursor-not-allowed rounded-[var(--radius-sm)]"
            :disabled="sendDisabled"
            @click="handleSend"
          >
            发送
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- 图片预览器 -->
  <ImagePreviewModal
    v-model:show="showImagePreview"
    :image-url="previewImageUrl"
    :file-name="previewFileName"
    @download="handlePreviewDownload"
  />

  <!-- 上下文菜单 -->
  <MessageContextMenu
    :visible="contextMenu.visible"
    :position="contextMenu.position"
    :actions="contextMenu.message ? getContextMenuActions(contextMenu.message) : []"
    @close="contextMenu.visible = false"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue';
import { getUserUsername, getOtherUser } from '../../utils/userHelpers';
import { formatTimeWithSeconds, computeTimeDividers } from '../../utils/formatTime';
import {
  BsPaperclip,
  BsCamera,
  BsCameraVideo,
  BsInfoCircle,
  BsFileEarmark,
  BsDownload,
  BsX,
  BsCpu,
  BsClipboard,
} from 'vue-icons-plus/bs';
import ResizableSplitter from '../common/Splitter.vue';
import EmojiPicker from '../common/EmojiPicker.vue';
import ImagePreviewModal from '../common/ImagePreviewModal.vue';
import MessageContextMenu from '../chat/MessageContextMenu.vue';
import type { ContextMenuAction } from '../chat/MessageContextMenu.vue';
import { useFileUpload } from '../../composables/useFileUpload';
import { useNotification } from '../../composables/useNotification';
import { useLongPress } from '../../composables/useLongPress';
import { usePlatform } from '../../composables/usePlatform';
import { api } from '../../models/api';
import { platformAdapters } from '../../platform';
import { websocketEventManager } from '../../services/websocketEventManager';
import { formatSystemMessageText } from '../../utils/messageHelpers';
import type { Conversation, Message, FileMessageContent } from '../../models/types';

interface Props {
  conversation: Conversation | null;
  messages: Message[];
  currentUserId: string | undefined;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'send-message': [content: string];
  'send-file-message': [fileData: FileMessageContent];
  'export-messages': [];
  'show-user': [user: any];
  'update-conversation': [];
  'create-group': [];
  'show-detail': [];
}>();

const newMessage = ref('');
const inputAreaHeight = ref(300);
const messagesContainer = ref<HTMLElement | null>(null);
const fileInputRef = ref<HTMLInputElement | null>(null);
const isDragOver = ref(false);

// ===== 上下文菜单（拍一拍） =====
const { isMobile } = usePlatform();
const contextMenu = ref<{
  visible: boolean;
  position: { x: number; y: number };
  message: Message | null;
  source: 'bubble' | 'avatar';
}>({ visible: false, position: { x: 0, y: 0 }, message: null, source: 'bubble' });

const { handlers: longPressHandlers } = useLongPress((pos) => {
  // 长按触发时，通过当前触摸位置找到对应的消息
  // 这里简化处理：使用最后记录的消息
  if (touchTargetMessage) {
    showContextMenu(pos.x, pos.y, touchTargetMessage);
  }
});

// 记录当前触摸的消息（用于长按）
let touchTargetMessage: Message | null = null;

function onBubbleContextMenu(event: MouseEvent, message: Message) {
  event.preventDefault();
  showContextMenu(event.clientX, event.clientY, message, 'bubble');
}

function onAvatarContextMenu(event: MouseEvent, message: Message) {
  event.preventDefault();
  showContextMenu(event.clientX, event.clientY, message, 'avatar');
}

function onMessageTouchStart(event: TouchEvent, message: Message) {
  touchTargetMessage = message;
  contextMenu.value.message = message;
  longPressHandlers.onTouchstart(event);
}

function onMessageTouchEnd(_event: TouchEvent) {
  longPressHandlers.onTouchend();
  touchTargetMessage = null;
}

function onMessageTouchMove(event: TouchEvent) {
  longPressHandlers.onTouchmove(event);
}

function showContextMenu(
  x: number,
  y: number,
  message: Message,
  source: 'bubble' | 'avatar' = 'bubble'
) {
  contextMenu.value = {
    visible: true,
    position: { x, y },
    message,
    source,
  };
}

function getContextMenuActions(message: Message): ContextMenuAction[] {
  const actions: ContextMenuAction[] = [];

  if (contextMenu.value.source === 'bubble' && message.msg_type === 'text') {
    actions.push({
      key: 'copy',
      label: '复制',
      icon: BsClipboard,
      handler: () => copyMessageContent(message),
    });
  }

  if (contextMenu.value.source === 'avatar' && message.sender_id !== props.currentUserId) {
    actions.push({
      key: 'poke',
      label: '拍一拍',
      handler: () => handlePoke(message),
    });
  }

  return actions;
}

async function copyMessageContent(message: Message) {
  try {
    await platformAdapters.clipboard.writeText(message.content);
    useNotification().success('已复制');
  } catch {
    useNotification().error('复制失败');
  }
}

async function handlePoke(message: Message) {
  if (!props.conversation?.id || !message.sender_id) return;

  contextMenu.value.visible = false;

  try {
    await api.pokeMessage(props.conversation.id, message.sender_id);
    // 系统消息会通过 WebSocket 自动到达
  } catch (error) {
    console.error('[ChatWindow] Failed to poke:', error);
    useNotification().error('拍一拍发送失败');
  }
}

// 工作流状态
const activeWorkflow = ref<{ bot_id: string; bot_name: string; conversation_id: string } | null>(
  null
);

const workflowChangeHandler = (
  event: string,
  data: { bot_id: string; bot_name: string; conversation_id: string }
) => {
  if (event === 'started') {
    if (data.conversation_id === props.conversation?.id) {
      activeWorkflow.value = data;
    }
  } else {
    if (activeWorkflow.value?.conversation_id === data?.conversation_id) {
      activeWorkflow.value = null;
    }
  }
};
const offWorkflow = websocketEventManager.onWorkflowChange(workflowChangeHandler);

async function handleDeactivateWorkflow() {
  if (!activeWorkflow.value) return;
  try {
    await api.deactivateWorkflow(activeWorkflow.value.bot_id, activeWorkflow.value.conversation_id);
    activeWorkflow.value = null;
  } catch {
    // 静默处理
  }
}

// 文件上传
const {
  uploading: fileUploading,
  uploadProgress: fileUploadProgress,
  fileData,
  thumbnailDataUrl,
  processAndUpload,
  clearFile,
} = useFileUpload();

// 图片预览器状态
const showImagePreview = ref(false);
const previewImageUrl = ref('');
const previewFileName = ref('');

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

// ===== 发送按钮状态 =====
const sendDisabled = computed(() => {
  if (fileUploading.value) return true;
  if (!fileData.value && !newMessage.value.trim()) return true;
  return false;
});

// ===== 系统消息辅助函数 =====
function isPokeMessage(message: Message): boolean {
  try {
    const sys = JSON.parse(message.content);
    return sys.type === 'poke';
  } catch {
    return false;
  }
}

function getSystemMessageText(message: Message): string {
  return formatSystemMessageText(message, props.currentUserId);
}

// ===== 文件消息辅助函数 =====
function isFileMessage(msg: Message): boolean {
  return msg.msg_type === 'file';
}

function getFileContent(msg: Message): FileMessageContent | null {
  try {
    return JSON.parse(msg.content) as FileMessageContent;
  } catch {
    return null;
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

// ===== 文件选择和拖拽 =====
function onPaste(event: ClipboardEvent) {
  const items = event.clipboardData?.items;
  if (!items) return;

  for (const item of items) {
    if (item.kind === 'file') {
      event.preventDefault();
      const file = item.getAsFile();
      if (file) processAndUpload(file);
      return;
    }
  }
}
function handleFileSelect() {
  fileInputRef.value?.click();
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;

  clearFile();
  input.value = '';

  if (file.size > 50 * 1024 * 1024) {
    useNotification().error('文件大小不能超过 50MB');
    return;
  }

  await processAndUpload(file);
}

function handleDrop(event: DragEvent) {
  isDragOver.value = false;
  const file = event.dataTransfer?.files[0];
  if (!file) return;

  clearFile();

  if (file.size > 50 * 1024 * 1024) {
    useNotification().error('文件大小不能超过 50MB');
    return;
  }

  processAndUpload(file);
}

function removePendingFile() {
  clearFile();
}

// ===== 图片预览 =====
function openImagePreview(message: Message) {
  const fileContent = getFileContent(message);
  if (!fileContent) return;
  previewImageUrl.value = fileContent.public_url;
  previewFileName.value = fileContent.file_name;
  showImagePreview.value = true;
}

async function handleFileDownload(message: Message) {
  const fileContent = getFileContent(message);
  if (!fileContent) return;

  try {
    await platformAdapters.files.downloadUrl(fileContent.public_url, fileContent.file_name);
  } catch (error) {
    console.error('下载文件失败:', error);
    useNotification().error('下载文件失败');
  }
}

function handlePreviewDownload() {
  if (!previewImageUrl.value) return;
  platformAdapters.files.downloadUrl(previewImageUrl.value, previewFileName.value).catch(() => {
    useNotification().error('下载文件失败');
  });
}

// ===== 滚动到底部 =====
const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
  }
};

// ===== 发送消息 =====
const handleSend = () => {
  if (!props.conversation?.id) return;
  if (sendDisabled.value) return;

  // 发送文件消息
  if (fileData.value) {
    emit('send-file-message', fileData.value);
    clearFile();
  }

  // 发送文字消息
  if (newMessage.value.trim()) {
    emit('send-message', newMessage.value);
    newMessage.value = '';
  }
};

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    handleSend();
  }
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
    // Native scrollbar updates automatically
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

watch(
  () => props.conversation?.id,
  () => {
    activeWorkflow.value = null;
  }
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
  offWorkflow();
});

// 暴露方法给父组件
defineExpose({
  scrollToBottom,
});
</script>

<style scoped>
.workflow-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
  border-bottom: 1px solid color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
  flex-shrink: 0;
  font-size: 12px;
}

.workflow-banner__icon {
  color: var(--theme-primary, #5a8f4e);
}

.workflow-banner__text {
  color: var(--text-secondary, #666);
  flex: 1;
}

.workflow-banner__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--theme-primary, #5a8f4e);
  animation: pulse-dot 2s infinite;
}

@keyframes pulse-dot {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

.workflow-banner__stop {
  padding: 2px 10px;
  font-size: 11px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid color-mix(in srgb, var(--theme-primary, #5a8f4e) 30%, transparent);
  background: none;
  color: var(--theme-primary, #5a8f4e);
  cursor: pointer;
  transition: all 0.15s;
}
.workflow-banner__stop:hover {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 10%, transparent);
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

/* 拍一拍消息动画 */
.poke-message {
  animation: poke-appear 0.3s ease-out;
}

@keyframes poke-appear {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
