<template>
  <div class="debug-flow">
    <div v-if="events.length === 0" class="debug-flow__empty">暂无事件链</div>
    <div v-else class="debug-flow__canvas">
      <VueFlow
        :nodes="nodes"
        :edges="edges"
        :node-types="customNodeTypes"
        :default-edge-options="defaultEdgeOptions"
        :nodes-draggable="false"
        :nodes-connectable="false"
        :elements-selectable="false"
        :zoom-on-scroll="true"
        :zoom-on-pinch="true"
        :pan-on-drag="true"
        fit-view-on-init
        :min-zoom="0.3"
        :max-zoom="2"
      >
        <Background :gap="20" :size="1" />
      </VueFlow>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, markRaw } from 'vue';
import { VueFlow } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import TriggerNode from '../events/TriggerNode.vue';
import EndNode from '../events/EndNode.vue';
import WaitNode from '../events/WaitNode.vue';
import IfNode from '../events/IfNode.vue';
import LoopNode from '../events/LoopNode.vue';
import SwitchNode from '../events/SwitchNode.vue';
import MergeNode from '../events/MergeNode.vue';
import ToolNode from '../events/ToolNode.vue';
import DifyNode from '../events/DifyNode.vue';
import N8nNode from '../events/N8nNode.vue';
import LlmNode from '../events/LlmNode.vue';
import BuiltinNode from '../events/BuiltinNode.vue';
import PythonNode from '../events/PythonNode.vue';
import TemplateNode from '../events/TemplateNode.vue';
import ReplyNode from '../events/ReplyNode.vue';
import HistoryNode from '../events/HistoryNode.vue';
import { eventsToFlowNodes, eventsToFlowEdges } from '../../../../../utils/eventFlowUtils';
import type { WorkflowEvent, EventTrace, FlowConnection } from '../../../../../models/types';
import type { Node, Edge } from '@vue-flow/core';

interface Props {
  events: WorkflowEvent[];
  eventTraces: EventTrace[];
  connections?: FlowConnection[];
}

const props = defineProps<Props>();

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

const defaultEdgeOptions = {
  type: 'smoothstep',
  animated: false,
  style: { stroke: 'var(--border-subtle-color, rgba(0,0,0,0.1))', strokeWidth: 2 },
};

// 构建执行状态映射
const traceStatusMap = computed(() => {
  const map: Record<string, string> = {};
  for (const trace of props.eventTraces) {
    map[trace.event_id] = trace.status;
  }
  return map;
});

// 覆盖节点的 data，添加 traceStatus
const nodes = computed<Node[]>(() => {
  const baseNodes = eventsToFlowNodes(props.events);
  return baseNodes.map((node) => ({
    ...node,
    data: {
      ...node.data,
      traceStatus: traceStatusMap.value[node.id] || 'pending',
    },
  }));
});

const edges = computed<Edge[]>(() => eventsToFlowEdges(props.events, props.connections));
</script>

<style scoped>
.debug-flow {
  height: 280px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--surface-secondary-color, #f4f1ec);
  overflow: hidden;
}

.debug-flow__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 13px;
  color: var(--text-tertiary-color, #a8a29e);
}

.debug-flow__canvas {
  width: 100%;
  height: 100%;
}
</style>
