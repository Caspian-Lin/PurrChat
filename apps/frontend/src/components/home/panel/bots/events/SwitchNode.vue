<template>
  <BaseNode
    tint="var(--color-warning, #d97706)"
    accent="var(--color-warning, #d97706)"
    :inputs="INPUTS"
    :outputs="dynamicOutputs"
    :handle-colors="HANDLE_COLORS"
  >
    <template #body>
      <div class="switch-node__info">
        {{ cases.length }} 个分支
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
  { id: 'in_value', name: '匹配值', dataType: 'any' },
];

const cases = computed(() => {
  const raw = (node.data.config?.cases || []) as { value: string; label: string }[];
  // 保证至少有 2 个分支
  if (raw.length < 2) {
    return [
      { value: '', label: '分支 1' },
      { value: '', label: '分支 2' },
    ];
  }
  return raw;
});

const dynamicOutputs = computed<PortDef[]>(() => {
  const ports: PortDef[] = cases.value.map((c, i) => ({
    id: `out_case_${i}`,
    name: c.label || `分支 ${i + 1}`,
    dataType: 'trigger' as const,
  }));
  ports.push({ id: 'out_default', name: '默认', dataType: 'trigger' });
  return ports;
});

const HANDLE_COLORS: Record<string, string> = {
  in_value: 'var(--text-tertiary-color, #a8a29e)',
  out_default: 'var(--text-tertiary-color, #a8a29e)',
};
</script>

<style scoped>
.switch-node__info {
  padding: 2px 10px 5px;
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
