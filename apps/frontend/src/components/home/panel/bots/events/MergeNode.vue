<template>
  <BaseNode
    tint="var(--theme-primary, #5a8f4e)"
    accent="var(--theme-primary, #5a8f4e)"
    :inputs="dynamicInputs"
    :outputs="OUTPUTS"
  >
    <template #body>
      <div class="merge-node__info">
        {{ inputCount }} 路汇聚
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

const inputCount = computed(() => {
  const count = (node.data.config?.input_count as number) || 2;
  return Math.max(2, count);
});

const dynamicInputs = computed<PortDef[]>(() => {
  const count = inputCount.value;
  return Array.from({ length: count }, (_, i) => ({
    id: `in_exec_${i}`,
    name: `输入 ${i + 1}`,
    dataType: 'trigger' as const,
  }));
});

const OUTPUTS: PortDef[] = [{ id: 'out_exec', name: '执行', dataType: 'trigger' }];
</script>

<style scoped>
.merge-node__info {
  padding: 2px 10px 5px;
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
