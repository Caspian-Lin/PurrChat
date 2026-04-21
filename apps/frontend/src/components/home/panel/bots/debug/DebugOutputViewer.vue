<template>
  <div class="debug-output">
    <div v-if="traces.length === 0" class="debug-output__empty">暂无执行结果</div>
    <template v-else>
      <!-- 事件 Tab 栏 -->
      <div class="debug-output__tabs">
        <button
          v-for="trace in traces"
          :key="trace.event_id"
          class="debug-output__tab"
          :class="[
            `debug-output__tab--${trace.status}`,
            { 'debug-output__tab--active': activeTab === trace.event_id },
          ]"
          @click="activeTab = trace.event_id"
        >
          <span class="debug-output__tab-icon">{{ typeIcon(trace.event_type) }}</span>
          <span class="debug-output__tab-name">{{ trace.event_name }}</span>
          <span v-if="trace.status === 'success'" class="debug-output__tab-duration"
            >{{ trace.duration_ms }}ms</span
          >
          <span v-else-if="trace.status === 'error'" class="debug-output__tab-status">✗</span>
          <span v-else-if="trace.status === 'pending'" class="debug-output__tab-status">…</span>
        </button>
      </div>

      <!-- 选中事件的详情 -->
      <div v-if="activeTrace" class="debug-output__detail">
        <!-- 状态行 -->
        <div class="debug-output__header">
          <span class="debug-output__badge" :class="`debug-output__badge--${activeTrace.status}`">
            {{ statusLabel(activeTrace.status) }}
          </span>
          <span v-if="activeTrace.duration_ms > 0" class="debug-output__duration">
            耗时 {{ activeTrace.duration_ms }}ms
          </span>
        </div>

        <!-- 错误信息 -->
        <div v-if="activeTrace.error" class="debug-output__error">
          <strong>错误：</strong>{{ activeTrace.error }}
        </div>

        <!-- 输入 -->
        <div class="debug-output__section">
          <h4 class="debug-output__section-title">输入</h4>
          <pre class="debug-output__code">{{ activeTrace.input || '(空)' }}</pre>
        </div>

        <!-- 输出 -->
        <div class="debug-output__section">
          <h4 class="debug-output__section-title">输出</h4>
          <pre class="debug-output__code">{{ activeTrace.output || '(空)' }}</pre>
        </div>

        <!-- 上下文（仅 LLM 事件） -->
        <div
          v-if="activeTrace.context_messages && activeTrace.context_messages.length > 0"
          class="debug-output__section"
        >
          <h4 class="debug-output__section-title">
            上下文 ({{ activeTrace.context_messages.length }} 条)
          </h4>
          <div class="debug-output__context-list">
            <div
              v-for="(msg, i) in activeTrace.context_messages"
              :key="i"
              class="debug-output__context-item"
            >
              <span class="debug-output__context-role">{{ msg.role }}</span>
              <span>{{ msg.content }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import type { EventTrace } from '../../../../../models/types';

interface Props {
  traces: EventTrace[];
}

const props = defineProps<Props>();

const activeTab = ref<string>('');

// 当 traces 变化且 activeTab 为空时，自动选中第一个
watch(
  () => props.traces,
  (traces) => {
    if (!activeTab.value && traces.length > 0) {
      activeTab.value = traces[0]!.event_id;
    }
  },
  { immediate: true }
);

const activeTrace = computed(() => {
  return props.traces.find((t) => t.event_id === activeTab.value) || null;
});

function typeIcon(type: string): string {
  const icons: Record<string, string> = { llm: '🧠', builtin: '⚙', python: '🐍', reply: '💬' };
  return icons[type] || '?';
}

function statusLabel(status: string): string {
  const labels: Record<string, string> = {
    pending: '等待中',
    running: '执行中',
    success: '成功',
    error: '失败',
  };
  return labels[status] || status;
}
</script>

<style scoped>
.debug-output {
  display: flex;
  flex-direction: column;
  height: 280px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--surface-secondary-color, #f4f1ec);
  overflow: hidden;
}

.debug-output__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 13px;
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-output__tabs {
  display: flex;
  gap: 2px;
  padding: 6px 6px 0;
  overflow-x: auto;
  flex-shrink: 0;
}

.debug-output__tab {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  font-size: 12px;
  border: none;
  border-radius: var(--radius-xs, 4px);
  background: none;
  color: var(--text-secondary-color, #57534e);
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.debug-output__tab:hover {
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.04));
}

.debug-output__tab--active {
  color: var(--text-color, #1c1917);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
  font-weight: 500;
}

.debug-output__tab--success .debug-output__tab-icon {
  opacity: 1;
}

.debug-output__tab--error .debug-output__tab-icon {
  opacity: 0.6;
}

.debug-output__tab-icon {
  font-size: 13px;
}

.debug-output__tab-duration {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-output__tab-status {
  font-size: 12px;
}

.debug-output__detail {
  flex: 1;
  overflow-y: auto;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.debug-output__header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.debug-output__badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-xs, 4px);
  font-weight: 500;
}

.debug-output__badge--success {
  background: var(--color-success-bg, rgba(22, 163, 74, 0.1));
  color: var(--color-success, #16a34a);
}

.debug-output__badge--error {
  background: var(--color-error-bg, rgba(239, 68, 68, 0.1));
  color: var(--color-error, #dc2626);
}

.debug-output__badge--pending {
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.05));
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-output__badge--running {
  background: var(--color-info-bg, rgba(37, 99, 235, 0.1));
  color: var(--color-info, #2563eb);
}

.debug-output__duration {
  font-size: 11px;
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-output__error {
  padding: 6px 10px;
  border-radius: var(--radius-xs, 4px);
  background: var(--color-error-bg, rgba(239, 68, 68, 0.06));
  color: var(--color-error, #dc2626);
  font-size: 12px;
}

.debug-output__section {
  flex-shrink: 0;
}

.debug-output__section-title {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.debug-output__code {
  font-family: 'SF Mono', 'Menlo', monospace;
  font-size: 11px;
  line-height: 1.5;
  padding: 8px 10px;
  border-radius: var(--radius-xs, 4px);
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.03));
  color: var(--text-color, #1c1917);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 80px;
  overflow-y: auto;
  margin: 0;
}

.debug-output__context-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
  max-height: 60px;
  overflow-y: auto;
}

.debug-output__context-item {
  font-size: 11px;
  line-height: 1.4;
  color: var(--text-secondary-color, #57534e);
}

.debug-output__context-role {
  font-weight: 500;
  color: var(--theme-primary, #5a8f4e);
  margin-right: 6px;
}
</style>
