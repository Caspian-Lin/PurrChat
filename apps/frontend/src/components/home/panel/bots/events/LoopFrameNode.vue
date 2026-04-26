<template>
  <div class="loop-frame" :class="{ 'loop-frame--selected': node.selected }">
    <!-- Header bar (has background for visual hierarchy) -->
    <div class="loop-frame__header">
      <span class="loop-frame__icon">{{ node.data.icon || '↻' }}</span>
      <span class="loop-frame__name">{{ node.data.label }}</span>
      <span v-if="node.data.summary" class="loop-frame__summary">
        {{ node.data.summary }}
      </span>
      <button
        v-if="addToLoop"
        class="loop-frame__add-btn"
        title="添加事件到循环体"
        @click.stop="handleAddEvent"
      >
        <BsPlus :size="12" />
      </button>
    </div>

    <!-- Port labels (border only) -->
    <div class="loop-frame__port-labels">
      <div class="loop-frame__port-labels-left">
        <span class="loop-frame__port-label loop-frame__port-label--trigger">
          <span class="loop-frame__port-label-icon">▶</span> 执行
        </span>
      </div>
      <div class="loop-frame__port-labels-right">
        <span
          class="loop-frame__port-label loop-frame__port-label--trigger loop-frame__port-label--done"
        >
          完成 <span class="loop-frame__port-label-icon">▶</span>
        </span>
      </div>
    </div>

    <!-- Internal ports row: condition input + body entry output -->
    <div class="loop-frame__internal-ports">
      <!-- Condition input (inside frame) -->
      <div class="loop-frame__internal-port loop-frame__internal-port--condition">
        <span
          class="loop-frame__internal-port-dot"
          style="background: var(--color-error, #f56c6c)"
        />
        <span>条件</span>
      </div>
      <!-- Body entry output (inside frame) -->
      <div class="loop-frame__internal-port loop-frame__internal-port--entry">
        <span class="loop-frame__entry-icon">▶</span>
        <span>循环体入口</span>
      </div>
    </div>

    <!-- in_exec: external execution input (left border) -->
    <Handle
      type="target"
      id="in_exec"
      :position="Position.Left"
      class="loop-frame__handle"
      :style="{ background: PORT_COLORS.trigger, top: handleOffset(0) }"
    />

    <!-- in_condition: condition input (left border, aligned with internal label) -->
    <Handle
      type="target"
      id="in_condition"
      :position="Position.Left"
      class="loop-frame__handle"
      :style="{ background: PORT_COLORS.boolean, top: '78px' }"
    />

    <!-- out_body: loop body entry (right border, aligned with internal label) -->
    <Handle
      type="source"
      id="out_body"
      :position="Position.Right"
      class="loop-frame__handle"
      :style="{ background: PORT_COLORS.trigger, top: '78px' }"
    />

    <!-- out_done: loop exit (right border) -->
    <Handle
      type="source"
      id="out_done"
      :position="Position.Right"
      class="loop-frame__handle"
      :style="{ background: PORT_COLORS.trigger, top: handleOffset(0) }"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position, useNode } from '@vue-flow/core';
import { inject } from 'vue';
import { BsPlus } from 'vue-icons-plus/bs';
import { PORT_COLORS } from '../../../../../utils/portTypes';

interface LoopFrameData {
  label: string;
  eventType: string;
  summary?: string;
  icon?: string;
  config?: Record<string, any>;
  [key: string]: any;
}

const { node } = useNode<LoopFrameData>();

const addToLoop = inject<(loopId: string) => void>('addToLoop');

// header(~24px) + labels-padding(2px) + row_index * row_height(20px) + half_row(10px)
function handleOffset(rowIndex: number): string {
  return `${26 + rowIndex * 20 + 10}px`;
}

function handleAddEvent() {
  addToLoop?.(node.id);
}
</script>

<style scoped>
.loop-frame {
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

.loop-frame--selected {
  border-color: var(--theme-primary, #5a8f4e);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--theme-primary, #5a8f4e) 12%, transparent);
}

/* ── Header bar ─────────────────────────────────────────── */

.loop-frame__header {
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

.loop-frame__icon {
  font-size: 13px;
  line-height: 1;
}

.loop-frame__name {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.loop-frame__summary {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-left: auto;
}

.loop-frame__add-btn {
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

.loop-frame__add-btn:hover {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 10%, transparent);
  border-style: solid;
}

/* ── Port labels (border) ────────────────────────────────── */

.loop-frame__port-labels {
  display: flex;
  justify-content: space-between;
  padding: 1px 10px 0;
}

.loop-frame__port-labels-left,
.loop-frame__port-labels-right {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.loop-frame__port-label {
  font-size: 10px;
  color: var(--text-tertiary-color, #57534e);
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 2px;
  min-height: 20px;
  cursor: default;
}

.loop-frame__port-label--trigger {
  color: var(--text-tertiary-color, #57534e);
}

.loop-frame__port-label--done {
  align-items: flex-end;
}

.loop-frame__port-label-icon {
  font-size: 7px;
  line-height: 1;
  opacity: 0.85;
}

/* ── Internal ports row ──────────────────────────────────── */

.loop-frame__internal-ports {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  pointer-events: auto;
}

.loop-frame__internal-port {
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

.loop-frame__internal-port--condition {
  color: var(--color-error, #f56c6c);
  opacity: 0.7;
}

.loop-frame__internal-port--entry {
  color: var(--theme-primary, #5a8f4e);
  opacity: 0.7;
}

.loop-frame__internal-port-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  flex-shrink: 0;
}

.loop-frame__entry-icon {
  font-size: 8px;
}

/* ── Handle (rectangular, on border) ──────────────────────── */

.loop-frame__handle {
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

.loop-frame__handle:hover {
  scale: 1.15;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
}
</style>
