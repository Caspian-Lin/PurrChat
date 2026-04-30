<template>
  <BaseNode
    tint="#FF6D5A"
    accent="#FF6D5A"
    :inputs="INPUTS"
    :outputs="OUTPUTS"
    :handle-colors="HANDLE_COLORS"
  >
    <template #body>
      <div class="n8n-node__info">
        <span class="n8n-node__method">{{ node.data.config?.method || 'POST' }}</span>
        <span v-if="node.data.config?.webhook_url" class="n8n-node__url">
          {{ node.data.config.webhook_url }}
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
  { id: 'in_input', name: '输入', dataType: 'string' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_exec', name: '执行', dataType: 'trigger' },
  { id: 'out_output', name: '输出', dataType: 'string' },
  { id: 'out_error', name: '错误', dataType: 'string' },
];

const HANDLE_COLORS: Record<string, string> = {
  in_input: '#3B82F6',
  out_output: '#3B82F6',
  out_error: '#F56C6C',
};
</script>

<style scoped>
.n8n-node__info {
  padding: 2px 10px 5px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.n8n-node__method {
  font-size: 10px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-weight: 600;
  color: #FF6D5A;
  background: color-mix(in srgb, #FF6D5A 10%, transparent);
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
}

.n8n-node__url {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}
</style>
