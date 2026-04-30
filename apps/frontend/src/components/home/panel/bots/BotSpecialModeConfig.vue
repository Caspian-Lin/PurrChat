<template>
  <div class="special-mode-config">
    <!-- 无事件时的空状态 -->
    <div v-if="!events || events.length === 0" class="empty-state">
      <p class="empty-state__text">尚未配置事件链</p>
      <p class="empty-state__hint">添加事件来构建 Agent 的工作流程</p>
      <button class="empty-state__btn" @click="showAddModal = true">
        <BsPlus :size="16" />
        添加第一个事件
      </button>
    </div>

    <!-- 事件链 DAG 编辑器 -->
    <template v-else>
      <!-- 工具栏 -->
      <div class="editor-toolbar">
        <button class="toolbar-btn" @click="showAddModal = true">
          <BsPlus :size="14" />
          添加事件
        </button>
        <button class="toolbar-btn" @click="handleAutoLayout">自动布局</button>
        <div class="toolbar-spacer" />
        <button class="toolbar-btn toolbar-btn--subtle" title="导入 YAML 流程" @click="handleYamlImport">
          <BsUpload :size="13" />
          YAML
        </button>
        <button class="toolbar-btn toolbar-btn--subtle" title="导出 YAML 流程" @click="handleYamlExport">
          <BsDownload :size="13" />
          YAML
        </button>
      </div>

      <!-- vue-flow 编辑器 -->
      <div class="flow-container">
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
          class="flow-canvas"
          @node-click="onNodeClick"
          @connect="onConnect"
          @edges-change="onEdgesChange"
          @nodes-change="onNodesChange"
        >
          <Background :gap="20" :size="1" />
          <Controls />
        </VueFlow>
      </div>

      <!-- 结束条件 -->
      <div class="end-conditions">
        <h4 class="end-conditions__title">结束条件</h4>
        <EndConditionConfig :conditions="endConditions" @update="handleEndConditionsUpdate" />
      </div>
    </template>

    <!-- 事件配置弹窗 -->
    <EventConfigModal
      :visible="showAddModal || !!editingEvent"
      :editing-event="editingEvent"
      :existing-events="events || []"
      :connections="connections || []"
      @close="closeModal"
      @confirm="handleEventConfirm"
      @delete="handleEventDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, markRaw, onMounted, nextTick } from 'vue';
import { VueFlow, ConnectionMode } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import { BsPlus, BsUpload, BsDownload } from 'vue-icons-plus/bs';
import TriggerNode from './events/TriggerNode.vue';
import EndNode from './events/EndNode.vue';
import WaitNode from './events/WaitNode.vue';
import IfNode from './events/IfNode.vue';
import LoopNode from './events/LoopNode.vue';
import SwitchNode from './events/SwitchNode.vue';
import MergeNode from './events/MergeNode.vue';
import ToolNode from './events/ToolNode.vue';
import DifyNode from './events/DifyNode.vue';
import N8nNode from './events/N8nNode.vue';
import LlmNode from './events/LlmNode.vue';
import BuiltinNode from './events/BuiltinNode.vue';
import PythonNode from './events/PythonNode.vue';
import TemplateNode from './events/TemplateNode.vue';
import ReplyNode from './events/ReplyNode.vue';
import HistoryNode from './events/HistoryNode.vue';
import EventEdge from './events/EventEdge.vue';
import EventConfigModal from './events/EventConfigModal.vue';
import EndConditionConfig from './events/EndConditionConfig.vue';
import {
  eventsToFlowNodes,
  eventsToFlowEdges,
  autoLayoutEvents,
} from '../../../../utils/eventFlowUtils';
import type {
  SpecialModeEvent as FullEvent,
  FlowConnection,
  SpecialModeEndCondition,
} from '../../../../models/types';
import type { Node, Edge } from '@vue-flow/core';
import { canConnect, getPortById, getDefaultPorts } from '../../../../utils/portTypes';
import { ensurePorts } from '../../../../utils/eventPorts';
import { flowToYaml, yamlToFlow } from '../../../../utils/yamlIR';

interface Props {
  events?: FullEvent[];
  endConditions?: SpecialModeEndCondition[];
  connections?: FlowConnection[];
}

const props = withDefaults(defineProps<Props>(), {
  events: () => [],
  endConditions: () => [],
  connections: () => [],
});

const emit = defineEmits<{
  updateEvents: [events: FullEvent[]];
  updateEndConditions: [conditions: SpecialModeEndCondition[]];
  updateConnections: [connections: FlowConnection[]];
}>();

const showAddModal = ref(false);
const editingEvent = ref<FullEvent | null>(null);

// 注册自定义节点类型

const customNodeTypes: Record<string, any> = {
  trigger: markRaw(TriggerNode),
  end: markRaw(EndNode),
  wait: markRaw(WaitNode),
  if: markRaw(IfNode),
  loop: markRaw(LoopNode),
  switch: markRaw(SwitchNode),
  merge: markRaw(MergeNode),
  tool: markRaw(ToolNode),
  dify: markRaw(DifyNode),
  n8n: markRaw(N8nNode),
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

// 确保 events 都有 ports 字段，且至少有一个 trigger 节点
const ensuredEvents = computed(() => {
  const events = ensurePorts(props.events || []);
  if (events.length === 0) {
    return [
      {
        id: 'evt_trigger_default',
        type: 'trigger' as const,
        name: '触发',
        config: {},
        ports: getDefaultPorts('trigger'),
      },
    ];
  }
  return events;
});

// 将 SpecialModeEvent 转换为 vue-flow Node
const flowNodes = computed<Node[]>(() => {
  // 读取 positionTrigger 以建立依赖（仅自动布局时递增）
  positionTrigger.value;
  return eventsToFlowNodes(ensuredEvents.value, positionCache, props.connections);
});

// 将 connections 转换为 vue-flow Edge
const flowEdges = computed<Edge[]>(() => {
  return eventsToFlowEdges(ensuredEvents.value, props.connections);
});

function onNodeClick({ node }: { node: Node }) {
  const evt = ensuredEvents.value?.find((e) => e.id === node.id);
  if (evt) {
    editingEvent.value = { ...evt };
  }
}

function closeModal() {
  showAddModal.value = false;
  editingEvent.value = null;
}

function handleEventConfirm(event: FullEvent) {
  // 确保事件有 ports
  if (!event.ports || event.ports.length === 0) {
    event.ports = getDefaultPorts(event.type);
  }

  const current = [...(props.events || [])];
  const existingIndex = current.findIndex((e) => e.id === event.id);

  if (existingIndex >= 0) {
    current[existingIndex] = event;
  } else {
    current.push(event);
  }

  emit('updateEvents', current);
  closeModal();
}

function handleEventDelete(eventId: string) {
  const updated = (props.events || []).filter((e) => e.id !== eventId);

  // 删除相关连接
  const updatedConnections = (props.connections || []).filter(
    (c) => c.sourceNodeId !== eventId && c.targetNodeId !== eventId
  );

  emit('updateEvents', updated);
  emit('updateConnections', updatedConnections);
  closeModal();
}

function handleEndConditionsUpdate(conditions: SpecialModeEndCondition[]) {
  emit('updateEndConditions', conditions);
}

// 监听 flowNodes 变化并缓存位置（写入普通对象，不触发 computed 重算）
watch(flowNodes, (nodes) => {
  for (const node of nodes) {
    positionCache[node.id] = { ...node.position };
  }
});

// 连线创建：端口化连接
function onConnect(connection: {
  source: string;
  target: string;
  sourceHandle?: string | null;
  targetHandle?: string | null;
}) {
  if (connection.source === connection.target) return;

  // 获取端口信息进行类型检查
  const sourceEvent = ensuredEvents.value.find((e) => e.id === connection.source);
  const targetEvent = ensuredEvents.value.find((e) => e.id === connection.target);
  if (!sourceEvent || !targetEvent) return;

  const sourcePort = getPortById(sourceEvent.ports || [], connection.sourceHandle || '');
  const targetPort = getPortById(targetEvent.ports || [], connection.targetHandle || '');
  if (!sourcePort || !targetPort) return;

  // 类型兼容检查
  if (!canConnect(sourcePort, targetPort)) {
    console.warn(`无法连接：${sourcePort.dataType} 端口不能连接到 ${targetPort.dataType} 端口`);
    return;
  }

  const newConnection: FlowConnection = {
    id: `conn_${connection.source}_${connection.sourceHandle}_${connection.target}_${connection.targetHandle}`,
    sourceNodeId: connection.source,
    sourcePortId: connection.sourceHandle || '',
    targetNodeId: connection.target,
    targetPortId: connection.targetHandle || '',
  };

  emit('updateConnections', [...(props.connections || []), newConnection]);
}

// 连线变更：检测删除并同步 connections
function onEdgesChange(changes: any[]) {
  const removeChanges = changes.filter((c) => c.type === 'remove');
  if (removeChanges.length === 0) return;

  const currentConnections = [...(props.connections || [])];
  const removeIds = new Set(removeChanges.map((c) => c.id));
  const updated = currentConnections.filter((c) => !removeIds.has(c.id));

  if (updated.length !== currentConnections.length) {
    emit('updateConnections', updated);
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
  const sourceEvent = ensuredEvents.value.find((e) => e.id === connection.source);
  const targetEvent = ensuredEvents.value.find((e) => e.id === connection.target);
  if (!sourceEvent || !targetEvent) return false;

  const sourcePort = getPortById(sourceEvent.ports || [], connection.sourceHandle || '');
  const targetPort = getPortById(targetEvent.ports || [], connection.targetHandle || '');
  if (!sourcePort || !targetPort) return false;

  return canConnect(sourcePort, targetPort);
}

// 自动布局：使用 dagre 重新计算节点位置
function handleAutoLayout() {
  const layouted = autoLayoutEvents(ensuredEvents.value, 'LR', props.connections);
  for (const node of layouted) {
    positionCache[node.id] = { ...node.position };
  }
  // 递增 trigger 以通知 flowNodes computed 使用新位置
  positionTrigger.value++;
}

// YAML 导出：将当前 events + connections 导出为 YAML 文件
function handleYamlExport() {
  const yamlStr = flowToYaml(ensuredEvents.value, props.connections || []);
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
          emit('updateEvents', result.events);
          emit('updateConnections', result.connections);
        }
      } catch (err) {
        console.error('[YAML Import] 解析失败:', err);
      }
    };
    reader.readAsText(file);
  };
  input.click();
}

// 初始化 + 事件变化时自动布局
onMounted(() => {
  nextTick(() => handleAutoLayout());
});

watch(
  () => [props.events?.length, props.connections?.length],
  () => {
    nextTick(() => handleAutoLayout());
  }
);
</script>

<style scoped>
.special-mode-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
}

.empty-state__text {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #1c1917);
  margin-bottom: 4px;
}

.empty-state__hint {
  font-size: 12px;
  color: var(--text-tertiary-color, #a8a29e);
  margin-bottom: 16px;
}

.empty-state__btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  font-size: 13px;
  border-radius: var(--radius-sm, 8px);
  border: 1px dashed var(--border-subtle-color, rgba(0, 0, 0, 0.12));
  background: none;
  color: var(--theme-primary, #5a8f4e);
  cursor: pointer;
  transition: all 0.15s;
}
.empty-state__btn:hover {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 6%, transparent);
  border-style: solid;
}

.editor-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
}

.toolbar-spacer {
  flex: 1;
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

.flow-container {
  height: 300px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--surface-secondary-color, #f4f1ec);
  overflow: hidden;
}

.flow-canvas {
  width: 100%;
  height: 100%;
}

.end-conditions {
  margin-top: 8px;
}

.end-conditions__title {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 8px;
}
</style>
