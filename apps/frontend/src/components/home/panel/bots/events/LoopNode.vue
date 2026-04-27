<template>
  <div class="loop-node" :class="{ 'loop-node--selected': node.selected }">
    <!-- Header bar -->
    <div class="loop-node__header">
      <span class="loop-node__icon">{{ node.data.icon || '↻' }}</span>
      <span class="loop-node__name">{{ node.data.label }}</span>
      <span v-if="node.data.summary" class="loop-node__summary">
        {{ node.data.summary }}
      </span>
      <button
        v-if="addToLoop"
        class="loop-node__add-btn"
        title="添加事件到循环体"
        @click.stop="handleAddEvent"
      >
        <BsPlus :size="12" />
      </button>
    </div>

    <!-- Port labels (border only) -->
    <div class="loop-node__port-labels">
      <div class="loop-node__port-labels-left">
        <span class="loop-node__port-label loop-node__port-label--trigger">
          <span class="base-node__port-label-icon">▶</span> 执行
        </span>
      </div>
      <div class="loop-node__port-labels-right">
        <span
          class="loop-node__port-label loop-node__port-label--trigger loop-node__port-label--done"
        >
          完成 <span class="base-node__port-label-icon">▶</span>
        </span>
      </div>
    </div>

    <!-- Internal ports row: condition input + body entry output -->
    <div class="loop-node__internal-ports">
      <div class="loop-node__internal-port loop-node__internal-port--condition">
        <span
          class="loop-node__internal-port-dot"
          style="background: var(--color-error, #f56c6c)"
        />
        <span>条件</span>
      </div>
      <div class="loop-node__internal-port loop-node__internal-port--entry">
        <span class="loop-node__entry-icon">▶</span>
        <span>循环体入口</span>
      </div>
    </div>

    <!-- in_exec: external execution input (left border) -->
    <Handle
      type="target"
      id="in_exec"
      :position="Position.Left"
      class="loop-node__handle"
      :style="{ background: PORT_COLORS.trigger, top: loopHandleOffset(0) }"
    />

    <!-- in_condition: condition input (left border) -->
    <Handle
      type="target"
      id="in_condition"
      :position="Position.Left"
      class="loop-node__handle"
      :style="{ background: PORT_COLORS.boolean, top: '78px' }"
    />

    <!-- out_body: loop body entry (right border) -->
    <Handle
      type="source"
      id="out_body"
      :position="Position.Right"
      class="loop-node__handle"
      :style="{ background: PORT_COLORS.trigger, top: '78px' }"
    />

    <!-- out_done: loop exit (right border) -->
    <Handle
      type="source"
      id="out_done"
      :position="Position.Right"
      class="loop-node__handle"
      :style="{ background: PORT_COLORS.trigger, top: loopHandleOffset(0) }"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position, useNode } from '@vue-flow/core';
import { inject } from 'vue';
import { BsPlus } from 'vue-icons-plus/bs';
import { PORT_COLORS } from '../../../../../utils/portTypes';
import type { BaseNodeData } from './BaseNode.vue';
import { handleOffset } from './useNodeLayout';

const { node } = useNode<BaseNodeData>();

const addToLoop = inject<(loopId: string) => void>('addToLoop');

// LoopNode header 比标准节点矮 4px
const loopHandleOffset = (rowIndex: number) => handleOffset(rowIndex, 26);

function handleAddEvent() {
  addToLoop?.(node.id);
}
</script>

<style scoped>
.loop-node {
  min-width: 500px;
  min-height: 300px;
  width: 100%;
  height: 100%;
  border: 2px dashed color-mix(in srgb, var(--theme-primary, #5a8f4e) 35%, transparent);
  border-radius: var(--radius-md, 12px);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 3%, transparent);
  position: relative;
  pointer-events: none;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.loop-node--selected {
  border-color: var(--theme-primary, #5a8f4e);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--theme-primary, #5a8f4e) 12%, transparent);
}

/* ── Header bar ─────────────────────────────────────────── */

.loop-node__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-md, 12px) var(--radius-md, 12px) 0 0;
  background: color-mix(
    in srgb,
    var(--theme-primary, #5a8f4e) 6%,
    var(--strong-background-color, #fff)
  );
  user-select: none;
  pointer-events: auto;
}

.loop-node__icon {
  font-size: 13px;
  line-height: 1;
}

.loop-node__name {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.loop-node__summary {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-left: auto;
}

.loop-node__add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: var(--radius-xs, 4px);
  border: 1px dashed color-mix(in srgb, var(--theme-primary, #5a8f4e) 40%, transparent);
  background: none;
  color: var(--theme-primary, #5a8f4e);
  cursor: pointer;
  flex-shrink: 0;
  pointer-events: auto;
  transition:
    background 0.15s ease,
    border-color 0.15s ease;
}

.loop-node__add-btn:hover {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 10%, transparent);
  border-style: solid;
}

/* ── Port labels (border) ────────────────────────────────── */

.loop-node__port-labels {
  display: flex;
  justify-content: space-between;
  padding: 1px 10px 0;
}

.loop-node__port-labels-left,
.loop-node__port-labels-right {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.loop-node__port-label {
  font-size: 10px;
  color: var(--text-tertiary-color, #57534e);
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 2px;
  min-height: 20px;
  cursor: default;
}

.loop-node__port-label--trigger {
  color: var(--text-tertiary-color, #57534e);
}

.loop-node__port-label--done {
  align-items: flex-end;
}

/* ── Internal ports row ──────────────────────────────────── */

.loop-node__internal-ports {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  pointer-events: auto;
}

.loop-node__internal-port {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px 2px 5px;
  border-radius: var(--radius-xs, 4px);
  border: 1px dashed var(--border-subtle-color, rgba(0, 0, 0, 0.12));
  background: color-mix(in srgb, var(--strong-background-color, #fff) 60%, transparent);
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  user-select: none;
  white-space: nowrap;
}

.loop-node__internal-port--condition {
  color: var(--color-error, #f56c6c);
  opacity: 0.7;
}

.loop-node__internal-port--entry {
  color: var(--theme-primary, #5a8f4e);
  opacity: 0.7;
}

.loop-node__internal-port-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  flex-shrink: 0;
}

.loop-node__entry-icon {
  font-size: 8px;
}

/* ── Handle (rectangular, on border) ──────────────────────── */

.loop-node__handle {
  width: 8px !important;
  height: 18px !important;
  min-width: 8px !important;
  min-height: 18px !important;
  border: none !important;
  border-radius: 3px !important;
  transition:
    scale 0.15s ease,
    box-shadow 0.15s ease;
  z-index: 1;
  pointer-events: auto;
}

.loop-node__handle:hover {
  scale: 1.15;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
}
</style>
