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
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import EventNode from '../events/EventNode.vue';
import { eventsToFlowNodes, eventsToFlowEdges } from '../../../../../utils/eventFlowUtils';
import type { SpecialModeEvent, EventTrace } from '../../../../../models/types';
import type { Node, Edge } from '@vue-flow/core';

interface Props {
  events: SpecialModeEvent[];
  eventTraces: EventTrace[];
}

const props = defineProps<Props>();

const customNodeTypes = {
  event: markRaw(EventNode),
};

const defaultEdgeOptions = {
  type: 'smoothstep',
  animated: false,
  style: { stroke: 'var(--border-subtle, rgba(0,0,0,0.1))', strokeWidth: 2 },
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

const edges = computed<Edge[]>(() => eventsToFlowEdges(props.events));
</script>

<style scoped>
.debug-flow {
  height: 280px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  background: var(--bg-quaternary, #faf9f7);
  overflow: hidden;
}

.debug-flow__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 13px;
  color: var(--text-tertiary, #999);
}

.debug-flow__canvas {
  width: 100%;
  height: 100%;
}
</style>
