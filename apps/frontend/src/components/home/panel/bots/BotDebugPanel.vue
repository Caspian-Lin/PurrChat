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
        v-if="trace?.waiting_for_step"
        class="debug-toolbar__btn debug-toolbar__btn--accent"
        :disabled="isRunning"
        @click="handleNextStep"
      >
        下一步
      </button>
      <div class="debug-toolbar__spacer" />
      <!-- 副作用策略 -->
      <select v-model="sideEffectPolicy" class="debug-toolbar__select" :disabled="isRunning">
        <option value="mock">Mock 副作用</option>
        <option value="sandbox">Sandbox (真实调用)</option>
      </select>
      <button
        class="debug-toolbar__btn debug-toolbar__btn--danger"
        :disabled="!sessionId"
        @click="handleReset"
      >
        重置
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
        @keydown.enter="stepMode ? handleStepMode() : handleRunAll()"
      />
    </div>

    <!-- 节点 trace 流 -->
    <div v-if="trace" class="debug-trace">
      <!-- 状态摘要 -->
      <div class="debug-trace__summary">
        <span class="debug-trace__status" :class="`debug-trace__status--${trace.status}`">
          {{ statusLabel(trace.status) }}
        </span>
        <span v-if="trace.durationMs !== undefined" class="debug-trace__meta">
          {{ trace.durationMs }}ms
        </span>
        <span v-if="trace.reply" class="debug-trace__meta debug-trace__reply-preview">
          → {{ trace.reply.slice(0, 60) }}{{ trace.reply.length > 60 ? '...' : '' }}
        </span>
      </div>

      <!-- 节点列表 -->
      <div class="debug-trace__nodes">
        <div
          v-for="node in trace.nodes"
          :key="node.nodeId"
          class="trace-node"
          :class="`trace-node--${node.status}`"
        >
          <div class="trace-node__header" @click="toggleNode(node.nodeId)">
            <span class="trace-node__icon">{{ statusIcon(node.status) }}</span>
            <span class="trace-node__type">{{ node.nodeType }}</span>
            <span class="trace-node__name">{{ node.nodeName || node.nodeId }}</span>
            <span v-if="node.branch" class="trace-node__branch"> branch: {{ node.branch }} </span>
            <span v-if="node.durationMs !== undefined" class="trace-node__duration">
              {{ node.durationMs }}ms
            </span>
            <span v-if="node.error" class="trace-node__error-icon">⚠</span>
          </div>

          <!-- 展开详情 -->
          <div v-if="expandedNodes.has(node.nodeId)" class="trace-node__detail">
            <!-- 错误信息 -->
            <div v-if="node.error" class="trace-node__error-detail">
              <strong>错误:</strong> {{ node.error }}
            </div>

            <!-- 输入端口 -->
            <div
              v-if="node.input && Object.keys(node.input).length > 0"
              class="trace-node__section"
            >
              <span class="trace-node__section-label">输入</span>
              <div class="trace-node__ports">
                <div v-for="(val, key) in node.input" :key="key" class="trace-node__port">
                  <code class="trace-node__port-key">{{ key }}</code>
                  <span class="trace-node__port-val">{{ truncate(val) }}</span>
                </div>
              </div>
            </div>

            <!-- 输出端口 -->
            <div
              v-if="node.output && Object.keys(node.output).length > 0"
              class="trace-node__section"
            >
              <span class="trace-node__section-label">输出</span>
              <div class="trace-node__ports">
                <div v-for="(val, key) in node.output" :key="key" class="trace-node__port">
                  <code class="trace-node__port-key">{{ key }}</code>
                  <span class="trace-node__port-val">{{ truncate(val) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Bot 回复 -->
      <div v-if="trace.reply" class="debug-trace__reply">
        <h4 class="debug-trace__reply-title">Bot 回复</h4>
        <p class="debug-trace__reply-content">{{ trace.reply }}</p>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="debug-panel__empty">
      <p>输入消息并点击"全部运行"开始调试</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { api } from '../../../../models/api';
import {
  createEmptyDocument,
  type WorkflowEvent,
  type WorkflowEndCondition,
  type FlowConnection,
  type RunTrace,
  type NodeTrace,
  type NodeTraceStatus,
  type RunTraceStatus,
} from '@purrchat/workflow-types';

interface Props {
  botId: string;
  events: WorkflowEvent[];
  connections: FlowConnection[];
  endConditions: WorkflowEndCondition[];
  botName: string;
}

const props = defineProps<Props>();

const inputMessage = ref('');
const sessionId = ref<string | null>(null);
const stepMode = ref(false);
const isRunning = ref(false);
const sideEffectPolicy = ref<'mock' | 'sandbox'>('mock');
const trace = ref<RunTrace | null>(null);
const expandedNodes = ref<Set<string>>(new Set());

/** 从编辑器状态构建 WorkflowDocument */
function buildDocument(): unknown {
  const doc = createEmptyDocument('debug');
  doc.spec.nodes = props.events.map((e, i) => ({
    id: e.id,
    type: e.type,
    name: e.name,
    key: (e as any).key ?? `${e.type}_${i}`,
    config: e.config ?? {},
    ports: e.ports,
    position: e.position,
  }));
  doc.spec.connections = props.connections.map((c, i) => ({
    id: c.id ?? `conn_${i}`,
    sourceNodeId: c.sourceNodeId,
    sourcePortId: c.sourcePortId,
    targetNodeId: c.targetNodeId,
    targetPortId: c.targetPortId,
  }));
  doc.spec.endConditions = props.endConditions;
  return doc;
}

async function handleRunAll() {
  if (!inputMessage.value.trim()) return;
  isRunning.value = true;
  stepMode.value = false;

  try {
    const result = await api.testRunWorkflow(props.botId, {
      message: inputMessage.value.trim(),
      document: buildDocument(),
      side_effects: sideEffectPolicy.value,
      step_mode: false,
      sender_name: '调试用户',
      session_id: sessionId.value || undefined,
    });

    if (result.success && result.data) {
      trace.value = result.data as unknown as RunTrace;
      sessionId.value = trace.value.session_id ?? trace.value.runId;
    }
  } catch (err: any) {
    const errorMsg = err.response?.data?.error || err.response?.data?.message || '调试执行失败';
    trace.value = {
      runId: 'error',
      status: 'error',
      nodes: [],
      startedAt: Date.now(),
      input: inputMessage.value,
      reply: `[错误] ${errorMsg}`,
    };
  } finally {
    isRunning.value = false;
  }
}

async function handleStepMode() {
  if (!inputMessage.value.trim()) return;
  isRunning.value = true;
  stepMode.value = true;

  try {
    const result = await api.testRunWorkflow(props.botId, {
      message: inputMessage.value.trim(),
      document: buildDocument(),
      side_effects: sideEffectPolicy.value,
      step_mode: true,
      sender_name: '调试用户',
      session_id: sessionId.value || undefined,
    });

    if (result.success && result.data) {
      trace.value = result.data as unknown as RunTrace;
      sessionId.value = trace.value.session_id ?? trace.value.runId;
    }
  } catch (err: any) {
    const errorMsg = err.response?.data?.error || '调试执行失败';
    trace.value = {
      runId: 'error',
      status: 'error',
      nodes: [],
      startedAt: Date.now(),
      input: inputMessage.value,
      reply: `[错误] ${errorMsg}`,
    };
  } finally {
    isRunning.value = false;
  }
}

async function handleNextStep() {
  if (!sessionId.value) return;
  isRunning.value = true;

  try {
    const result = await api.testRunStep(props.botId, sessionId.value);
    if (result.success && result.data) {
      trace.value = result.data as unknown as RunTrace;
    }
  } catch (err: any) {
    const errorMsg = err.response?.data?.error || '逐步执行失败';
    if (trace.value) {
      trace.value.status = 'error';
      trace.value.reply = `[错误] ${errorMsg}`;
    }
  } finally {
    isRunning.value = false;
  }
}

async function handleReset() {
  if (!sessionId.value) return;
  try {
    await api.testRunWorkflow(props.botId, {
      message: '',
      document: buildDocument(),
    });
  } catch {
    // reset 失败静默处理
  }
  sessionId.value = null;
  stepMode.value = false;
  trace.value = null;
  expandedNodes.value.clear();
}

function toggleNode(nodeId: string) {
  if (expandedNodes.value.has(nodeId)) {
    expandedNodes.value.delete(nodeId);
  } else {
    expandedNodes.value.add(nodeId);
  }
}

function statusIcon(status: NodeTraceStatus): string {
  switch (status) {
    case 'success':
      return '✓';
    case 'error':
      return '✗';
    case 'skip':
      return '○';
    case 'running':
      return '◐';
    default:
      return '○';
  }
}

function statusLabel(status: RunTraceStatus): string {
  switch (status) {
    case 'completed':
      return '完成';
    case 'error':
      return '出错';
    case 'cancelled':
      return '已取消';
    case 'running':
      return '执行中';
    default:
      return status;
  }
}

function truncate(val: string): string {
  if (!val) return '';
  return val.length > 200 ? val.slice(0, 200) + '...' : val;
}
</script>

<style scoped>
.debug-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
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
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--strong-background-color, #fff);
  color: var(--text-secondary-color, #57534e);
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
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
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
  background: var(--color-info-bg, rgba(37, 99, 235, 0.1));
  color: var(--color-info, #2563eb);
  border-color: color-mix(in srgb, var(--color-info, #2563eb) 30%, transparent);
}

.debug-toolbar__btn--danger {
  color: var(--text-tertiary-color, #a8a29e);
  border-color: transparent;
}

.debug-toolbar__btn--danger:hover:not(:disabled) {
  color: var(--color-error, #dc2626);
}

.debug-toolbar__select {
  padding: 4px 8px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-secondary-color, #57534e);
  outline: none;
}

/* 输入区 */
.debug-input__field {
  width: 100%;
  padding: 8px 12px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-color, #1c1917);
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

/* Trace 区域 */
.debug-trace {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.debug-trace__summary {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  padding: 4px 0;
}

.debug-trace__status {
  font-weight: 500;
  padding: 2px 8px;
  border-radius: var(--radius-xs, 4px);
}

.debug-trace__status--completed {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 12%, transparent);
  color: var(--theme-primary, #5a8f4e);
}

.debug-trace__status--error {
  background: color-mix(in srgb, var(--color-error, #dc2626) 12%, transparent);
  color: var(--color-error, #dc2626);
}

.debug-trace__status--cancelled {
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.06));
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-trace__status--running {
  background: color-mix(in srgb, var(--color-info, #2563eb) 12%, transparent);
  color: var(--color-info, #2563eb);
}

.debug-trace__meta {
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-trace__reply-preview {
  color: var(--text-secondary-color, #57534e);
}

/* 节点列表 */
.debug-trace__nodes {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.trace-node {
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  overflow: hidden;
}

.trace-node--skip {
  opacity: 0.45;
}

.trace-node--error {
  border-color: color-mix(in srgb, var(--color-error, #dc2626) 30%, transparent);
}

.trace-node__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 12px;
  cursor: pointer;
  background: var(--surface-secondary-color, rgba(0, 0, 0, 0.02));
  transition: background 0.1s;
}

.trace-node__header:hover {
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.04));
}

.trace-node__icon {
  width: 16px;
  text-align: center;
  flex-shrink: 0;
}

.trace-node--success .trace-node__icon {
  color: var(--theme-primary, #5a8f4e);
}

.trace-node--error .trace-node__icon {
  color: var(--color-error, #dc2626);
}

.trace-node--skip .trace-node__icon {
  color: var(--text-tertiary-color, #a8a29e);
}

.trace-node__type {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  flex-shrink: 0;
}

.trace-node__name {
  color: var(--text-color, #1c1917);
  font-weight: 500;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trace-node__branch {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 10px;
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.06));
  color: var(--text-secondary-color, #57534e);
}

.trace-node__duration {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  flex-shrink: 0;
}

.trace-node__error-icon {
  color: var(--color-error, #dc2626);
}

/* 节点详情 */
.trace-node__detail {
  padding: 8px 10px;
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  background: var(--surface-color, #faf9f7);
}

.trace-node__error-detail {
  font-size: 12px;
  color: var(--color-error, #dc2626);
  padding: 4px 0;
  margin-bottom: 6px;
}

.trace-node__section {
  margin-top: 4px;
}

.trace-node__section-label {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-tertiary-color, #a8a29e);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.trace-node__ports {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 2px;
}

.trace-node__port {
  display: flex;
  gap: 6px;
  font-size: 11px;
  align-items: baseline;
}

.trace-node__port-key {
  font-family: monospace;
  color: var(--theme-primary, #5a8f4e);
  flex-shrink: 0;
}

.trace-node__port-val {
  color: var(--text-secondary-color, #57534e);
  word-break: break-word;
}

/* 回复 */
.debug-trace__reply {
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--surface-secondary-color, #f4f1ec);
  padding: 10px;
}

.debug-trace__reply-title {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.debug-trace__reply-content {
  font-size: 13px;
  color: var(--text-color, #1c1917);
  line-height: 1.5;
  word-break: break-word;
  margin: 0;
}

/* 空状态 */
.debug-panel__empty {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
