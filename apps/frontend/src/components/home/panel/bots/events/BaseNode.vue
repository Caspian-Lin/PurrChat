<template>
  <div
    class="base-node"
    :class="[traceStatusClass, { 'base-node--selected': node.selected }]"
    :style="nodeStyle"
  >
    <!-- Header -->
    <div class="base-node__header">
      <span class="base-node__icon">{{ node.data.icon || typeIcon }}</span>
      <span class="base-node__name">{{ node.data.label }}</span>
      <slot name="header-extra" />
      <span
        v-if="node.data.traceStatus"
        class="base-node__status"
        :class="`base-node__status--${node.data.traceStatus}`"
      >
        <span v-if="node.data.traceStatus === 'success'">✓</span>
        <span v-else-if="node.data.traceStatus === 'error'">✗</span>
        <span v-else-if="node.data.traceStatus === 'running'" class="base-node__pulse" />
      </span>
    </div>

    <!-- Port labels -->
    <div v-if="hasPorts" class="base-node__ports">
      <div v-for="(_, rowIndex) in portRowCount" :key="rowIndex" class="base-node__port-row">
        <!-- Input port label -->
        <span
          v-if="inputPorts[rowIndex]"
          class="base-node__port-label base-node__port-label--left"
          :class="labelClasses[inputPorts[rowIndex].id]"
        >
          <span
            v-if="inputPorts[rowIndex].dataType === 'trigger'"
            class="base-node__port-label-icon"
            >▶</span
          >
          <span
            v-else
            class="base-node__port-dot"
            :style="{
              background: PORT_COLORS[inputPorts[rowIndex].dataType] || PORT_COLORS.any,
            }"
          />
          <span
            :style="{
              color:
                inputPorts[rowIndex].dataType === 'trigger'
                  ? undefined
                  : PORT_COLORS[inputPorts[rowIndex].dataType] || undefined,
            }"
            >{{ inputPorts[rowIndex].name }}</span
          >
        </span>
        <span v-else />

        <!-- Output port label -->
        <span
          v-if="outputPorts[rowIndex]"
          class="base-node__port-label base-node__port-label--right"
          :class="labelClasses[outputPorts[rowIndex].id]"
        >
          <span
            v-if="outputPorts[rowIndex].dataType === 'trigger'"
            class="base-node__port-label-icon"
            >▶</span
          >
          <span
            v-else
            class="base-node__port-dot"
            :style="{
              background: PORT_COLORS[outputPorts[rowIndex].dataType] || PORT_COLORS.any,
            }"
          />
          <span
            :style="{
              color:
                outputPorts[rowIndex].dataType === 'trigger'
                  ? undefined
                  : PORT_COLORS[outputPorts[rowIndex].dataType] || undefined,
            }"
            >{{ outputPorts[rowIndex].name }}</span
          >
        </span>
      </div>
    </div>

    <!-- Body slot for type-specific content -->
    <slot name="body" />

    <!-- Summary -->
    <div v-if="node.data.summary" class="base-node__summary">
      {{ node.data.summary }}
    </div>

    <!-- Output variables reference -->
    <div
      v-if="dataOutputPorts.length"
      class="base-node__vars"
    >
      <div class="base-node__vars-header" @click="varsExpanded = !varsExpanded">
        <span class="base-node__vars-icon">{{ varsExpanded ? '▼' : '▶' }}</span>
        <span class="base-node__vars-label">输出变量</span>
      </div>
      <div v-if="varsExpanded" class="base-node__vars-list">
        <span v-for="port in dataOutputPorts" :key="port.id" class="base-node__var">
          <span
            class="base-node__var-dot"
            :style="{ background: PORT_COLORS[port.dataType] || PORT_COLORS.any }"
          />
          <code>{{ node.id }}:{{ port.id }}</code>
        </span>
      </div>
    </div>

    <!-- Input handles -->
    <Handle
      v-for="(port, idx) in inputPorts"
      :key="port.id"
      type="target"
      :id="port.id"
      :position="Position.Left"
      class="base-node__handle"
      :style="{
        background: handleColors[port.id] ?? PORT_COLORS[port.dataType] ?? PORT_COLORS.any,
        top: handleOffset(idx),
      }"
    />

    <!-- Output handles -->
    <Handle
      v-for="(port, idx) in outputPorts"
      :key="port.id"
      type="source"
      :id="port.id"
      :position="Position.Right"
      class="base-node__handle"
      :style="{
        background: handleColors[port.id] ?? PORT_COLORS[port.dataType] ?? PORT_COLORS.any,
        top: handleOffset(idx),
      }"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Handle, Position, useNode } from '@vue-flow/core';
import { PORT_COLORS, NODE_TYPE_META, type PortDataType } from '../../../../../utils/portTypes';
import type { EventPort } from '../../../../../models/types';
import { handleOffset } from './useNodeLayout';

// ─── Public types (imported by child node components) ──────

export interface PortDef {
  id: string;
  name: string;
  dataType: PortDataType;
}

export interface BaseNodeData {
  label: string;
  eventType: string;
  summary?: string;
  icon?: string;
  traceStatus?: 'pending' | 'running' | 'success' | 'error';
  ports?: EventPort[];
  config?: Record<string, any>;
  [key: string]: any;
}

// ─── Props ────────────────────────────────────────────────

const props = withDefaults(
  defineProps<{
    /** Background tint color (mixed at 6% opacity) */
    tint?: string;
    /** Accent color for selected state, handle hover glow */
    accent?: string;
    /** Static input port declarations */
    inputs?: PortDef[];
    /** Static output port declarations */
    outputs?: PortDef[];
    /** Custom Handle colors keyed by port id */
    handleColors?: Record<string, string>;
    /** Custom CSS classes for port labels keyed by port id */
    labelClasses?: Record<string, string>;
  }>(),
  {
    tint: undefined,
    accent: undefined,
    inputs: () => [],
    outputs: () => [],
    handleColors: () => ({}),
    labelClasses: () => ({}),
  }
);

// ─── Node context ─────────────────────────────────────────

const { node } = useNode<BaseNodeData>();

// ─── Computed ──────────────────────────────────────────────

const typeIcon = computed(() => {
  const meta = NODE_TYPE_META[node.data.eventType as keyof typeof NODE_TYPE_META];
  return meta?.icon ?? '';
});

const traceStatusClass = computed(() => {
  const status = node.data.traceStatus;
  if (!status) return '';
  return `base-node--trace-${status}`;
});

const effectiveAccent = computed(() => props.accent || 'var(--theme-primary, #5a8f4e)');

const nodeStyle = computed(() => {
  const style: Record<string, string> = {
    '--node-accent': effectiveAccent.value,
  };
  if (props.tint) {
    style.background = `color-mix(in srgb, ${props.tint} 6%, var(--strong-background-color, #fff))`;
  }
  return style;
});

// ─── Port merging ──────────────────────────────────────────

const mergedPorts = computed<EventPort[]>(() => {
  const hasStaticPorts = props.inputs.length > 0 || props.outputs.length > 0;

  // Fallback: no static declarations → use node.data.ports directly
  if (!hasStaticPorts) {
    return node.data.ports ?? [];
  }

  const map = new Map<string, EventPort>();

  // 1. Static ports from props
  for (const p of props.inputs) {
    map.set(p.id, { ...p, direction: 'input' });
  }
  for (const p of props.outputs) {
    map.set(p.id, { ...p, direction: 'output' });
  }

  // 2. Dynamic ports from node.data.ports (in_custom_*, out_custom_*, etc.)
  for (const p of node.data.ports ?? []) {
    if (!map.has(p.id)) {
      map.set(p.id, p);
    }
  }

  return Array.from(map.values());
});

const inputPorts = computed(() => mergedPorts.value.filter((p) => p.direction === 'input'));
const outputPorts = computed(() => mergedPorts.value.filter((p) => p.direction === 'output'));
const hasPorts = computed(() => mergedPorts.value.length > 0);
const portRowCount = computed(() => Math.max(inputPorts.value.length, outputPorts.value.length));

// 输出变量（排除 trigger 类型的控制流端口）
const dataOutputPorts = computed(() => outputPorts.value.filter((p) => p.dataType !== 'trigger'));
const varsExpanded = ref(false);
</script>

<style scoped>
.base-node {
  background: var(--strong-background-color, #fff);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  border-radius: var(--radius-sm, 8px);
  min-width: 160px;
  max-width: 240px;
  font-size: 12px;
  transition:
    box-shadow 0.2s ease,
    border-color 0.2s ease,
    opacity 0.2s ease;
  box-shadow: var(--shadow-xs, 0 1px 2px rgba(28, 25, 23, 0.04));
  overflow: visible;
}

.base-node:hover {
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(28, 25, 23, 0.06));
}

.base-node--selected {
  border-color: var(--node-accent, var(--theme-primary, #5a8f4e));
  box-shadow: 0 0 0 2px
    color-mix(in srgb, var(--node-accent, var(--theme-primary, #5a8f4e)) 15%, transparent);
}

/* ── Trace status styles ──────────────────────────────────── */

.base-node--trace-pending {
  opacity: 0.6;
}
.base-node--trace-success {
  opacity: 1;
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--theme-primary, #5a8f4e) 20%, transparent);
}
.base-node--trace-error {
  opacity: 1;
  box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.3);
  border-color: rgba(239, 68, 68, 0.4);
}
.base-node--trace-running {
  opacity: 1;
  animation: pulse-border 1.5s ease-in-out infinite;
}

@keyframes pulse-border {
  0%,
  100% {
    box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.3);
  }
  50% {
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
  }
}

/* ── Header ───────────────────────────────────────────────── */

.base-node__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  cursor: pointer;
}

.base-node__icon {
  font-size: 14px;
  line-height: 1;
  width: 18px;
  text-align: center;
}

.base-node__name {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-color, #1c1917);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.base-node__status {
  font-size: 12px;
  flex-shrink: 0;
}
.base-node__status--success {
  color: var(--color-success, #16a34a);
}
.base-node__status--error {
  color: var(--color-error, #dc2626);
}
.base-node__status--pending {
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 14px;
}

.base-node__pulse {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-info, #2563eb);
  animation: pulse-dot 1s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%,
  100% {
    opacity: 0.4;
    transform: scale(0.8);
  }
  50% {
    opacity: 1;
    transform: scale(1.2);
  }
}

/* ── Port labels ──────────────────────────────────────────── */

.base-node__ports {
  padding: 1px 10px 3px;
}

.base-node__port-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 20px;
}

.base-node__port-label {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 2px;
  cursor: default;
}

.base-node__port-label--left {
  padding-left: 4px;
}
.base-node__port-label--right {
  padding-right: 4px;
}

.base-node__port-label-icon {
  font-size: 7px;
  line-height: 1;
  opacity: 0.85;
}

.base-node__port-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* ── Summary ─────────────────────────────────────────────── */

.base-node__summary {
  padding: 2px 10px 5px;
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 10px;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Output variables ───────────────────────────────────── */

.base-node__vars {
  padding: 0 10px 3px;
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.04));
}

.base-node__vars-header {
  display: flex;
  align-items: center;
  gap: 3px;
  padding: 3px 0 0;
  cursor: pointer;
  user-select: none;
}

.base-node__vars-icon {
  font-size: 6px;
  color: var(--text-tertiary-color, #a8a29e);
  width: 8px;
  text-align: center;
}

.base-node__vars-label {
  font-size: 9px;
  color: var(--text-tertiary-color, #a8a29e);
  letter-spacing: 0.3px;
}

.base-node__vars-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 2px;
}

.base-node__var {
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 9px;
  color: var(--text-tertiary-color, #a8a29e);
}

.base-node__var-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  flex-shrink: 0;
}

.base-node__var code {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 9px;
  line-height: 1.2;
  color: var(--text-tertiary-color, #78716c);
  word-break: break-all;
}

/* ── Handle (rectangular, on border) ──────────────────────── */

.base-node__handle {
  width: 8px !important;
  height: 18px !important;
  min-width: 8px !important;
  min-height: 18px !important;
  border: none !important;
  border-radius: 3px !important;
  transition:
    scale 0.15s ease,
    box-shadow 0.15s ease;
}

.base-node__handle:hover {
  scale: 1.15;
  box-shadow: 0 0 0 3px
    color-mix(in srgb, var(--node-accent, var(--theme-primary, #5a8f4e)) 15%, transparent);
}
</style>
