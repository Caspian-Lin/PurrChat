<template>
  <BaseNode tint="#7c6ff0" :inputs="INPUTS" :outputs="OUTPUTS">
    <template #body>
      <div v-if="node.data.config?.model" class="llm-node__model">
        {{ node.data.config.model }}
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
  { id: 'in_prompt', name: '提示词', dataType: 'string' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_exec', name: '执行', dataType: 'trigger' },
  { id: 'out_output', name: '输出', dataType: 'string' },
];
</script>

<style scoped>
.llm-node__model {
  padding: 2px 10px 5px;
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
