<template>
  <BaseNode
    tint="#6366f1"
    accent="#6366f1"
    :inputs="INPUTS"
    :outputs="OUTPUTS"
    :handle-colors="HANDLE_COLORS"
  >
    <template #body>
      <div class="tool-node__info">
        <span v-if="node.data.config?.method" class="tool-node__method">
          {{ node.data.config.method }}
        </span>
        <span v-if="node.data.config?.url" class="tool-node__url">
          {{ node.data.config.url }}
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
  { id: 'in_body', name: '请求体', dataType: 'string' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_exec', name: '执行', dataType: 'trigger' },
  { id: 'out_output', name: '响应', dataType: 'string' },
  { id: 'out_status', name: '状态码', dataType: 'number' },
];

const HANDLE_COLORS: Record<string, string> = {
  in_body: '#3B82F6',
  out_output: '#3B82F6',
  out_status: '#E6A23C',
};
</script>

<style scoped>
.tool-node__info {
  padding: 2px 10px 5px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tool-node__method {
  font-size: 10px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-weight: 600;
  color: #6366f1;
  background: color-mix(in srgb, #6366f1 10%, transparent);
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
}

.tool-node__url {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}
</style>
