<template>
  <g @mouseenter="hovered = true" @mouseleave="hovered = false">
    <BaseEdge :id="id" :style="edgeStyle" :path="path[0]" :marker-end="markerEnd" />
    <g v-if="hovered" :transform="`translate(${path[1]}, ${path[2]})`">
      <circle
        r="10"
        fill="var(--strong-background-color, #fff)"
        stroke="var(--border-subtle-color, rgba(0,0,0,0.1))"
        stroke-width="1"
      />
      <text
        text-anchor="middle"
        dominant-baseline="central"
        font-size="11"
        fill="var(--text-tertiary-color, #999)"
        style="cursor: pointer; user-select: none"
        @click.stop="handleDelete"
      >
        ×
      </text>
    </g>
  </g>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { BaseEdge, getBezierPath, useVueFlow } from '@vue-flow/core';
import { PORT_COLORS } from '../../../../../utils/portTypes';

const props = defineProps<{
  id: string;
  source: string;
  target: string;
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
  sourcePosition: any;
  targetPosition: any;
  markerEnd: string;
  style?: any;
  data?: {
    dataType?: string;
    isExec?: boolean;
    sourcePortId?: string;
    targetPortId?: string;
  };
}>();

const hovered = ref(false);
const { removeEdges } = useVueFlow();

const path = computed(() =>
  getBezierPath({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    targetX: props.targetX,
    targetY: props.targetY,
    sourcePosition: props.sourcePosition,
    targetPosition: props.targetPosition,
    curvature: 0.3,
  })
);

const edgeStyle = computed(() => {
  const dataType = props.data?.dataType || 'any';
  const color = PORT_COLORS[dataType as keyof typeof PORT_COLORS] || PORT_COLORS.any;
  return {
    stroke: color,
    strokeWidth: 2,
    ...(props.data?.isExec ? { strokeDasharray: '6 3' } : {}),
    ...props.style,
  };
});

function handleDelete() {
  removeEdges([props.id]);
}
</script>
