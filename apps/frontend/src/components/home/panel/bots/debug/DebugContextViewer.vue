<template>
  <div class="debug-context">
    <div v-if="messages.length === 0" class="debug-context__empty">暂无上下文</div>
    <div v-else class="debug-context__list">
      <div
        v-for="(msg, index) in messages"
        :key="index"
        class="debug-context__message"
        :class="`debug-context__message--${msg.role}`"
      >
        <span class="debug-context__role">{{ msg.role === 'user' ? '用户' : 'Bot' }}</span>
        <span class="debug-context__content">{{ msg.content }}</span>
      </div>
    </div>
    <div v-if="round > 0" class="debug-context__footer">
      第 {{ round }} 轮对话 · {{ messages.length }} 条上下文
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DebugContextMessage } from '../../../../../models/types';

interface Props {
  messages: DebugContextMessage[];
  round: number;
}

defineProps<Props>();
</script>

<style scoped>
.debug-context {
  display: flex;
  flex-direction: column;
  height: 280px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  background: var(--bg-quaternary, #faf9f7);
  overflow: hidden;
}

.debug-context__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-size: 13px;
  color: var(--text-tertiary, #999);
}

.debug-context__list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.debug-context__message {
  display: flex;
  gap: 8px;
  padding: 6px 10px;
  border-radius: var(--radius-xs, 4px);
  font-size: 12px;
  line-height: 1.5;
}

.debug-context__message--user {
  background: rgba(90, 143, 78, 0.06);
}

.debug-context__message--assistant {
  background: rgba(0, 0, 0, 0.03);
}

.debug-context__role {
  flex-shrink: 0;
  font-weight: 500;
  color: var(--text-secondary, #666);
  min-width: 28px;
}

.debug-context__message--user .debug-context__role {
  color: var(--theme-primary, #5a8f4e);
}

.debug-context__content {
  color: var(--text-primary, #1a1a1a);
  word-break: break-word;
}

.debug-context__footer {
  flex-shrink: 0;
  padding: 6px 12px;
  border-top: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  font-size: 11px;
  color: var(--text-tertiary, #999);
}
</style>
