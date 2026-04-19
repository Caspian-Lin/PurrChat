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
      </div>

      <!-- vue-flow 编辑器 -->
      <div class="flow-container">
        <VueFlow
          v-model:nodes="flowNodes"
          v-model:edges="flowEdges"
          :node-types="customNodeTypes"
          :default-edge-options="defaultEdgeOptions"
          fit-view-on-init
          :min-zoom="0.3"
          :max-zoom="2"
          class="flow-canvas"
          @node-click="onNodeClick"
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
      @close="closeModal"
      @confirm="handleEventConfirm"
      @delete="handleEventDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, markRaw } from 'vue';
import { VueFlow } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import '@vue-flow/controls/dist/style.css';
import { BsPlus } from 'vue-icons-plus/bs';
import EventNode from './events/EventNode.vue';
import EventConfigModal from './events/EventConfigModal.vue';
import EndConditionConfig from './events/EndConditionConfig.vue';
import { eventsToFlowNodes, eventsToFlowEdges } from '../../../../utils/eventFlowUtils';
import type { SpecialModeEvent, SpecialModeEndCondition } from '../../../../models/types';
import type { Node, Edge } from '@vue-flow/core';

interface Props {
  events?: SpecialModeEvent[];
  endConditions?: SpecialModeEndCondition[];
}

const props = withDefaults(defineProps<Props>(), {
  events: () => [],
  endConditions: () => [],
});

const emit = defineEmits<{
  updateEvents: [events: SpecialModeEvent[]];
  updateEndConditions: [conditions: SpecialModeEndCondition[]];
}>();

const showAddModal = ref(false);
const editingEvent = ref<SpecialModeEvent | null>(null);

// 注册自定义节点类型
const customNodeTypes = {
  event: markRaw(EventNode),
};

const defaultEdgeOptions = {
  type: 'smoothstep',
  animated: true,
  style: { stroke: 'var(--theme-primary, #5a8f4e)', strokeWidth: 2 },
};

// 节点位置缓存
const nodePositions: Record<string, { x: number; y: number }> = {};

// 将 SpecialModeEvent 转换为 vue-flow Node
const flowNodes = computed<Node[]>({
  get() {
    return eventsToFlowNodes(props.events || [], nodePositions);
  },
  set() {
    // vue-flow 内部更新位置，不需要同步
  },
});

// 将 event.next 转换为 vue-flow Edge
const flowEdges = computed<Edge[]>({
  get() {
    return eventsToFlowEdges(props.events || []);
  },
  set() {
    // vue-flow 内部更新
  },
});

function onNodeClick({ node }: { node: Node }) {
  const evt = props.events?.find((e) => e.id === node.id);
  if (evt) {
    editingEvent.value = { ...evt };
  }
}

function closeModal() {
  showAddModal.value = false;
  editingEvent.value = null;
}

function handleEventConfirm(event: SpecialModeEvent) {
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
  const updated = (props.events || [])
    .filter((e) => e.id !== eventId)
    .map((e) => ({
      ...e,
      next: (e.next || []).filter((n) => n !== eventId),
    }));
  emit('updateEvents', updated);
  closeModal();
}

function handleEndConditionsUpdate(conditions: SpecialModeEndCondition[]) {
  emit('updateEndConditions', conditions);
}

// 监听 node position 变化并缓存
watch(flowNodes, (nodes) => {
  for (const node of nodes) {
    nodePositions[node.id] = { ...node.position };
  }
});
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
  color: var(--text-primary, #1a1a1a);
  margin-bottom: 4px;
}

.empty-state__hint {
  font-size: 12px;
  color: var(--text-tertiary, #999);
  margin-bottom: 16px;
}

.empty-state__btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  font-size: 13px;
  border-radius: var(--radius-sm, 8px);
  border: 1px dashed var(--border-subtle, rgba(0, 0, 0, 0.12));
  background: none;
  color: var(--theme-primary, #5a8f4e);
  cursor: pointer;
  transition: all 0.15s;
}
.empty-state__btn:hover {
  background: rgba(90, 143, 78, 0.06);
  border-style: solid;
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
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--bg-quaternary, #f8f7f5);
  color: var(--text-secondary, #666);
  cursor: pointer;
  transition: all 0.15s;
}
.toolbar-btn:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.flow-container {
  height: 300px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  background: var(--bg-quaternary, #faf9f7);
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
  color: var(--text-secondary, #666);
  margin-bottom: 8px;
}
</style>
