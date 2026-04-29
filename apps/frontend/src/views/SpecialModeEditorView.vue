<template>
  <div class="flex flex-col h-screen bg-bg-primary">
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
        <p class="text-xs text-text-tertiary truncate">特殊模式事件链编辑器</p>
      </div>
      <!-- 自动保存提示 -->
      <span v-if="saveState === 'saving'" class="text-xs text-text-tertiary">保存中...</span>
      <span v-else-if="saveState === 'saved'" class="text-xs text-green-600">已保存</span>
      <span v-else-if="saveState === 'error'" class="text-xs text-red-500">保存失败</span>
      <button
        class="px-4 py-1.5 text-xs rounded-[var(--radius-sm,8px)] text-white transition-colors"
        style="background: var(--theme-primary)"
        :disabled="saveState === 'saving'"
        @click="handleSave"
      >
        保存
      </button>
    </div>

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
          <button class="toolbar-btn toolbar-btn--subtle" title="导入 YAML 流程" @click="handleYamlImport">
            <BsUpload :size="13" />
            YAML
          </button>
          <button class="toolbar-btn toolbar-btn--subtle" title="导出 YAML 流程" @click="handleYamlExport">
            <BsDownload :size="13" />
            YAML
          </button>
          <span v-if="validationIssues.length" class="toolbar-validation-badge">
            {{ validationIssues.filter((i) => i.type === 'error').length }} 个问题
          </span>
        </div>

        <!-- 验证警告列表 -->
        <div v-if="validationIssues.length" class="editor-validation absolute top-3 right-3 z-10">
          <div
            v-for="(issue, idx) in validationIssues"
            :key="idx"
            class="editor-validation__item"
            :class="`editor-validation__item--${issue.type}`"
          >
            {{ issue.type === 'error' ? '×' : '!' }} {{ issue.message }}
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
            <h4 class="guide-section__title">循环与条件</h4>
            <p class="guide-section__desc">
              <strong>条件节点</strong>（◇）接收一个布尔值，根据真假走不同分支。
              <strong>循环节点</strong
              >（↻）从「循环体」出口开始执行一条子链，子链末尾需要连回循环节点的「执行」入口，
              形成可见的回环线。当条件为假时，从「完成」出口退出循环。
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
            v-if="localMechanism"
            :bot-id="botId"
            :mechanism="localMechanism"
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
import { ref, computed, watch, onMounted, nextTick, markRaw, reactive } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { VueFlow, ConnectionMode } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import { BsArrowLeft, BsPlus, BsUpload, BsDownload } from 'vue-icons-plus/bs';
import { api } from '../models/api';
import type {
  Bot,
  Mechanism,
  SpecialModeEvent as FullEvent,
  FlowConnection,
  SpecialModeEndCondition,
} from '../models/types';
import type { Node, Edge } from '@vue-flow/core';
import {
  eventsToFlowNodes,
  eventsToFlowEdges,
  autoLayoutEvents,
  validateEventChain,
} from '../utils/eventFlowUtils';
import type { ValidationIssue } from '../utils/eventFlowUtils';
import { canConnect, getPortById, getDefaultPorts } from '../utils/portTypes';
import { ensurePorts } from '../utils/eventPorts';
import { flowToYaml, yamlToFlow } from '../utils/yamlIR';
import TriggerNode from '../components/home/panel/bots/events/TriggerNode.vue';
import EndNode from '../components/home/panel/bots/events/EndNode.vue';
import WaitNode from '../components/home/panel/bots/events/WaitNode.vue';
import IfNode from '../components/home/panel/bots/events/IfNode.vue';
import LoopNode from '../components/home/panel/bots/events/LoopNode.vue';
import SwitchNode from '../components/home/panel/bots/events/SwitchNode.vue';
import MergeNode from '../components/home/panel/bots/events/MergeNode.vue';
import ToolNode from '../components/home/panel/bots/events/ToolNode.vue';
import LlmNode from '../components/home/panel/bots/events/LlmNode.vue';
import BuiltinNode from '../components/home/panel/bots/events/BuiltinNode.vue';
import PythonNode from '../components/home/panel/bots/events/PythonNode.vue';
import TemplateNode from '../components/home/panel/bots/events/TemplateNode.vue';
import ReplyNode from '../components/home/panel/bots/events/ReplyNode.vue';
import HistoryNode from '../components/home/panel/bots/events/HistoryNode.vue';
import EventEdge from '../components/home/panel/bots/events/EventEdge.vue';
import EventConfigModal from '../components/home/panel/bots/events/EventConfigModal.vue';
import EndConditionConfig from '../components/home/panel/bots/events/EndConditionConfig.vue';
import BotDebugPanel from '../components/home/panel/bots/BotDebugPanel.vue';

const route = useRoute();
const router = useRouter();

const botId = route.params.botId as string;
const mechanismId = route.params.mechanismId as string;

// 状态
const loading = ref(true);
const error = ref<string | null>(null);
const saveState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle');
const activeBottomTab = ref<'guide' | 'conditions' | 'debug'>('conditions');
const showAddModal = ref(false);

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
const editingEvent = ref<FullEvent | null>(null);

// 数据
const bot = ref<Bot | null>(null);
const localMechanism = ref<Mechanism | null>(null);

const botName = computed(() => bot.value?.name || 'Bot');
const mechanismName = computed(() => localMechanism.value?.name || '机制');

const events = computed<FullEvent[]>(() => {
  const raw = localMechanism.value?.reply?.special_mode?.events || [];
  return ensurePorts(raw);
});

const endConditions = computed<SpecialModeEndCondition[]>(() => {
  return localMechanism.value?.reply?.special_mode?.end_conditions || [];
});

const connections = computed<FlowConnection[]>(() => {
  return localMechanism.value?.reply?.special_mode?.connections || [];
});

const validationIssues = computed<ValidationIssue[]>(() => {
  return validateEventChain(events.value, connections.value);
});

// VueFlow 注册

const customNodeTypes: Record<string, any> = {
  trigger: markRaw(TriggerNode),
  end: markRaw(EndNode),
  wait: markRaw(WaitNode),
  if: markRaw(IfNode),
  loop: markRaw(LoopNode),
  switch: markRaw(SwitchNode),
  merge: markRaw(MergeNode),
  tool: markRaw(ToolNode),
  llm: markRaw(LlmNode),
  builtin: markRaw(BuiltinNode),
  python: markRaw(PythonNode),
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

// 节点位置缓存（普通对象，非响应式，避免 watch→computed→watch 无限循环）
const positionCache: Record<string, { x: number; y: number }> = {};
const positionTrigger = ref(0);

const flowNodes = computed<Node[]>(() => {
  // 读取 positionTrigger 以建立依赖（仅自动布局时递增）
  positionTrigger.value;
  return eventsToFlowNodes(events.value, positionCache, connections.value);
});

const flowEdges = computed<Edge[]>(() => {
  // 优先使用 connections
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
  nextTick(() => handleAutoLayout());

  // 监听来自其他 Tab 的数据变更通知
  channel?.addEventListener('message', (e) => {
    if (e.data.type === 'bot-updated' && e.data.botId === botId) {
      loadData();
    }
  });
});

async function loadData() {
  loading.value = true;
  error.value = null;

  try {
    const result = await api.getBot(botId);
    if (!result.success || !result.data) {
      error.value = 'Bot 不存在或加载失败';
      return;
    }

    bot.value = result.data;

    // 从 mechanism_config 中找到对应的机制
    const mechanisms = result.data.mechanism_config?.mechanisms || [];
    const found = mechanisms.find((m) => m.id === mechanismId);

    if (!found) {
      error.value = `未找到机制 ${mechanismId}`;
      return;
    }

    // 深拷贝一份用于本地编辑
    localMechanism.value = deepCloneMechanism(found);

    // 防御性检查：确保事件链有 trigger 节点
    const sm = localMechanism.value.reply?.special_mode;
    if (sm && sm.events.length === 0) {
      sm.events = [
        {
          id: 'evt_trigger_default',
          type: 'trigger' as const,
          name: '触发',
          config: {},
          ports: getDefaultPorts('trigger'),
        },
      ];
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '加载失败';
  } finally {
    loading.value = false;
  }
}

function deepCloneMechanism(m: Mechanism): Mechanism {
  return {
    id: m.id,
    name: m.name,
    enabled: m.enabled,
    trigger: {
      ...m.trigger,
      rules: m.trigger?.rules?.map((r) => ({ ...r })) || [],
    },
    reply: {
      ...m.reply,
      predefined: m.reply.predefined
        ? { ...m.reply.predefined, replies: [...(m.reply.predefined.replies || [])] }
        : undefined,
      llm: m.reply.llm ? { ...m.reply.llm } : undefined,
      special_mode: m.reply.special_mode
        ? {
            events: m.reply.special_mode.events.map((e) => ({
              ...e,
              config: e.config ? { ...e.config } : {},
              ports: e.ports ? e.ports.map((p) => ({ ...p })) : undefined,
              position: e.position ? { ...e.position } : undefined,
            })),
            end_conditions: m.reply.special_mode.end_conditions.map((c) => ({ ...c })),
            connections: m.reply.special_mode.connections?.map((c) => ({ ...c })) || [],
          }
        : undefined,
    },
  };
}

async function handleSave() {
  if (!localMechanism.value) return;
  saveState.value = 'saving';

  try {
    // 构建更新后的 mechanism_config
    const mechanisms = bot.value?.mechanism_config?.mechanisms?.map((m) =>
      m.id === mechanismId ? deepCloneMechanism(localMechanism.value!) : m
    ) || [deepCloneMechanism(localMechanism.value!)];

    const result = await api.updateBot(botId, {
      mechanism_config: { mechanisms },
    });

    if (result.success && result.data) {
      bot.value = result.data;
      // 重新从更新后的数据中找到机制
      const found = result.data.mechanism_config?.mechanisms?.find((m) => m.id === mechanismId);
      if (found) {
        localMechanism.value = deepCloneMechanism(found);
      }
      saveState.value = 'saved';
      // 通知主 Tab 数据已更新
      channel?.postMessage({ type: 'bot-updated', botId });
      setTimeout(() => {
        if (saveState.value === 'saved') saveState.value = 'idle';
      }, 2000);
    } else {
      saveState.value = 'error';
    }
  } catch {
    saveState.value = 'error';
  }
}

function goBack() {
  router.back();
}

function onNodeClick({ node }: { node: Node }) {
  const evt = events.value.find((e) => e.id === node.id);
  if (evt) {
    editingEvent.value = { ...evt };
  }
}

function closeModal() {
  showAddModal.value = false;
  editingEvent.value = null;
}

function handleEventConfirm(event: FullEvent) {
  if (!localMechanism.value?.reply?.special_mode) return;

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

  localMechanism.value.reply.special_mode.events = currentEvents;
  closeModal();
}

function handleEventDelete(eventId: string) {
  if (!localMechanism.value?.reply?.special_mode) return;

  const updatedEvents = events.value.filter((e) => e.id !== eventId);

  // 删除相关连接
  const updatedConnections = connections.value.filter(
    (c) => c.sourceNodeId !== eventId && c.targetNodeId !== eventId
  );

  localMechanism.value.reply.special_mode.events = updatedEvents;
  localMechanism.value.reply.special_mode.connections = updatedConnections;
  closeModal();
}

function handleEndConditionsUpdate(conditions: SpecialModeEndCondition[]) {
  if (!localMechanism.value?.reply?.special_mode) return;
  localMechanism.value.reply.special_mode.end_conditions = conditions;
}

// 监听 flowNodes 变化并缓存位置（写入普通对象，不触发 computed 重算）
watch(flowNodes, (nodes) => {
  for (const node of nodes) {
    positionCache[node.id] = { ...node.position };
  }
});

// 在事件 ports 中查找端口，找不到时 fallback 到 getDefaultPorts
function findPort(event: FullEvent, portId: string) {
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

  if (!localMechanism.value?.reply?.special_mode) return;

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

  localMechanism.value.reply.special_mode.connections = [...connections.value, newConnection];
}

// 连线变更：检测删除并同步 connections
function onEdgesChange(changes: any[]) {
  const removeChanges = changes.filter((c) => c.type === 'remove');
  if (removeChanges.length === 0) return;

  if (!localMechanism.value?.reply?.special_mode) return;

  const currentConnections = [...connections.value];
  const removeIds = new Set(removeChanges.map((c) => c.id));
  const updated = currentConnections.filter((c) => !removeIds.has(c.id));

  if (updated.length !== currentConnections.length) {
    localMechanism.value.reply.special_mode.connections = updated;
  }
}

// 节点变更：捕获位置变化并缓存（写入普通对象即可，VueFlow 内部已管理位置）
function onNodesChange(changes: any[]) {
  for (const change of changes) {
    if (change.type === 'position' && change.dragging === false && change.position) {
      positionCache[change.id] = { ...change.position };
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
  const layouted = autoLayoutEvents(events.value, 'LR', connections.value);
  for (const node of layouted) {
    positionCache[node.id] = { ...node.position };
  }
  // 递增 trigger 以通知 flowNodes computed 使用新位置
  positionTrigger.value++;
}

// 事件变化时自动布局
watch(
  () => [events.value.length, connections.value.length],
  () => {
    nextTick(() => handleAutoLayout());
  }
);

// YAML 导出：将当前 events + connections 导出为 YAML 文件
function handleYamlExport() {
  const yamlStr = flowToYaml(events.value, connections.value);
  const blob = new Blob([yamlStr], { type: 'text/yaml' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'agent-flow.yaml';
  a.click();
  URL.revokeObjectURL(url);
}

// YAML 导入：从 YAML 文件解析 events + connections
function handleYamlImport() {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.yaml,.yml';
  input.onchange = (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      try {
        const result = yamlToFlow(ev.target?.result as string);
        if (result.errors.length > 0) {
          console.warn('[YAML Import]', result.errors);
        }
        if (result.events.length > 0) {
          events.value = result.events;
          connections.value = result.connections;
          saveFlow();
        }
      } catch (err) {
        console.error('[YAML Import] 解析失败:', err);
      }
    };
    reader.readAsText(file);
  };
  input.click();
}
</script>

<style scoped>
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
