<template>
  <div class="debug-panel">
    <!-- 工具栏 -->
    <div class="debug-toolbar">
      <button
        class="debug-toolbar__btn debug-toolbar__btn--primary"
        :disabled="isRunning || !inputMessage.trim()"
        @click="handleRunAll"
      >
        {{ isRunning ? '执行中...' : '全部运行' }}
      </button>
      <button
        class="debug-toolbar__btn"
        :class="{ 'debug-toolbar__btn--active': stepMode }"
        :disabled="isRunning || !inputMessage.trim()"
        @click="handleStepMode"
      >
        逐步执行
      </button>
      <button
        v-if="waitingForStep"
        class="debug-toolbar__btn debug-toolbar__btn--accent"
        :disabled="isRunning"
        @click="handleNextStep"
      >
        下一步
      </button>
      <div class="debug-toolbar__spacer" />
      <button
        class="debug-toolbar__btn debug-toolbar__btn--danger"
        :disabled="!sessionId"
        @click="handleReset"
      >
        重置会话
      </button>
    </div>

    <!-- 消息输入 -->
    <div class="debug-input">
      <input
        v-model="inputMessage"
        type="text"
        class="debug-input__field"
        placeholder="输入模拟消息..."
        :disabled="isRunning"
        @keydown.enter="handleQuickSend"
      />
      <div class="debug-input__sender">
        <input
          v-model="senderName"
          type="text"
          class="debug-input__sender-field"
          placeholder="发送者"
        />
        <button
          class="debug-input__send"
          :disabled="isRunning || !inputMessage.trim()"
          @click="stepMode ? handleStepMode() : handleRunAll()"
        >
          发送
        </button>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="debug-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="debug-tabs__btn"
        :class="{ 'debug-tabs__btn--active': activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
        <span v-if="tab.count !== undefined" class="debug-tabs__count">{{ tab.count }}</span>
      </button>
    </div>

    <!-- Tab 内容 -->
    <div class="debug-content">
      <DebugEventFlow v-if="activeTab === 'flow'" :events="events" :event-traces="eventTraces" />
      <DebugContextViewer
        v-else-if="activeTab === 'context'"
        :messages="contextMessages"
        :round="round"
      />
      <DebugOutputViewer v-else-if="activeTab === 'output'" :traces="eventTraces" />

      <!-- 模拟对话记录 -->
      <div v-if="messages.length > 0" class="debug-messages">
        <div class="debug-messages__divider">对话记录</div>
        <div
          v-for="(msg, i) in messages"
          :key="i"
          class="debug-messages__item"
          :class="`debug-messages__item--${msg.role}`"
        >
          <span class="debug-messages__sender">{{ msg.sender }}</span>
          <span class="debug-messages__text">{{ msg.content }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { api } from '../../../../models/api';
import type {
  Mechanism,
  SpecialModeEvent,
  EventTrace,
  DebugContextMessage,
  DebugTraceResult,
} from '../../../../models/types';
import DebugEventFlow from './debug/DebugEventFlow.vue';
import DebugContextViewer from './debug/DebugContextViewer.vue';
import DebugOutputViewer from './debug/DebugOutputViewer.vue';

interface Props {
  botId: string;
  mechanism: Mechanism;
  botName: string;
}

const props = defineProps<Props>();

// 状态
const inputMessage = ref('');
const senderName = ref('调试用户');
const sessionId = ref<string | null>(null);
const stepMode = ref(false);
const isRunning = ref(false);
const waitingForStep = ref(false);
const activeTab = ref<'flow' | 'context' | 'output'>('flow');
const round = ref(0);

const eventTraces = ref<EventTrace[]>([]);
const contextMessages = ref<DebugContextMessage[]>([]);
const messages = ref<{ role: 'user' | 'assistant'; sender: string; content: string }[]>([]);

const events = computed<SpecialModeEvent[]>(() => {
  return props.mechanism.reply?.special_mode?.events || [];
});

const tabs = computed(() => [
  { id: 'flow' as const, label: '事件流', count: eventTraces.value.length },
  { id: 'context' as const, label: '上下文', count: contextMessages.value.length },
  {
    id: 'output' as const,
    label: '输出',
    count: eventTraces.value.filter((t) => t.status !== 'pending').length,
  },
]);

async function handleRunAll() {
  if (!inputMessage.value.trim()) return;

  isRunning.value = true;
  stepMode.value = false;
  waitingForStep.value = false;

  const message = inputMessage.value;
  inputMessage.value = '';
  messages.value.push({ role: 'user', sender: senderName.value, content: message });
  activeTab.value = 'flow';

  try {
    const result = await api.debugBot(props.botId, {
      message,
      step_mode: false,
      session_id: sessionId.value || undefined,
      sender_name: senderName.value,
      special_mode_config: props.mechanism.reply?.special_mode as any,
    });

    if (result.success && result.data) {
      applyResult(result.data, message);
    }
  } catch (err: any) {
    const errorMsg = err.response?.data?.message || '调试执行失败';
    messages.value.push({ role: 'assistant', sender: '系统', content: `[错误] ${errorMsg}` });
  } finally {
    isRunning.value = false;
  }
}

async function handleStepMode() {
  if (!inputMessage.value.trim()) return;

  isRunning.value = true;
  stepMode.value = true;

  const message = inputMessage.value;
  inputMessage.value = '';
  messages.value.push({ role: 'user', sender: senderName.value, content: message });
  activeTab.value = 'flow';

  try {
    const result = await api.debugBot(props.botId, {
      message,
      step_mode: true,
      session_id: sessionId.value || undefined,
      sender_name: senderName.value,
      special_mode_config: props.mechanism.reply?.special_mode as any,
    });

    if (result.success && result.data) {
      applyResult(result.data, message);
    }
  } catch (err: any) {
    const errorMsg = err.response?.data?.message || '调试执行失败';
    messages.value.push({ role: 'assistant', sender: '系统', content: `[错误] ${errorMsg}` });
  } finally {
    isRunning.value = false;
  }
}

async function handleNextStep() {
  if (!sessionId.value) return;
  isRunning.value = true;

  try {
    const result = await api.debugStep(props.botId, {
      session_id: sessionId.value,
    });

    if (result.success && result.data) {
      applyResult(result.data);
    }
  } catch (err: any) {
    const errorMsg = err.response?.data?.message || '逐步执行失败';
    messages.value.push({ role: 'assistant', sender: '系统', content: `[错误] ${errorMsg}` });
  } finally {
    isRunning.value = false;
  }
}

function applyResult(data: DebugTraceResult, _userMessage?: string) {
  sessionId.value = data.session_id;
  round.value = data.round;
  eventTraces.value = data.event_traces;
  contextMessages.value = data.context_messages;
  waitingForStep.value = data.waiting_for_step;

  if (data.reply) {
    messages.value.push({ role: 'assistant', sender: props.botName, content: data.reply });
  }
}

async function handleReset() {
  if (!sessionId.value) return;

  try {
    await api.debugReset(props.botId, { session_id: sessionId.value });
  } catch {
    // 静默处理
  }

  sessionId.value = null;
  stepMode.value = false;
  waitingForStep.value = false;
  round.value = 0;
  eventTraces.value = [];
  contextMessages.value = [];
  messages.value = [];
}

function handleQuickSend() {
  if (stepMode.value) {
    handleStepMode();
  } else {
    handleRunAll();
  }
}
</script>

<style scoped>
.debug-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  border-radius: var(--radius-md, 12px);
  padding: 12px;
  background: var(--bg-tertiary, #f0efed);
}

/* 工具栏 */
.debug-toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
}

.debug-toolbar__spacer {
  flex: 1;
}

.debug-toolbar__btn {
  padding: 5px 12px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--strong-background-color, #fff);
  color: var(--text-secondary, #666);
  cursor: pointer;
  transition: all 0.15s;
}

.debug-toolbar__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.debug-toolbar__btn:hover:not(:disabled) {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.debug-toolbar__btn--active {
  border-color: var(--theme-primary, #5a8f4e);
  background: rgba(90, 143, 78, 0.08);
  color: var(--theme-primary, #5a8f4e);
}

.debug-toolbar__btn--primary {
  background: var(--theme-primary, #5a8f4e);
  color: white;
  border-color: transparent;
}

.debug-toolbar__btn--primary:hover:not(:disabled) {
  opacity: 0.9;
  color: white;
}

.debug-toolbar__btn--accent {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.debug-toolbar__btn--danger {
  color: var(--text-tertiary, #999);
  border-color: transparent;
}

.debug-toolbar__btn--danger:hover:not(:disabled) {
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.3);
}

/* 输入区 */
.debug-input {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.debug-input__field {
  width: 100%;
  padding: 8px 12px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--strong-background-color, #fff);
  color: var(--text-primary, #1a1a1a);
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s;
}

.debug-input__field:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.debug-input__field:disabled {
  opacity: 0.5;
}

.debug-input__sender {
  display: flex;
  gap: 6px;
}

.debug-input__sender-field {
  width: 100px;
  padding: 5px 8px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--strong-background-color, #fff);
  color: var(--text-secondary, #666);
  outline: none;
}

.debug-input__sender-field:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.debug-input__send {
  padding: 5px 14px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: none;
  background: var(--theme-primary, #5a8f4e);
  color: white;
  cursor: pointer;
  transition: opacity 0.15s;
}

.debug-input__send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* Tab */
.debug-tabs {
  display: flex;
  gap: 2px;
  border-bottom: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  padding-bottom: 0;
}

.debug-tabs__btn {
  padding: 6px 12px;
  font-size: 12px;
  border: none;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--text-tertiary, #999);
  cursor: pointer;
  transition: all 0.15s;
}

.debug-tabs__btn:hover {
  color: var(--text-secondary, #666);
}

.debug-tabs__btn--active {
  color: var(--text-primary, #1a1a1a);
  border-bottom-color: var(--theme-primary, #5a8f4e);
}

.debug-tabs__count {
  font-size: 10px;
  margin-left: 4px;
  padding: 0 5px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.06);
  color: var(--text-tertiary, #999);
}

.debug-tabs__btn--active .debug-tabs__count {
  background: rgba(90, 143, 78, 0.1);
  color: var(--theme-primary, #5a8f4e);
}

/* 内容区 */
.debug-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* 对话记录 */
.debug-messages {
  margin-top: 4px;
}

.debug-messages__divider {
  font-size: 11px;
  color: var(--text-tertiary, #999);
  padding: 4px 0;
  border-bottom: 1px dashed var(--border-subtle, rgba(0, 0, 0, 0.06));
  margin-bottom: 6px;
}

.debug-messages__item {
  display: flex;
  gap: 8px;
  padding: 4px 0;
  font-size: 12px;
  line-height: 1.5;
}

.debug-messages__sender {
  flex-shrink: 0;
  font-weight: 500;
  color: var(--text-secondary, #666);
  min-width: 48px;
}

.debug-messages__item--user .debug-messages__sender {
  color: var(--theme-primary, #5a8f4e);
}

.debug-messages__text {
  color: var(--text-primary, #1a1a1a);
  word-break: break-word;
}
</style>
