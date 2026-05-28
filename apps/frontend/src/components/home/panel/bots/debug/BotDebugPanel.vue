<template>
  <div class="debug-panel">
    <!-- 输入区域 -->
    <div class="debug-panel__input-row">
      <input
        v-model="message"
        type="text"
        class="debug-panel__input"
        placeholder="输入测试消息..."
        :disabled="loading"
        @keydown.enter="handleRun"
      />
      <button
        class="debug-panel__btn debug-panel__btn--primary"
        :disabled="!message.trim() || loading"
        @click="handleRun"
      >
        {{ loading ? '...' : '运行' }}
      </button>
      <button
        v-if="traceResult?.waiting_for_step"
        class="debug-panel__btn"
        :disabled="loading"
        @click="handleStep"
      >
        单步
      </button>
      <button
        v-if="sessionId"
        class="debug-panel__btn debug-panel__btn--ghost"
        @click="handleReset"
      >
        重置
      </button>
    </div>

    <!-- 执行结果区域 -->
    <div v-if="traceResult" class="debug-panel__result">
      <!-- 事件链可视化 -->
      <DebugEventFlow :events="events" :event-traces="traceResult.event_traces" />

      <!-- 执行详情 -->
      <div class="debug-panel__output-row">
        <DebugOutputViewer :traces="traceResult.event_traces" />

        <!-- Bot 回复预览 -->
        <div v-if="traceResult.reply" class="debug-panel__reply">
          <h4 class="debug-panel__reply-title">Bot 回复</h4>
          <p class="debug-panel__reply-content">{{ traceResult.reply }}</p>
        </div>
      </div>

      <!-- 上下文 -->
      <DebugContextViewer :messages="traceResult.context_messages" :round="traceResult.round" />
    </div>

    <!-- 空状态 -->
    <div v-else class="debug-panel__empty">
      <p>输入消息并点击"运行"开始调试</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { api } from '../../../../../models/api';
import type { Mechanism, DebugTraceResult, WorkflowEvent } from '../../../../../models/types';
import DebugEventFlow from './DebugEventFlow.vue';
import DebugOutputViewer from './DebugOutputViewer.vue';
import DebugContextViewer from './DebugContextViewer.vue';

interface Props {
  botId: string;
  mechanism: Mechanism;
  botName?: string;
}

const props = defineProps<Props>();

const message = ref('');
const sessionId = ref<string | null>(null);
const traceResult = ref<DebugTraceResult | null>(null);
const loading = ref(false);

const events = computed<WorkflowEvent[]>(
  () => (props.mechanism?.reply?.workflow ?? props.mechanism?.reply?.special_mode)?.events || []
);

async function handleRun() {
  if (!message.value.trim() || loading.value) return;
  loading.value = true;

  try {
    const result = await api.debugBot(props.botId, {
      message: message.value.trim(),
      step_mode: false,
      session_id: sessionId.value || undefined,
      sender_name: props.botName || '调试者',
      workflow_config: props.mechanism.reply?.workflow ?? props.mechanism.reply?.special_mode,
    });

    if (result.success && result.data) {
      traceResult.value = result.data;
      sessionId.value = result.data.session_id;
    }
  } catch {
    // 调试失败静默处理
  } finally {
    loading.value = false;
  }
}

async function handleStep() {
  if (!sessionId.value || loading.value) return;
  loading.value = true;

  try {
    const result = await api.debugStep(props.botId, {
      session_id: sessionId.value,
    });

    if (result.success && result.data) {
      traceResult.value = result.data;
      sessionId.value = result.data.session_id;
    }
  } catch {
    // 调试失败静默处理
  } finally {
    loading.value = false;
  }
}

async function handleReset() {
  if (!sessionId.value) return;

  try {
    await api.debugReset(props.botId, {
      session_id: sessionId.value,
    });
  } catch {
    // 忽略
  }

  sessionId.value = null;
  traceResult.value = null;
}
</script>

<style scoped>
.debug-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.debug-panel__input-row {
  display: flex;
  gap: 6px;
  align-items: center;
}

.debug-panel__input {
  flex: 1;
  padding: 7px 12px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-color, #1c1917);
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.debug-panel__input:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.debug-panel__input:disabled {
  opacity: 0.6;
}

.debug-panel__btn {
  padding: 7px 14px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--surface-tertiary-color, #e8e4de);
  color: var(--text-secondary-color, #57534e);
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.debug-panel__btn:hover:not(:disabled) {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.debug-panel__btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.debug-panel__btn--primary {
  background: var(--theme-primary, #5a8f4e);
  color: white;
  border-color: transparent;
}

.debug-panel__btn--primary:hover:not(:disabled) {
  opacity: 0.9;
}

.debug-panel__btn--ghost {
  background: transparent;
  border-color: transparent;
}

.debug-panel__result {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.debug-panel__output-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.debug-panel__reply {
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--surface-secondary-color, #f4f1ec);
  padding: 12px;
  overflow: hidden;
}

.debug-panel__reply-title {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.debug-panel__reply-content {
  font-size: 13px;
  color: var(--text-color, #1c1917);
  line-height: 1.5;
  word-break: break-word;
  margin: 0;
}

.debug-panel__empty {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
