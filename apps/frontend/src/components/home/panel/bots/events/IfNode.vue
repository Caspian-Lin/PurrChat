<template>
  <BaseNode
    tint="var(--color-warning, #d97706)"
    accent="var(--color-warning, #d97706)"
    :inputs="INPUTS"
    :outputs="OUTPUTS"
    :handle-colors="HANDLE_COLORS"
    :label-classes="LABEL_CLASSES"
  >
    <template #body>
      <div class="if-node__builder">
        <select v-model="operator" class="if-node__operator" @change="syncOperator">
          <option value="==">等于 ==</option>
          <option value="!=">不等于 !=</option>
          <option value="contains">包含</option>
          <option value=">">&gt; 大于</option>
          <option value="<">&lt; 小于</option>
        </select>
      </div>
    </template>
  </BaseNode>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useNode } from '@vue-flow/core';
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const { node } = useNode();

const INPUTS: PortDef[] = [
  { id: 'in_exec', name: '执行', dataType: 'trigger' },
  { id: 'in_left', name: '左操作数', dataType: 'any' },
  { id: 'in_right', name: '右操作数', dataType: 'any' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_true', name: '真', dataType: 'trigger' },
  { id: 'out_false', name: '假', dataType: 'trigger' },
];

const HANDLE_COLORS: Record<string, string> = {
  out_true: 'var(--color-success, #16a34a)',
  out_false: 'var(--color-error, #dc2626)',
  in_left: 'var(--text-tertiary-color, #a8a29e)',
  in_right: 'var(--text-tertiary-color, #a8a29e)',
};

const LABEL_CLASSES: Record<string, string> = {
  out_true: 'if-node__port-label--true',
  out_false: 'if-node__port-label--false',
};

const operator = ref((node.data.config?.operator as string) || '==');

// 当 config 从外部更新时同步
watch(
  () => node.data.config?.operator,
  (val) => {
    if (typeof val === 'string') operator.value = val;
  }
);

function syncOperator() {
  node.data.config = { ...node.data.config, operator: operator.value };
}
</script>

<style scoped>
.if-node__port-label--true {
  color: var(--color-success, #16a34a);
  font-weight: 500;
}

.if-node__port-label--false {
  color: var(--color-error, #dc2626);
  font-weight: 500;
}

/* ── Operator selector ──────────────────────────────────── */

.if-node__builder {
  padding: 2px 10px 5px;
}

.if-node__operator {
  width: 100%;
  padding: 3px 6px;
  font-size: 10px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  line-height: 1.3;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid color-mix(in srgb, var(--color-warning, #d97706) 20%, transparent);
  background: color-mix(in srgb, var(--color-warning, #d97706) 8%, transparent);
  color: var(--text-color, #1c1917);
  cursor: pointer;
  outline: none;
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='8' height='8' viewBox='0 0 8 8'%3E%3Cpath fill='%23a8a29e' d='M2 3l2 2 2-2'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 6px center;
  padding-right: 18px;
}

.if-node__operator:focus {
  border-color: color-mix(in srgb, var(--color-warning, #d97706) 40%, transparent);
}
</style>
