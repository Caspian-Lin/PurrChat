<template>
  <div class="bot-call-logs">
    <!-- 加载状态 -->
    <div v-if="botStore.callLogsLoading && botStore.callLogs.length === 0" class="py-8 text-center">
      <div class="text-sm text-text-tertiary">加载中...</div>
    </div>

    <!-- 空状态 -->
    <div
      v-else-if="!botStore.callLogsLoading && botStore.callLogs.length === 0"
      class="py-8 text-center"
    >
      <div class="text-sm text-text-quaternary">暂无调用记录</div>
    </div>

    <!-- 调用记录列表 -->
    <div v-else class="space-y-2">
      <div
        v-for="log in botStore.callLogs"
        :key="log.id"
        class="call-log-item"
        :class="{ 'call-log-item--failed': !log.success }"
      >
        <!-- 头部：时间、会话、状态 -->
        <div class="call-log-item__header">
          <span class="call-log-item__time">{{ formatRelativeTime(log.created_at) }}</span>
          <span v-if="log.conversation_name" class="call-log-item__conversation">
            #{{ log.conversation_name }}
          </span>
          <span class="call-log-item__spacer" />
          <span
            class="call-log-item__status"
            :class="log.success ? 'call-log-item__status--ok' : 'call-log-item__status--err'"
          >
            {{ log.success ? 'OK' : 'FAIL' }}
          </span>
          <span class="call-log-item__duration">{{ log.duration_ms }}ms</span>
          <span v-if="log.reply_type" class="call-log-item__type">{{ log.reply_type }}</span>
        </div>

        <!-- 触发消息 -->
        <div class="call-log-item__message">
          <span class="call-log-item__sender">{{ log.sender_name }}</span
          >: {{ truncate(log.trigger_message, 80) }}
        </div>

        <!-- 回复内容 -->
        <div class="call-log-item__reply">
          <span class="call-log-item__bot-label">Bot</span
          >: {{ truncate(log.reply_content, 80) }}
        </div>

        <!-- 错误信息 -->
        <div v-if="!log.success && log.error_message" class="call-log-item__error">
          {{ log.error_message }}
        </div>
      </div>
    </div>

    <!-- 加载更多 -->
    <div v-if="botStore.callLogsHasMore" class="mt-4 text-center">
      <button
        class="load-more-btn"
        :disabled="botStore.callLogsLoading"
        @click="botStore.loadMoreCallLogs(botId)"
      >
        {{ botStore.callLogsLoading ? '加载中...' : '加载更多' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { useBotStore } from '../../../../stores/bot';

interface Props {
  botId: string;
}

const props = defineProps<Props>();

const botStore = useBotStore();

onMounted(() => {
  botStore.loadCallLogs(props.botId);
});

function truncate(text: string, max: number): string {
  if (!text) return '';
  return text.length > max ? text.slice(0, max) + '...' : text;
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return '未知时间';

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSeconds < 60) return '刚刚';
  if (diffMinutes < 60) return `${diffMinutes}分钟前`;
  if (diffHours < 24) return `${diffHours}小时前`;
  if (diffDays === 1) return '昨天';
  if (diffDays < 7) return `${diffDays}天前`;

  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });

  return formatter.format(date).replace(/\//g, '-');
}
</script>

<style scoped>
.bot-call-logs {
  width: 100%;
}

.call-log-item {
  padding: 10px 12px;
  border-radius: var(--radius-sm, 8px);
  background: var(--bg-quaternary, rgba(0, 0, 0, 0.03));
  transition: background 0.15s ease;
}

.call-log-item:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.call-log-item__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
  font-size: 11px;
}

.call-log-item__time {
  color: var(--text-tertiary, #999);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.call-log-item__conversation {
  color: var(--text-secondary, #666);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}

.call-log-item__spacer {
  flex: 1;
}

.call-log-item__status {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 4px;
  letter-spacing: 0.3px;
}

.call-log-item__status--ok {
  color: #3d8b37;
  background: rgba(61, 139, 55, 0.1);
}

.call-log-item__status--err {
  color: #c0392b;
  background: rgba(192, 57, 43, 0.1);
}

.call-log-item__duration {
  color: var(--text-quaternary, #aaa);
  font-variant-numeric: tabular-nums;
}

.call-log-item__type {
  color: var(--text-quaternary, #aaa);
  font-size: 10px;
  text-transform: uppercase;
}

.call-log-item__message {
  font-size: 12px;
  color: var(--text-secondary, #666);
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.call-log-item__sender {
  color: var(--text-primary, #333);
  font-weight: 500;
}

.call-log-item__reply {
  font-size: 12px;
  color: var(--text-secondary, #666);
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}

.call-log-item__bot-label {
  color: var(--theme-primary, #5a8f4e);
  font-weight: 500;
  font-size: 11px;
}

.call-log-item__error {
  font-size: 11px;
  color: #c0392b;
  margin-top: 4px;
  line-height: 1.4;
}

.load-more-btn {
  font-size: 12px;
  color: var(--text-tertiary, #999);
  padding: 6px 16px;
  border-radius: var(--radius-sm, 8px);
  background: var(--bg-quaternary, rgba(0, 0, 0, 0.03));
  transition: all 0.15s ease;
  cursor: pointer;
}

.load-more-btn:hover:not(:disabled) {
  color: var(--text-primary);
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.load-more-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
