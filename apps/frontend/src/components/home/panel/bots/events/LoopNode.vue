<template>
  <BaseNode
    tint="var(--theme-primary, #5a8f4e)"
    accent="var(--theme-primary, #5a8f4e)"
    :inputs="INPUTS"
    :outputs="OUTPUTS"
    :handle-colors="HANDLE_COLORS"
    :label-classes="LABEL_CLASSES"
  >
    <template #body>
      <div class="loop-node__config">
        <span class="loop-node__detail">
          {{ node.data.config?.max_iterations || 10 }} 次
        </span>
        <span v-if="node.data.config?.condition" class="loop-node__condition">
          {{ node.data.config.condition }}
        </span>
      </div>
    </template>
  </BaseNode>
</template>

<script setup lang="ts">
import { useNode } from '@vue-flow/core';
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const { node } = useNode();

const INPUTS: PortDef[] = [
  { id: 'in_exec', name: '执行', dataType: 'trigger' },
  { id: 'in_condition', name: '条件', dataType: 'boolean' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_body', name: '循环体', dataType: 'trigger' },
  { id: 'out_done', name: '完成', dataType: 'trigger' },
];

const HANDLE_COLORS: Record<string, string> = {
  in_condition: 'var(--color-error, #f56c6c)',
  out_body: 'var(--theme-primary, #5a8f4e)',
  out_done: 'var(--theme-primary, #5a8f4e)',
};

const LABEL_CLASSES: Record<string, string> = {
  out_body: 'loop-node__port-label--body',
  out_done: 'loop-node__port-label--done',
};
</script>

<style scoped>
.loop-node__config {
  padding: 2px 10px 5px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.loop-node__detail {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.loop-node__condition {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

.loop-node__port-label--body {
  color: var(--theme-primary, #5a8f4e);
  font-weight: 500;
}

.loop-node__port-label--done {
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
