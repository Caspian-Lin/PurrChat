<template>
  <div
    class="event-node"
    :class="[
      `event-node--${node.data.eventType}`,
      traceStatusClass,
      { 'event-node--selected': node.selected },
    ]"
  >
    <!-- Header -->
    <div class="event-node__header">
      <span class="event-node__icon">{{ node.data.icon || typeIcon }}</span>
      <span class="event-node__name">{{ node.data.label }}</span>
      <span
        v-if="node.data.traceStatus"
        class="event-node__status"
        :class="`event-node__status--${node.data.traceStatus}`"
      >
        <span v-if="node.data.traceStatus === 'success'">✓</span>
        <span v-else-if="node.data.traceStatus === 'error'">✗</span>
        <span v-else-if="node.data.traceStatus === 'running'" class="event-node__pulse" />
      </span>
    </div>

    <!-- Port labels (handles are absolutely positioned on border) -->
    <div v-if="hasPorts" class="event-node__ports">
      <div v-for="(_, rowIndex) in portRowCount" :key="rowIndex" class="event-node__port-row">
        <span
          v-if="inputPorts[rowIndex]"
          class="event-node__port-label event-node__port-label--left"
        >
          <span
            v-if="inputPorts[rowIndex].dataType === 'trigger'"
            class="event-node__port-label-icon"
            >▶</span
          >
          <span
            v-else
            class="event-node__port-dot"
            :style="{ background: PORT_COLORS[inputPorts[rowIndex].dataType] || PORT_COLORS.any }"
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
        <span
          v-if="outputPorts[rowIndex]"
          class="event-node__port-label event-node__port-label--right"
        >
          <span
            v-if="outputPorts[rowIndex].dataType === 'trigger'"
            class="event-node__port-label-icon"
            >▶</span
          >
          <span
            v-else
            class="event-node__port-dot"
            :style="{ background: PORT_COLORS[outputPorts[rowIndex].dataType] || PORT_COLORS.any }"
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

    <!-- Summary + type indicator -->
    <div v-if="node.data.summary" class="event-node__summary">
      {{ node.data.summary }}
    </div>

    <!-- Input handles (absolute positioned on left border) -->
    <Handle
      v-for="(port, idx) in inputPorts"
      :key="port.id"
      type="target"
      :id="port.id"
      :position="Position.Left"
      class="event-node__handle"
      :style="{ background: PORT_COLORS[port.dataType] ?? PORT_COLORS.any, top: handleOffset(idx) }"
    />

    <!-- Output handles (absolute positioned on right border) -->
    <Handle
      v-for="(port, idx) in outputPorts"
      :key="port.id"
      type="source"
      :id="port.id"
      :position="Position.Right"
      class="event-node__handle"
      :style="{ background: PORT_COLORS[port.dataType] ?? PORT_COLORS.any, top: handleOffset(idx) }"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Handle, Position, useNode } from '@vue-flow/core';
import { PORT_COLORS, NODE_TYPE_META } from '../../../../../utils/portTypes';
import type { EventPort } from '../../../../../models/types';

interface EventNodeData {
  label: string;
  eventType: string;
  summary?: string;
  icon?: string;
  traceStatus?: 'pending' | 'running' | 'success' | 'error';
  ports?: EventPort[];
  [key: string]: any;
}

const { node } = useNode<EventNodeData>();

const typeIcon = computed(() => {
  const meta = NODE_TYPE_META[node.data.eventType as keyof typeof NODE_TYPE_META];
  return meta?.icon ?? '';
});

const traceStatusClass = computed(() => {
  const status = node.data.traceStatus;
  if (!status) return '';
  return `event-node--trace-${status}`;
});

// ─── Port computations ──────────────────────────────────────

const ports = computed(() => node.data.ports ?? []);
const hasPorts = computed(() => ports.value.length > 0);
const inputPorts = computed(() => ports.value.filter((p) => p.direction === 'input'));
const outputPorts = computed(() => ports.value.filter((p) => p.direction === 'output'));
const portRowCount = computed(() => Math.max(inputPorts.value.length, outputPorts.value.length));

// ─── Handle offset calculation ─────────────────────────────
// header(28px) + ports-padding(2px) + row_index * row_height(20px) + half_row(10px)
function handleOffset(rowIndex: number): string {
  return `${30 + rowIndex * 20 + 10}px`;
}
</script>

<style scoped>
.event-node {
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

.event-node:hover {
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(28, 25, 23, 0.06));
}

.event-node--selected {
  border-color: var(--theme-primary, #5a8f4e);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
}

/* ── Trace status styles ──────────────────────────────────── */

.event-node--trace-pending {
  opacity: 0.6;
}
.event-node--trace-success {
  opacity: 1;
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--theme-primary, #5a8f4e) 20%, transparent);
}
.event-node--trace-error {
  opacity: 1;
  box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.3);
  border-color: rgba(239, 68, 68, 0.4);
}
.event-node--trace-running {
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

.event-node__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  cursor: pointer;
}

.event-node__icon {
  font-size: 14px;
  line-height: 1;
  width: 18px;
  text-align: center;
}

.event-node__name {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-color, #1c1917);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.event-node__status {
  font-size: 12px;
  flex-shrink: 0;
}
.event-node__status--success {
  color: var(--color-success, #16a34a);
}
.event-node__status--error {
  color: var(--color-error, #dc2626);
}
.event-node__status--pending {
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 14px;
}

.event-node__pulse {
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

.event-node__ports {
  padding: 1px 10px 3px;
}

.event-node__port-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 20px;
}

.event-node__port-label {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 2px;
  cursor: default;
}

.event-node__port-label--left {
  padding-left: 4px;
}
.event-node__port-label--right {
  padding-right: 4px;
}

.event-node__port-label-icon {
  font-size: 7px;
  line-height: 1;
  opacity: 0.85;
}

.event-node__port-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* ── Summary ─────────────────────────────────────────────── */

.event-node__summary {
  padding: 2px 10px 5px;
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 10px;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Handle (rectangular, on border) ──────────────────────── */

.event-node__handle {
  width: 8px !important;
  height: 18px !important;
  min-width: 8px !important;
  min-height: 18px !important;
  border: none !important;
  border-radius: 3px !important;
  transition:
    scale 0.15s ease,
    box-shadow 0.15s ease;
  /* Don't override position — let Vue Flow absolute-position on border */
  /* Don't override transform — Vue Flow's translateY(-50%) centers handle at computed top */
}

.event-node__handle:hover {
  scale: 1.15;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
}

/* ── Node type tints ──────────────────────────────────────── */

.event-node--trigger,
.event-node--end,
.event-node--wait,
.event-node--if,
.event-node--loop {
  background: color-mix(
    in srgb,
    var(--theme-primary, #5a8f4e) 6%,
    var(--strong-background-color, #fff)
  );
}

.event-node--llm {
  background: color-mix(in srgb, #7c6ff0 6%, var(--strong-background-color, #fff));
}

.event-node--builtin {
  background: color-mix(in srgb, #e6a23c 6%, var(--strong-background-color, #fff));
}

.event-node--python {
  background: color-mix(in srgb, #5a8f4e 6%, var(--strong-background-color, #fff));
}

.event-node--template {
  background: color-mix(in srgb, #9c78b4 6%, var(--strong-background-color, #fff));
}

.event-node--reply {
  background: color-mix(in srgb, #409eff 6%, var(--strong-background-color, #fff));
}
</style>
