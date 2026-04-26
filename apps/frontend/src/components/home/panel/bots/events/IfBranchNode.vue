<template>
  <div
    class="if-branch"
    :class="[{ 'if-branch--selected': node.selected }, `if-branch--${node.data.traceStatus || ''}`]"
  >
    <!-- Header -->
    <div class="if-branch__header">
      <span class="if-branch__diamond">◇</span>
      <span class="if-branch__name">{{ node.data.label }}</span>
      <span
        v-if="node.data.traceStatus"
        class="if-branch__status"
        :class="`if-branch__status--${node.data.traceStatus}`"
      >
        <span v-if="node.data.traceStatus === 'success'">✓</span>
        <span v-else-if="node.data.traceStatus === 'error'">✗</span>
        <span v-else-if="node.data.traceStatus === 'running'" class="if-branch__pulse" />
      </span>
    </div>

    <!-- Port labels -->
    <div class="if-branch__ports">
      <div class="if-branch__port-row">
        <span class="if-branch__port-label if-branch__port-label--left">
          <span class="if-branch__port-label-icon">▶</span>
          执行
        </span>
        <span class="if-branch__port-label if-branch__port-label--true">
          真 <span class="if-branch__port-label-icon">▶</span>
        </span>
      </div>
      <div class="if-branch__port-row">
        <span
          class="if-branch__port-label if-branch__port-label--left if-branch__port-label--boolean"
        >
          条件
        </span>
        <span class="if-branch__port-label if-branch__port-label--false">
          假 <span class="if-branch__port-label-icon">▶</span>
        </span>
      </div>
    </div>

    <!-- Summary -->
    <div v-if="node.data.summary" class="if-branch__summary">
      {{ node.data.summary }}
    </div>

    <!-- Condition expression -->
    <div v-if="node.data.config?.condition" class="if-branch__condition">
      <code>{{ node.data.config.condition }}</code>
    </div>

    <!-- Input handles -->
    <Handle
      type="target"
      id="in_exec"
      :position="Position.Left"
      class="if-branch__handle"
      :style="{ background: PORT_COLORS.trigger, top: handleOffset(0) }"
    />
    <Handle
      type="target"
      id="in_condition"
      :position="Position.Left"
      class="if-branch__handle"
      :style="{ background: PORT_COLORS.boolean, top: handleOffset(1) }"
    />

    <!-- Output handles -->
    <Handle
      type="source"
      id="out_true"
      :position="Position.Right"
      class="if-branch__handle"
      :style="{ background: 'var(--color-success, #16a34a)', top: handleOffset(0) }"
    />
    <Handle
      type="source"
      id="out_false"
      :position="Position.Right"
      class="if-branch__handle"
      :style="{ background: 'var(--color-error, #dc2626)', top: handleOffset(1) }"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position, useNode } from '@vue-flow/core';
import { PORT_COLORS } from '../../../../../utils/portTypes';

interface IfBranchData {
  label: string;
  eventType: string;
  summary?: string;
  icon?: string;
  traceStatus?: 'pending' | 'running' | 'success' | 'error';
  ports?: any[];
  [key: string]: any;
}

const { node } = useNode<IfBranchData>();

function handleOffset(rowIndex: number): string {
  return `${30 + rowIndex * 20 + 10}px`;
}
</script>

<style scoped>
.if-branch {
  background: color-mix(
    in srgb,
    var(--color-warning, #d97706) 6%,
    var(--strong-background-color, #fff)
  );
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

.if-branch:hover {
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(28, 25, 23, 0.06));
}

.if-branch--selected {
  border-color: var(--color-warning, #d97706);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-warning, #d97706) 15%, transparent);
}

/* ── Trace status ────────────────────────────────────────── */

.if-branch--pending {
  opacity: 0.6;
}
.if-branch--success {
  opacity: 1;
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--theme-primary, #5a8f4e) 20%, transparent);
}
.if-branch--error {
  opacity: 1;
  box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.3);
  border-color: rgba(239, 68, 68, 0.4);
}
.if-branch--running {
  opacity: 1;
  animation: pulse-border-if 1.5s ease-in-out infinite;
}

@keyframes pulse-border-if {
  0%,
  100% {
    box-shadow: 0 0 0 1px rgba(217, 119, 6, 0.3);
  }
  50% {
    box-shadow: 0 0 0 3px rgba(217, 119, 6, 0.15);
  }
}

/* ── Header ──────────────────────────────────────────────── */

.if-branch__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  cursor: pointer;
}

.if-branch__diamond {
  font-size: 15px;
  line-height: 1;
  width: 18px;
  text-align: center;
  color: var(--color-warning, #d97706);
}

.if-branch__name {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-color, #1c1917);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.if-branch__status {
  font-size: 12px;
  flex-shrink: 0;
}
.if-branch__status--success {
  color: var(--color-success, #16a34a);
}
.if-branch__status--error {
  color: var(--color-error, #dc2626);
}
.if-branch__status--pending {
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 14px;
}

.if-branch__pulse {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-info, #2563eb);
  animation: pulse-dot-if 1s ease-in-out infinite;
}

@keyframes pulse-dot-if {
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

/* ── Port labels ─────────────────────────────────────────── */

.if-branch__ports {
  padding: 1px 10px 3px;
}

.if-branch__port-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 20px;
}

.if-branch__port-label {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 2px;
  cursor: default;
}

.if-branch__port-label--left {
  padding-left: 4px;
}
.if-branch__port-label--right {
  padding-right: 4px;
}

.if-branch__port-label--boolean {
  color: PORT_COLORS.boolean;
}
.if-branch__port-label--true {
  color: var(--color-success, #16a34a);
  font-weight: 500;
}
.if-branch__port-label--false {
  color: var(--color-error, #dc2626);
  font-weight: 500;
}

.if-branch__port-label-icon {
  font-size: 7px;
  line-height: 1;
  opacity: 0.85;
}

/* ── Summary ─────────────────────────────────────────────── */

.if-branch__summary {
  padding: 2px 10px 5px;
  color: var(--text-tertiary-color, #a8a29e);
  font-size: 10px;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Condition expression ──────────────────────────────── */

.if-branch__condition {
  padding: 2px 10px 5px;
  max-width: 200px;
}

.if-branch__condition code {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 10px;
  line-height: 1.3;
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
  background: color-mix(in srgb, var(--color-warning, #d97706) 8%, transparent);
  color: var(--text-color, #1c1917);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  max-width: 100%;
}

/* ── Handle ──────────────────────────────────────────────── */

.if-branch__handle {
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

.if-branch__handle:hover {
  scale: 1.15;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-warning, #d97706) 15%, transparent);
}
</style>
