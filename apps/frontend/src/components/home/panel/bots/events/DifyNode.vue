<template>
  <BaseNode
    tint="#2354E6"
    accent="#2354E6"
    :inputs="INPUTS"
    :outputs="OUTPUTS"
    :handle-colors="HANDLE_COLORS"
  >
    <template #body>
      <div class="dify-node__info">
        <span class="dify-node__badge" :class="`dify-node__badge--${node.data.config?.app_type || 'workflow'}`">
          {{ node.data.config?.app_type === 'chatflow' ? 'Chat' : 'WF' }}
        </span>
        <span v-if="node.data.config?.api_base" class="dify-node__url">
          {{ node.data.config.api_base }}
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
.dify-node__info {
  padding: 2px 10px 5px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.dify-node__badge {
  font-size: 9px;
  font-weight: 600;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
  letter-spacing: 0.3px;
}

.dify-node__badge--workflow {
  color: #2354E6;
  background: color-mix(in srgb, #2354E6 12%, transparent);
}

.dify-node__badge--chatflow {
  color: #0FC6C2;
  background: color-mix(in srgb, #0FC6C2 12%, transparent);
}

.dify-node__url {
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}
</style>
