<template>
  <div
    class="event-node"
    :class="[
      `event-node--${node.data.eventType}`,
      traceStatusClass,
      { 'event-node--selected': node.selected },
    ]"
  >
    <Handle type="target" :position="Position.Left" />

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

    <div v-if="node.data.summary" class="event-node__summary">
      {{ node.data.summary }}
    </div>

    <Handle type="source" :position="Position.Right" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Handle, Position, useNode } from '@vue-flow/core';
import { typeIcons } from '../../../../../utils/eventFlowUtils';

interface EventNodeData {
  label: string;
  eventType: 'llm' | 'builtin' | 'python' | 'reply';
  summary?: string;
  icon?: string;
  traceStatus?: 'pending' | 'running' | 'success' | 'error';
  [key: string]: any;
}

const { node } = useNode<EventNodeData>();

const typeIcon = computed(() => {
  return typeIcons[node.data.eventType] || '';
});

const traceStatusClass = computed(() => {
  const status = node.data.traceStatus;
  if (!status || status === 'pending') return 'event-node--pending';
  return `event-node--trace-${status}`;
});
</script>

<style scoped>
.event-node {
  background: var(--bg-secondary, #f8f7f5);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  border-radius: var(--radius-sm, 8px);
  min-width: 180px;
  max-width: 220px;
  font-size: 13px;
  transition:
    box-shadow 0.2s ease,
    border-color 0.2s ease,
    opacity 0.2s ease;
}

.event-node:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.event-node--selected {
  border-color: var(--theme-primary, #5a8f4e);
  box-shadow: 0 0 0 2px rgba(90, 143, 78, 0.15);
}

/* 执行状态样式 */
.event-node--pending {
  opacity: 0.5;
}

.event-node--trace-success {
  opacity: 1;
  box-shadow: 0 0 0 1px rgba(90, 143, 78, 0.2);
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

.event-node__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.04));
}

.event-node__icon {
  font-size: 16px;
  line-height: 1;
  width: 20px;
  text-align: center;
}

.event-node__name {
  font-weight: 500;
  color: var(--text-primary, #1a1a1a);
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
  color: #5a8f4e;
}

.event-node__status--error {
  color: #ef4444;
}

.event-node__status--pending {
  color: var(--text-tertiary, #999);
  font-size: 14px;
}

.event-node__pulse {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #3b82f6;
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

.event-node__summary {
  padding: 6px 12px 8px;
  color: var(--text-tertiary, #999);
  font-size: 11px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-node--llm {
  border-left: 2px solid #7c6ff0;
}

.event-node--builtin {
  border-left: 2px solid #e6a23c;
}

.event-node--python {
  border-left: 2px solid #5a8f4e;
}

.event-node--reply {
  border-left: 2px solid #409eff;
}
</style>
