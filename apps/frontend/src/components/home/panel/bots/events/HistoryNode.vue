<template>
  <BaseNode tint="#8b5cf6" :inputs="INPUTS" :outputs="OUTPUTS">
    <template #body>
      <div v-if="count > 0" class="history-node__info">
        最近 {{ count }} 条
      </div>
    </template>
  </BaseNode>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useNode } from '@vue-flow/core';
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const { node } = useNode();

const INPUTS: PortDef[] = [
  { id: 'in_exec', name: '执行', dataType: 'trigger' },
  { id: 'in_count', name: '消息数量', dataType: 'number' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_exec', name: '执行', dataType: 'trigger' },
  { id: 'out_history', name: '历史记录', dataType: 'string' },
];

const count = computed(() => {
  const c = node.data.config?.count;
  return typeof c === 'number' ? c : 20;
});
</script>

<style scoped>
.history-node__info {
  padding: 2px 10px 5px;
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
