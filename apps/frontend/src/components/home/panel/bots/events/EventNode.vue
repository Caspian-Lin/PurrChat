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

    <!-- Ports -->
    <div v-if="hasPorts" class="event-node__ports">
      <div v-for="(_, rowIndex) in portRowCount" :key="rowIndex" class="event-node__port-row">
        <!-- Input port -->
        <div class="event-node__port-cell event-node__port-cell--left">
          <template v-if="inputPorts[rowIndex]">
            <Handle
              type="target"
              :id="inputPorts[rowIndex].id"
              :position="Position.Left"
              class="event-node__handle"
              :style="handleStyle(inputPorts[rowIndex])"
            />
            <span
              class="event-node__port-label"
              :title="`${inputPorts[rowIndex].name} (${inputPorts[rowIndex].dataType})`"
            >
              <span
                v-if="inputPorts[rowIndex].dataType === 'trigger'"
                class="event-node__trigger-icon"
                >▶</span
              >
              {{ inputPorts[rowIndex].name }}
            </span>
          </template>
        </div>

        <!-- Separator -->
        <div class="event-node__port-separator" />

        <!-- Output port -->
        <div class="event-node__port-cell event-node__port-cell--right">
          <template v-if="outputPorts[rowIndex]">
            <Handle
              type="source"
              :id="outputPorts[rowIndex].id"
              :position="Position.Right"
              class="event-node__handle"
              :style="handleStyle(outputPorts[rowIndex])"
            />
            <span
              class="event-node__port-label"
              :title="`${outputPorts[rowIndex].name} (${outputPorts[rowIndex].dataType})`"
            >
              <span
                v-if="outputPorts[rowIndex].dataType === 'trigger'"
                class="event-node__trigger-icon"
                >▶</span
              >
              {{ outputPorts[rowIndex].name }}
            </span>
          </template>
        </div>
      </div>
    </div>

    <!-- Summary -->
    <div v-if="node.data.summary" class="event-node__summary">
      {{ node.data.summary }}
    </div>
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
  // 无调试状态时不添加任何 class（避免默认 opacity: 0.5）
  if (!status) return '';
  return `event-node--trace-${status}`;
});

// ─── Port computations ──────────────────────────────────────

const ports = computed(() => node.data.ports ?? []);

const hasPorts = computed(() => ports.value.length > 0);

const inputPorts = computed(() => ports.value.filter((p) => p.direction === 'input'));

const outputPorts = computed(() => ports.value.filter((p) => p.direction === 'output'));

const portRowCount = computed(() => Math.max(inputPorts.value.length, outputPorts.value.length));

function handleStyle(port: EventPort) {
  const color = PORT_COLORS[port.dataType] ?? PORT_COLORS.any;
  return { background: color };
}
</script>

<style scoped>
.event-node {
  background: var(--strong-background-color, #fff);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  border-radius: var(--radius-sm, 8px);
  min-width: 180px;
  max-width: 240px;
  font-size: 13px;
  transition:
    box-shadow 0.2s ease,
    border-color 0.2s ease,
    opacity 0.2s ease;
  box-shadow: var(--shadow-xs, 0 1px 2px rgba(28, 25, 23, 0.04));
}

.event-node:hover {
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(28, 25, 23, 0.06));
}

.event-node--selected {
  border-color: var(--theme-primary, #5a8f4e);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
}

/* ── Trace status styles（仅在调试面板中使用） ──────────── */

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

/* ── Header ──────────────────────────────────────────────── */

.event-node__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
}

.event-node__icon {
  font-size: 16px;
  line-height: 1;
  width: 20px;
  text-align: center;
}

.event-node__name {
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

/* ── Ports ───────────────────────────────────────────────── */

.event-node__ports {
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  padding: 4px 0;
}

.event-node__port-row {
  display: flex;
  align-items: center;
  min-height: 26px;
  padding: 0 12px;
}

.event-node__port-cell {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.event-node__port-cell--left {
  justify-content: flex-start;
  gap: 6px;
  padding-right: 4px;
}

.event-node__port-cell--right {
  justify-content: flex-end;
  gap: 6px;
  padding-left: 4px;
}

/* 右侧端口：label 在右，dot 在左 */
.event-node__port-cell--right {
  flex-direction: row-reverse;
}

.event-node__port-separator {
  width: 1px;
  align-self: stretch;
  margin: 2px 4px;
  background: var(--border-subtle-color, rgba(0, 0, 0, 0.08));
}

/* ── Handle（端口圆点） ─────────────────────────────────── */

.event-node__handle {
  width: 10px !important;
  height: 10px !important;
  min-width: 10px;
  min-height: 10px;
  border: none !important;
  border-radius: 50% !important;
  transition:
    transform 0.15s ease,
    box-shadow 0.15s ease;
  flex-shrink: 0;
  /* 覆盖 vue-flow 默认绝对定位 — 通过 flex 布局定位 */
  position: relative !important;
  top: auto !important;
  transform: none !important;
}

.event-node__handle:hover {
  transform: scale(1.3) !important;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
}

/* ── Port label ──────────────────────────────────────────── */

.event-node__port-label {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  gap: 3px;
  cursor: default;
}

.event-node__trigger-icon {
  font-size: 7px;
  line-height: 1;
  opacity: 0.85;
}

/* ── Summary ─────────────────────────────────────────────── */

.event-node__summary {
  padding: 6px 12px 8px;
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 11px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
}

/* ── 节点类型色调（使用 color-mix 适配暗色模式） ────────── */

/* 控制节点 */
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

/* 处理节点 */
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

/* 输出节点 */
.event-node--reply {
  background: color-mix(in srgb, #409eff 6%, var(--strong-background-color, #fff));
}
</style>
