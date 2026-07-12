<template>
  <!-- 移动端：显示桌面端提示 -->
  <div v-if="isMobile" class="mobile-workflow-notice">
    <div class="mobile-notice-content">
      <div class="mobile-notice-icon">
        <BsPcDisplay :size="48" />
      </div>
      <h2 class="mobile-notice-title">请在桌面端使用</h2>
      <p class="mobile-notice-desc">
        工作流编辑器需要鼠标拖拽和精确连线操作，建议在桌面浏览器中使用。
      </p>
      <button class="mobile-notice-btn" @click="goBack">
        <BsArrowLeft :size="16" />
        返回
      </button>
    </div>
  </div>

  <!-- 桌面端：正常编辑器 -->
  <div v-else class="relative flex flex-col h-screen bg-bg-primary">
    <!-- 顶部工具栏 -->
    <div
      class="flex items-center gap-3 px-5 py-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
    >
      <button
        class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
        title="返回"
        @click="goBack"
      >
        <BsArrowLeft :size="18" />
      </button>
      <div class="flex-1 min-w-0">
        <h2 class="text-sm font-medium text-text-primary truncate">
          {{ botName }}
          <span class="text-text-tertiary mx-1.5">/</span>
          {{ mechanismName }}
        </h2>
        <p class="text-xs text-text-tertiary truncate">工作流事件链编辑器</p>
      </div>
      <div class="workflow-status" aria-live="polite">
        <span>草稿 r{{ revision }}</span>
        <span>已发布 {{ publishedRevision === null ? '无' : `r${publishedRevision}` }}</span>
        <span v-if="dirty" class="workflow-status__dirty">未保存</span>
      </div>
      <span v-if="saveState === 'saving'" class="text-xs text-text-tertiary">保存中...</span>
      <span v-else-if="saveState === 'publishing'" class="text-xs text-text-tertiary"
        >发布中...</span
      >
      <span v-else-if="saveState === 'saved'" class="text-xs text-green-600">已保存</span>
      <span v-else-if="saveState === 'error'" class="text-xs text-red-500">保存失败</span>
      <button
        class="toolbar-btn"
        :disabled="saveState === 'saving' || saveState === 'publishing'"
        @click="toggleHistory"
      >
        版本历史
      </button>
      <button
        class="px-4 py-1.5 text-xs rounded-[var(--radius-sm,8px)] text-white transition-colors"
        style="background: var(--theme-primary)"
        :disabled="saveState === 'saving' || saveState === 'publishing'"
        @click="handleSave"
      >
        保存
      </button>
      <button
        class="px-4 py-1.5 text-xs rounded-[var(--radius-sm,8px)] border border-border-subtle text-text-primary bg-bg-quaternary transition-colors hover:border-[var(--theme-primary)]"
        :disabled="saveState === 'saving' || saveState === 'publishing'"
        @click="handlePublish"
      >
        发布
      </button>
    </div>

    <div v-if="operationError" class="workflow-error" role="alert">{{ operationError }}</div>

    <section v-if="showHistory" class="version-popover" aria-label="工作流版本历史">
      <div class="version-popover__header">
        <strong>版本历史</strong>
        <button class="version-popover__close" @click="showHistory = false">关闭</button>
      </div>
      <p class="version-popover__hint">恢复操作只创建新草稿，不会自动发布。</p>
      <p v-if="historyLoading" class="version-popover__empty">加载中...</p>
      <p v-else-if="versions.length === 0" class="version-popover__empty">尚无已发布版本</p>
      <div v-else class="version-list">
        <div v-for="version in versions" :key="version.id" class="version-item">
          <div>
            <strong>r{{ version.revision }}</strong>
            <span>{{ formatPublishedAt(version.published_at) }}</span>
          </div>
          <button :disabled="historyLoading" @click="restoreVersion(version.revision)">
            恢复为草稿
          </button>
        </div>
      </div>
    </section>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <p class="text-sm text-text-tertiary">加载中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="flex-1 flex items-center justify-center">
      <div class="text-center">
        <p class="text-sm text-red-500 mb-2">{{ error }}</p>
        <button
          class="text-xs text-text-secondary hover:text-text-primary underline"
          @click="loadData"
        >
          重试
        </button>
      </div>
    </div>

    <!-- 编辑器主体 -->
    <template v-else>
      <!-- 空状态 -->
      <div v-if="events.length === 0" class="flex-1 flex items-center justify-center">
        <div class="text-center">
          <p class="text-sm text-text-tertiary mb-1">尚未配置事件链</p>
          <p class="text-xs text-text-quaternary mb-4">添加事件来构建 Agent 的工作流程</p>
          <button
            class="inline-flex items-center gap-1.5 px-4 py-2 text-xs text-text-secondary rounded-[var(--radius-sm,8px)] border border-border-subtle bg-bg-quaternary hover:border-[var(--theme-primary)] hover:text-[var(--theme-primary)] transition-colors"
            @click="showAddModal = true"
          >
            <BsPlus :size="14" />
            添加第一个事件
          </button>
        </div>
      </div>

      <!-- VueFlow DAG 编辑器 -->
      <div v-else class="flex-1 relative overflow-hidden">
        <!-- 连线提示 toast -->
        <Transition name="toast-fade">
          <div
            v-if="connectionToast.visible"
            class="connection-toast"
            :class="`connection-toast--${connectionToast.type}`"
          >
            {{ connectionToast.message }}
          </div>
        </Transition>
        <div class="editor-toolbar absolute top-3 left-3 z-10 flex gap-2">
          <button class="toolbar-btn" @click="showAddModal = true">
            <BsPlus :size="14" />
            添加事件
          </button>
          <button class="toolbar-btn" @click="handleAutoLayout">自动布局</button>
          <button
            class="toolbar-btn toolbar-btn--subtle"
            title="导入 YAML 流程"
            @click="handleYamlImport"
          >
            <BsUpload :size="13" />
            YAML
          </button>
          <button
            class="toolbar-btn toolbar-btn--subtle"
            title="导出 YAML 流程"
            @click="handleYamlExport"
          >
            <BsDownload :size="13" />
            YAML
          </button>
          <span v-if="validationIssues.length" class="toolbar-validation-badge">
            {{ validationIssues.filter((i) => i.level === 'error').length }} 个问题
          </span>
        </div>

        <!-- 验证警告列表 -->
        <div v-if="validationIssues.length" class="editor-validation absolute top-3 right-3 z-10">
          <div
            v-for="(issue, idx) in validationIssues"
            :key="idx"
            class="editor-validation__item"
            :class="`editor-validation__item--${issue.level}`"
          >
            {{ issue.level === 'error' ? '×' : '!' }} {{ issue.message }}
          </div>
        </div>

        <VueFlow
          :nodes="flowNodes"
          :edges="flowEdges"
          :node-types="customNodeTypes"
          :edge-types="customEdgeTypes"
          :default-edge-options="defaultEdgeOptions"
          :is-valid-connection="isValidConnection"
          :connection-mode="ConnectionMode.Loose"
          fit-view-on-init
          :min-zoom="0.3"
          :max-zoom="2"
          class="w-full h-full"
          @node-click="onNodeClick"
          @connect="onConnect"
          @edges-change="onEdgesChange"
          @nodes-change="onNodesChange"
        >
          <Background :gap="20" :size="1" />
          <Controls />
        </VueFlow>
      </div>

      <!-- 底部面板：指南 + 结束条件 + 调试 -->
      <div class="flex-shrink-0 border-t border-border-subtle bg-bg-secondary">
        <!-- Tab 切换 -->
        <div class="flex border-b border-border-subtle">
          <button
            class="bottom-tab"
            :class="{ 'bottom-tab--active': activeBottomTab === 'guide' }"
            @click="activeBottomTab = 'guide'"
          >
            指南
          </button>
          <button
            class="bottom-tab"
            :class="{ 'bottom-tab--active': activeBottomTab === 'conditions' }"
            @click="activeBottomTab = 'conditions'"
          >
            结束条件
          </button>
          <button
            class="bottom-tab"
            :class="{ 'bottom-tab--active': activeBottomTab === 'debug' }"
            @click="activeBottomTab = 'debug'"
          >
            调试
          </button>
        </div>

        <!-- 指南面板 -->
        <div v-if="activeBottomTab === 'guide'" class="guide-panel">
          <div class="guide-section">
            <h4 class="guide-section__title">模板变量</h4>
            <p class="guide-section__desc">
              触发节点会自动收集当前消息的上下文信息，作为输出端口供下游节点使用。
              将变量输出端口连接到下游节点的对应输入端口即可传递数据。
            </p>
            <div class="guide-vars">
              <div class="guide-var">
                <code class="guide-var__code">{input}</code>
                <span class="guide-var__label">用户消息</span>
                <span class="guide-var__desc">用户发送的完整消息内容</span>
              </div>
              <div class="guide-var">
                <code class="guide-var__code">{username}</code>
                <span class="guide-var__label">发送者</span>
                <span class="guide-var__desc">发送消息的用户名</span>
              </div>
              <div class="guide-var">
                <code class="guide-var__code">{time}</code>
                <span class="guide-var__label">时间</span>
                <span class="guide-var__desc">当前时间，格式 HH:MM</span>
              </div>
              <div class="guide-var">
                <code class="guide-var__code">{args}</code>
                <span class="guide-var__label">参数</span>
                <span class="guide-var__desc">消息中除首个词外的所有参数</span>
              </div>
              <div class="guide-var">
                <code class="guide-var__code">{args:N}</code>
                <span class="guide-var__label">第 N 个词</span>
                <span class="guide-var__desc">按空格分割后的第 N 个词（从 0 开始）</span>
              </div>
            </div>
            <div class="guide-example">
              <span class="guide-example__prompt">用户发送：</span>
              <code>天气 北京</code>
              <div class="guide-example__results">
                <span><code>{input}</code> → "天气 北京"</span>
                <span><code>{args}</code> → "北京"</span>
                <span><code>{args:0}</code> → "天气"</span>
                <span><code>{args:1}</code> → "北京"</span>
              </div>
            </div>
          </div>

          <div class="guide-section">
            <h4 class="guide-section__title">连线规则</h4>
            <p class="guide-section__desc">
              端口按数据类型严格匹配：<span
                class="guide-type"
                style="color: var(--color-success, #16a34a)"
                >▶ 执行流</span
              >（绿色）只能连接执行流端口；
              <span class="guide-type" style="color: #3b82f6">数据</span
              >（蓝色）可连接任意数据输入端口。 拖拽连线时，不兼容的端口会自动拒绝。
            </p>
          </div>

          <div class="guide-section">
            <h4 class="guide-section__title">条件分支</h4>
            <p class="guide-section__desc">
              <strong>条件节点</strong
              >（◇）根据条件结果选择真或假分支。未通过生产验证的节点不会出现在添加列表中。
            </p>
          </div>
        </div>

        <!-- 结束条件面板 -->
        <div v-if="activeBottomTab === 'conditions'" class="p-4">
          <EndConditionConfig :conditions="endConditions" @update="handleEndConditionsUpdate" />
        </div>

        <!-- 调试面板 -->
        <div v-if="activeBottomTab === 'debug'" class="p-4">
          <BotDebugPanel
            v-if="events.length > 0"
            :bot-id="botId"
            :events="events"
            :connections="connections"
            :end-conditions="endConditions"
            :bot-name="botName"
          />
          <p v-else class="text-xs text-text-quaternary">无法加载调试面板</p>
        </div>
      </div>
    </template>

    <!-- 事件配置弹窗 -->
    <EventConfigModal
      :visible="showAddModal || !!editingEvent"
      :editing-event="editingEvent"
      :existing-events="events"
      :connections="connections"
      @close="closeModal"
      @confirm="handleEventConfirm"
      @delete="handleEventDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, markRaw, reactive, provide } from 'vue';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import { VueFlow, ConnectionMode } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import { BsArrowLeft, BsPlus, BsUpload, BsDownload, BsPcDisplay } from 'vue-icons-plus/bs';
import { usePlatform } from '../composables/usePlatform';
import { useNotification } from '../composables/useNotification';
import { api } from '../models/api';
import type { Bot, Mechanism, WorkflowVersion } from '../models/types';
import type { Node, Edge } from '@vue-flow/core';
import { eventsToFlowNodes, eventsToFlowEdges, autoLayoutEvents } from '../utils/eventFlowUtils';
import { canConnect, getPortById, getDefaultPorts } from '../utils/portTypes';
import { ensurePorts } from '../utils/eventPorts';
import { documentToYaml, sanitizeForExport } from '@purrchat/workflow-engine';
import {
  isWorkflowDocument,
  migrateMechanismToDocument,
  type FlowConnection,
  type WorkflowDocument,
  type WorkflowEndCondition,
  type WorkflowEvent,
} from '@purrchat/workflow-types';
import { useWorkflowValidator } from '../composables/useWorkflowValidator';
import {
  cloneWorkflowDocument,
  cloneWorkflowEvent,
  evaluateWorkflowGate,
  nextUniqueNodeKey,
  parseWorkflowYamlCandidate,
  serializeWorkflowDocument,
} from '../utils/workflowDocument';
import TriggerNode from '../components/home/panel/bots/events/TriggerNode.vue';
import EndNode from '../components/home/panel/bots/events/EndNode.vue';
import WaitNode from '../components/home/panel/bots/events/WaitNode.vue';
import IfNode from '../components/home/panel/bots/events/IfNode.vue';
import ToolNode from '../components/home/panel/bots/events/ToolNode.vue';
import DifyNode from '../components/home/panel/bots/events/DifyNode.vue';
import N8nNode from '../components/home/panel/bots/events/N8nNode.vue';
import LlmNode from '../components/home/panel/bots/events/LlmNode.vue';
import BuiltinNode from '../components/home/panel/bots/events/BuiltinNode.vue';
import TemplateNode from '../components/home/panel/bots/events/TemplateNode.vue';
import ReplyNode from '../components/home/panel/bots/events/ReplyNode.vue';
import HistoryNode from '../components/home/panel/bots/events/HistoryNode.vue';
import EventEdge from '../components/home/panel/bots/events/EventEdge.vue';
import EventConfigModal from '../components/home/panel/bots/events/EventConfigModal.vue';
import EndConditionConfig from '../components/home/panel/bots/events/EndConditionConfig.vue';
import BotDebugPanel from '../components/home/panel/bots/BotDebugPanel.vue';

const route = useRoute();
const router = useRouter();
const { isMobile } = usePlatform();
const notify = useNotification();
const { validate } = useWorkflowValidator();

const botId = route.params.botId as string;
const mechanismId = route.params.mechanismId as string;

// 状态
const loading = ref(true);
const error = ref<string | null>(null);
const operationError = ref<string | null>(null);
const saveState = ref<'idle' | 'saving' | 'publishing' | 'saved' | 'error'>('idle');
const activeBottomTab = ref<'guide' | 'conditions' | 'debug'>('conditions');
const showAddModal = ref(false);
const showHistory = ref(false);
const historyLoading = ref(false);
const versions = ref<WorkflowVersion[]>([]);
const revision = ref(0);
const publishedRevision = ref<number | null>(null);
const baseline = ref('');
const workflowDocument = ref<WorkflowDocument | null>(null);

// 连线提示 toast
const connectionToast = reactive<{ visible: boolean; message: string; type: 'error' | 'warn' }>({
  visible: false,
  message: '',
  type: 'error',
});
let toastTimer: ReturnType<typeof setTimeout> | null = null;

function showConnectionToast(message: string, type: 'error' | 'warn' = 'error') {
  if (toastTimer) clearTimeout(toastTimer);
  connectionToast.visible = true;
  connectionToast.message = message;
  connectionToast.type = type;
  toastTimer = setTimeout(() => {
    connectionToast.visible = false;
  }, 3000);
}
const editingEvent = ref<WorkflowEvent | null>(null);

// 供子组件（EventEdge）直接删除连线，避免走 VueFlow 的 edges-change 回调
function removeConnection(connectionId: string) {
  if (!workflowDocument.value) return;
  const updated = connections.value.filter((c) => c.id !== connectionId);
  if (updated.length !== connections.value.length)
    workflowDocument.value.spec.connections = updated;
}
provide('removeWorkflowConnection', removeConnection);

// 数据
const bot = ref<Bot | null>(null);
const legacyMechanism = ref<Mechanism | null>(null);

const botName = computed(() => bot.value?.name || 'Bot');
const mechanismName = computed(() => legacyMechanism.value?.name || '工作流');

const events = computed<WorkflowEvent[]>(() => {
  const raw = workflowDocument.value?.spec.nodes || [];
  return ensurePorts(raw) as WorkflowEvent[];
});

const endConditions = computed<WorkflowEndCondition[]>(() => {
  return workflowDocument.value?.spec.endConditions || [];
});

const connections = computed<FlowConnection[]>(() => {
  return workflowDocument.value?.spec.connections || [];
});

const validationResult = computed(() => validate(workflowDocument.value));
const validationIssues = computed(() => validationResult.value.issues);
const dirty = computed(
  () =>
    !!workflowDocument.value && serializeWorkflowDocument(workflowDocument.value) !== baseline.value
);

// VueFlow 注册

const customNodeTypes: Record<string, any> = {
  trigger: markRaw(TriggerNode),
  end: markRaw(EndNode),
  wait: markRaw(WaitNode),
  if: markRaw(IfNode),
  loop: markRaw(BuiltinNode),
  switch: markRaw(BuiltinNode),
  merge: markRaw(BuiltinNode),
  tool: markRaw(ToolNode),
  dify: markRaw(DifyNode),
  n8n: markRaw(N8nNode),
  llm: markRaw(LlmNode),
  builtin: markRaw(BuiltinNode),
  template: markRaw(TemplateNode),
  reply: markRaw(ReplyNode),
  history: markRaw(HistoryNode),
};

const customEdgeTypes: Record<string, any> = {
  event: markRaw(EventEdge),
};

const defaultEdgeOptions = {
  type: 'event',
};

const positionTrigger = ref(0);

const flowNodes = computed<Node[]>(() => {
  positionTrigger.value;
  return eventsToFlowNodes(events.value, undefined, connections.value);
});

const flowEdges = computed<Edge[]>(() => {
  const conns = connections.value;
  return eventsToFlowEdges(events.value, conns);
});

// BroadcastChannel 用于主 Tab 同步
let channel: BroadcastChannel | null = null;
try {
  channel = new BroadcastChannel('purrchat-bot-editor');
} catch {
  // BroadcastChannel 不可用时忽略
}

onMounted(async () => {
  await loadData();
  channel?.addEventListener('message', (e) => {
    if (e.data.type === 'bot-updated' && e.data.botId === botId) {
      if (!dirty.value) loadData();
    }
  });
  window.addEventListener('beforeunload', handleBeforeUnload);
});

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload);
  channel?.close();
  if (toastTimer) clearTimeout(toastTimer);
});

onBeforeRouteLeave(() => !dirty.value || window.confirm('工作流有未保存修改，确定离开吗？'));

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return;
  event.preventDefault();
  event.returnValue = '';
}

async function loadData() {
  loading.value = true;
  error.value = null;

  try {
    const [botResult, workflowResult] = await Promise.all([
      api.getBot(botId),
      api.getWorkflow(botId).catch((requestError: any) => {
        if (requestError.response?.status === 404) return null;
        throw requestError;
      }),
    ]);
    if (!botResult.success || !botResult.data) {
      error.value = 'Bot 不存在或加载失败';
      return;
    }

    bot.value = botResult.data;
    const mechanisms = botResult.data.mechanism_config?.mechanisms || [];
    const found = mechanisms.find((m) => m.id === mechanismId);
    legacyMechanism.value = found || null;

    let documentValue = workflowResult?.document;
    const loadedPersistedDocument =
      isWorkflowDocument(documentValue) && documentValue.spec.nodes.length > 0;
    if (!isWorkflowDocument(documentValue) || documentValue.spec.nodes.length === 0) {
      const migrationSource = found ? { mechanisms: [found] } : botResult.data.mechanism_config;
      const migrated = migrateMechanismToDocument(migrationSource, botResult.data.name);
      if (!isWorkflowDocument(documentValue) || migrated.spec.nodes.length > 0)
        documentValue = migrated;
    }

    if (!isWorkflowDocument(documentValue)) {
      error.value = '工作流文档格式无效';
      return;
    }

    revision.value = workflowResult?.revision ?? 0;
    publishedRevision.value = workflowResult?.published_revision ?? null;
    documentValue.metadata.revision = revision.value;
    ensureStableKeys(documentValue);
    workflowDocument.value = cloneWorkflowDocument(documentValue);
    baseline.value = loadedPersistedDocument
      ? serializeWorkflowDocument(workflowDocument.value)
      : '';
  } catch (err: any) {
    error.value = err.response?.data?.message || '加载失败';
  } finally {
    loading.value = false;
  }
}

function ensureStableKeys(documentValue: WorkflowDocument) {
  const used = new Set<string>();
  for (const node of documentValue.spec.nodes) {
    if (node.key && !used.has(node.key)) {
      used.add(node.key);
      continue;
    }
    node.key = nextUniqueNodeKey(documentValue, node.type);
    used.add(node.key);
  }
}

function showGateErrors(prefix: string, errors: string[]) {
  operationError.value = `${prefix}：${errors.join('；')}`;
}

async function passValidationGate(action: string): Promise<boolean> {
  if (!workflowDocument.value) return false;
  operationError.value = null;
  const localGate = evaluateWorkflowGate(validationResult.value, (message) =>
    window.confirm(message)
  );
  if (!localGate.allowed) {
    if (localGate.errors.length) showGateErrors(`${action}已被本地验证阻止`, localGate.errors);
    return false;
  }

  const serverResult = await api.validateWorkflow(botId, workflowDocument.value);
  const serverGate = evaluateWorkflowGate(
    { issues: (serverResult.issues || []).map((issue) => ({ ...issue, nodeId: issue.node_id })) },
    (message) => window.confirm(message)
  );
  if (!serverGate.allowed) {
    if (serverGate.errors.length) showGateErrors(`${action}已被服务端验证阻止`, serverGate.errors);
    return false;
  }
  return true;
}

async function handleSave(): Promise<boolean> {
  if (!workflowDocument.value || saveState.value === 'saving') return false;
  const prevSaveState = saveState.value;
  saveState.value = 'saving';
  try {
    if (!(await passValidationGate('保存'))) {
      // operationError 为 null 说明用户主动取消警告确认，不算失败
      saveState.value = operationError.value ? 'error' : prevSaveState;
      return false;
    }
    workflowDocument.value.metadata.revision = revision.value + 1;
    const response = await api.updateWorkflow(botId, {
      revision: revision.value,
      document: workflowDocument.value,
    });
    applyWorkflowResponse(response);
    saveState.value = 'saved';
    channel?.postMessage({ type: 'bot-updated', botId });
    return true;
  } catch (requestError: any) {
    operationError.value = apiErrorMessage(requestError, '保存失败');
    notify.error(operationError.value);
    saveState.value = 'error';
    return false;
  }
}

async function handlePublish() {
  if (!workflowDocument.value) return;
  if (dirty.value && !(await handleSave())) return;
  const prevSaveState = saveState.value;
  saveState.value = 'publishing';
  try {
    if (!(await passValidationGate('发布'))) {
      saveState.value = operationError.value ? 'error' : prevSaveState;
      return;
    }
    const version = await api.publishWorkflow(botId, revision.value);
    publishedRevision.value = version.revision;
    saveState.value = 'saved';
    if (showHistory.value) await loadVersions();
  } catch (requestError: any) {
    operationError.value = apiErrorMessage(requestError, '发布失败');
    notify.error(operationError.value);
    saveState.value = 'error';
  }
}

function applyWorkflowResponse(response: Awaited<ReturnType<typeof api.getWorkflow>>) {
  const next = cloneWorkflowDocument(response.document);
  revision.value = response.revision;
  publishedRevision.value = response.published_revision ?? publishedRevision.value;
  next.metadata.revision = response.revision;
  ensureStableKeys(next);
  workflowDocument.value = next;
  baseline.value = serializeWorkflowDocument(next);
  operationError.value = null;
}

function apiErrorMessage(requestError: any, fallback: string): string {
  if (requestError.response?.status === 409) {
    return '版本冲突：服务端草稿已更新。你的本地内容已保留，请刷新版本后再决定如何处理。';
  }
  const data = requestError.response?.data;
  const detail = (typeof data === 'string' && data) || data?.error || data?.message || data?.detail;
  if (detail) return detail;
  if (requestError.response) {
    return `${fallback}（HTTP ${requestError.response.status}）`;
  }
  return `${fallback}：${requestError.message || '网络错误'}`;
}

function goBack() {
  router.push('/bots');
}

function onNodeClick({ node }: { node: Node }) {
  const evt = events.value.find((e) => e.id === node.id);
  if (evt) editingEvent.value = cloneWorkflowEvent(evt);
}

function closeModal() {
  showAddModal.value = false;
  editingEvent.value = null;
}

function handleEventConfirm(event: WorkflowEvent) {
  if (!workflowDocument.value) return;

  // 确保事件有 ports
  if (!event.ports || event.ports.length === 0) {
    event.ports = getDefaultPorts(event.type);
  }

  const currentEvents = [...events.value];
  const existingIndex = currentEvents.findIndex((e) => e.id === event.id);

  if (existingIndex >= 0) {
    currentEvents[existingIndex] = event;
  } else {
    currentEvents.push(event);
  }

  event.key ||= nextUniqueNodeKey(workflowDocument.value, event.type);
  workflowDocument.value.spec.nodes = currentEvents;
  closeModal();
}

function handleEventDelete(eventId: string) {
  if (!workflowDocument.value) return;

  const updatedEvents = events.value.filter((e) => e.id !== eventId);

  // 删除相关连接
  const updatedConnections = connections.value.filter(
    (c) => c.sourceNodeId !== eventId && c.targetNodeId !== eventId
  );

  workflowDocument.value.spec.nodes = updatedEvents;
  workflowDocument.value.spec.connections = updatedConnections;
  closeModal();
}

function handleEndConditionsUpdate(conditions: WorkflowEndCondition[]) {
  if (workflowDocument.value) workflowDocument.value.spec.endConditions = conditions;
}

// 在事件 ports 中查找端口，找不到时 fallback 到 getDefaultPorts
function findPort(event: WorkflowEvent, portId: string) {
  return getPortById(event.ports || [], portId) || getPortById(getDefaultPorts(event.type), portId);
}

// 连线创建：端口化连接
function onConnect(connection: {
  source: string;
  target: string;
  sourceHandle?: string | null;
  targetHandle?: string | null;
}) {
  if (connection.source === connection.target) {
    showConnectionToast('不能连接到自身');
    return;
  }

  if (!workflowDocument.value) return;

  const sourceEvent = events.value.find((e) => e.id === connection.source);
  const targetEvent = events.value.find((e) => e.id === connection.target);
  if (!sourceEvent || !targetEvent) {
    showConnectionToast('找不到源或目标节点');
    return;
  }

  const sourcePort = findPort(sourceEvent, connection.sourceHandle || '');
  const targetPort = findPort(targetEvent, connection.targetHandle || '');
  if (!sourcePort) {
    showConnectionToast(`源节点"${sourceEvent.name}"上找不到端口 ${connection.sourceHandle || ''}`);
    return;
  }
  if (!targetPort) {
    showConnectionToast(
      `目标节点"${targetEvent.name}"上找不到端口 ${connection.targetHandle || ''}`
    );
    return;
  }

  // 类型兼容检查
  if (!canConnect(sourcePort, targetPort)) {
    showConnectionToast(
      `类型不兼容：${sourcePort.dataType}（${sourcePort.name}）→ ${targetPort.dataType}（${targetPort.name}）`
    );
    return;
  }

  const newConnection: FlowConnection = {
    id: `conn_${connection.source}_${connection.sourceHandle}_${connection.target}_${connection.targetHandle}`,
    sourceNodeId: connection.source,
    sourcePortId: connection.sourceHandle || '',
    targetNodeId: connection.target,
    targetPortId: connection.targetHandle || '',
  };

  workflowDocument.value.spec.connections = [...connections.value, newConnection];
}

// 连线变更：仅做日志，不再处理 remove（删除由 removeConnection 通过 provide/inject 驱动）
function onEdgesChange(_changes: any[]) {
  // no-op: 边删除通过 removeConnection() 直接操作 localMechanism 状态
}

function onNodesChange(changes: any[]) {
  for (const change of changes) {
    if (change.type === 'position' && change.dragging === false && change.position) {
      const node = workflowDocument.value?.spec.nodes.find((item) => item.id === change.id);
      if (node) node.position = { ...change.position };
    }
  }
}

// 端口类型校验（用于 VueFlow 拖拽时的实时预览）
function isValidConnection(connection: {
  source: string;
  target: string;
  sourceHandle?: string | null;
  targetHandle?: string | null;
}) {
  if (connection.source === connection.target) return false;
  const sourceEvent = events.value.find((e) => e.id === connection.source);
  const targetEvent = events.value.find((e) => e.id === connection.target);
  if (!sourceEvent || !targetEvent) return false;

  const sourcePort = findPort(sourceEvent, connection.sourceHandle || '');
  const targetPort = findPort(targetEvent, connection.targetHandle || '');
  if (!sourcePort || !targetPort) return true; // 端口未找到时允许连接，由 onConnect 做最终校验

  return canConnect(sourcePort, targetPort);
}

// 自动布局：使用 dagre 重新计算节点位置
function handleAutoLayout() {
  if (!workflowDocument.value) return;
  const layouted = autoLayoutEvents(events.value, 'LR', connections.value);
  for (const node of layouted) {
    const documentNode = workflowDocument.value.spec.nodes.find((item) => item.id === node.id);
    if (documentNode) documentNode.position = { ...node.position };
  }
  positionTrigger.value++;
}

function handleYamlExport() {
  if (!workflowDocument.value) return;
  const yamlStr = documentToYaml(sanitizeForExport(workflowDocument.value));
  const blob = new Blob([yamlStr], { type: 'text/yaml' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'agent-flow.yaml';
  a.click();
  URL.revokeObjectURL(url);
}

function handleYamlImport() {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.yaml,.yml';
  input.onchange = (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const result = parseWorkflowYamlCandidate(ev.target?.result as string, validate);
      if (!result.candidate) {
        showGateErrors('YAML 导入失败，当前草稿未变更', result.errors);
        return;
      }
      if (
        result.warnings.length > 0 &&
        !window.confirm(
          `导入内容包含以下警告：\n\n${result.warnings.join('\n')}\n\n仍要替换当前草稿吗？`
        )
      ) {
        return;
      }
      const candidate = cloneWorkflowDocument(result.candidate);
      candidate.metadata.revision = revision.value;
      ensureStableKeys(candidate);
      workflowDocument.value = candidate;
      operationError.value = null;
    };
    reader.readAsText(file);
  };
  input.click();
}

async function toggleHistory() {
  showHistory.value = !showHistory.value;
  if (showHistory.value) await loadVersions();
}

async function loadVersions() {
  historyLoading.value = true;
  try {
    versions.value = await api.listWorkflowVersions(botId);
  } catch (requestError: any) {
    operationError.value = apiErrorMessage(requestError, '版本历史加载失败');
    notify.error(operationError.value);
  } finally {
    historyLoading.value = false;
  }
}

async function restoreVersion(versionRevision: number) {
  if (dirty.value && !window.confirm('恢复版本会覆盖当前未保存修改，确定继续吗？')) return;
  if (!window.confirm(`将已发布的 r${versionRevision} 恢复为新草稿？恢复后不会自动发布。`)) return;
  historyLoading.value = true;
  try {
    const response = await api.rollbackWorkflow(botId, versionRevision);
    applyWorkflowResponse(response);
    saveState.value = 'saved';
    showHistory.value = false;
  } catch (requestError: any) {
    operationError.value = apiErrorMessage(requestError, '恢复版本失败');
    notify.error(operationError.value);
  } finally {
    historyLoading.value = false;
  }
}

function formatPublishedAt(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
</script>

<style scoped>
.workflow-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: var(--text-tertiary-color, #a8a29e);
}

.workflow-status__dirty {
  color: var(--color-warning, #b45309);
  font-weight: 600;
}

.workflow-error {
  padding: 8px 20px;
  border-bottom: 1px solid color-mix(in srgb, var(--color-error, #dc2626) 20%, transparent);
  background: color-mix(in srgb, var(--color-error, #dc2626) 8%, transparent);
  color: var(--color-error, #dc2626);
  font-size: 12px;
  line-height: 1.5;
}

.version-popover {
  position: absolute;
  top: 52px;
  right: 20px;
  z-index: 30;
  width: min(360px, calc(100vw - 40px));
  max-height: 420px;
  overflow-y: auto;
  padding: 16px;
  border-radius: var(--radius-md, 12px);
  background: var(--strong-background-color, #fff);
  box-shadow: var(--shadow-md, 0 4px 16px rgba(28, 25, 23, 0.08));
}

.version-popover__header,
.version-item,
.version-item > div {
  display: flex;
  align-items: center;
}

.version-popover__header,
.version-item {
  justify-content: space-between;
  gap: 12px;
}

.version-popover__header {
  color: var(--text-color, #1c1917);
  font-size: 13px;
}

.version-popover__close,
.version-item button {
  color: var(--theme-primary, #5a8f4e);
  font-size: 11px;
}

.version-popover__hint,
.version-popover__empty {
  margin-top: 6px;
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 11px;
}

.version-list {
  margin-top: 12px;
}

.version-item {
  padding: 10px 0;
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
}

.version-item > div {
  gap: 8px;
  color: var(--text-secondary-color, #57534e);
  font-size: 11px;
}

.version-item strong {
  color: var(--text-color, #1c1917);
}

.version-item button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

/* ── Mobile notice ──────────────────────────────────────── */

.mobile-workflow-notice {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 100vh;
  background: var(--background-color);
  padding: 24px;
}

.mobile-notice-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-width: 320px;
}

.mobile-notice-icon {
  width: 80px;
  height: 80px;
  border-radius: var(--radius-xl, 20px);
  background: var(--surface-color);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary-color);
  margin-bottom: 24px;
}

.mobile-notice-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 8px;
}

.mobile-notice-desc {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-secondary-color);
  margin-bottom: 32px;
}

.mobile-notice-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  border-radius: var(--radius-md, 12px);
  background: var(--theme-primary);
  color: white;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: opacity 0.15s ease;
}

.mobile-notice-btn:active {
  opacity: 0.8;
}

/* ── Connection toast ──────────────────────────────────── */

.connection-toast {
  position: absolute;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 20;
  padding: 6px 14px;
  border-radius: var(--radius-sm, 8px);
  font-size: 12px;
  color: #fff;
  pointer-events: none;
  white-space: nowrap;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.connection-toast--error {
  background: var(--color-error, #dc2626);
}

.connection-toast--warn {
  background: var(--color-warning, #d97706);
}

.toast-fade-enter-active,
.toast-fade-leave-active {
  transition:
    opacity 0.25s ease,
    transform 0.25s ease;
}

.toast-fade-enter-from,
.toast-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-6px);
}

.editor-toolbar {
  display: flex;
  gap: 8px;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--surface-tertiary-color, #e8e4de);
  color: var(--text-secondary-color, #57534e);
  cursor: pointer;
  transition: all 0.15s;
  box-shadow: var(--shadow-xs, 0 1px 2px rgba(28, 25, 23, 0.04));
}

.toolbar-btn:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.toolbar-btn--subtle {
  opacity: 0.7;
  font-size: 11px;
  padding: 5px 10px;
}

.toolbar-btn--subtle:hover {
  opacity: 1;
}

.toolbar-validation-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  font-size: 11px;
  border-radius: var(--radius-xs, 4px);
  background: var(--color-error-bg, rgba(239, 68, 68, 0.08));
  color: var(--color-error, #ef4444);
}

.editor-validation {
  max-width: 260px;
}

.editor-validation__item {
  padding: 4px 10px;
  font-size: 11px;
  line-height: 1.4;
  border-radius: var(--radius-xs, 4px);
  background: var(--strong-background-color, #fff);
  color: var(--text-color, #1c1917);
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(28, 25, 23, 0.06));
  margin-bottom: 4px;
}

.editor-validation__item--error {
  background: var(--color-error-bg, rgba(239, 68, 68, 0.08));
  color: var(--color-error, #dc2626);
}

.editor-validation__item--warning {
  background: var(--color-warning-bg, rgba(217, 119, 6, 0.08));
  color: var(--color-warning, #d97706);
}

.bottom-tab {
  padding: 6px 14px;
  font-size: 12px;
  border: none;
  border-radius: var(--radius-xs, 4px);
  background: none;
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  transition: all 0.15s;
}

.bottom-tab:hover {
  color: var(--text-secondary-color, #57534e);
  background: var(--surface-tertiary-color, rgba(0, 0, 0, 0.04));
}

.bottom-tab--active {
  color: var(--text-color, #1c1917);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
  font-weight: 500;
}

/* ── Guide panel ─────────────────────────────────────────── */

.guide-panel {
  padding: 16px 20px;
  max-height: 240px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.guide-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.guide-section__title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color, #1c1917);
}

.guide-section__desc {
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-secondary-color, #57534e);
}

.guide-section__desc code {
  font-size: 10px;
  padding: 1px 4px;
  border-radius: var(--radius-xs, 4px);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
  color: var(--text-color, #1c1917);
  font-family: 'JetBrains Mono', monospace;
}

.guide-type {
  font-size: 11px;
  font-weight: 500;
}

/* ── Variable list ───────────────────────────────────────── */

.guide-vars {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.guide-var {
  display: grid;
  grid-template-columns: 80px 56px 1fr;
  gap: 8px;
  align-items: center;
  font-size: 11px;
}

.guide-var__code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-xs, 4px);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
  color: var(--text-color, #1c1917);
  white-space: nowrap;
}

.guide-var__label {
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
}

.guide-var__desc {
  color: var(--text-tertiary-color, #a8a29e);
}

/* ── Example ─────────────────────────────────────────────── */

.guide-example {
  padding: 8px 10px;
  border-radius: var(--radius-xs, 4px);
  background: color-mix(in srgb, var(--text-tertiary-color, #a8a29e) 6%, transparent);
  font-size: 11px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.guide-example__prompt {
  color: var(--text-secondary-color, #57534e);
  font-weight: 500;
}

.guide-example code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  padding: 1px 4px;
  border-radius: var(--radius-xs, 4px);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 8%, transparent);
}

.guide-example__results {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 2px;
}

.guide-example__results span {
  font-size: 10px;
  color: var(--text-secondary-color, #57534e);
  white-space: nowrap;
}

.guide-example__results code {
  color: var(--text-color, #1c1917);
}
</style>
