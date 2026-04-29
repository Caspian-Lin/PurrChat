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
      <div class="if-node__summary">
        <span class="if-node__count">{{ conditionCount }} 个条件</span>
        <span class="if-node__logic-badge" :class="'if-node__logic-badge--' + logicType">
          {{ logicType }}
        </span>
      </div>
      <div v-if="preview" class="if-node__preview">
        <span class="if-node__preview-text">{{ preview.left || '?' }}</span>
        <span class="if-node__preview-op">{{ preview.operator }}</span>
        <span class="if-node__preview-text">{{ preview.right || '?' }}</span>
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
];

const OUTPUTS: PortDef[] = [
  { id: 'out_true', name: '真', dataType: 'trigger' },
  { id: 'out_false', name: '假', dataType: 'trigger' },
];

const HANDLE_COLORS: Record<string, string> = {
  out_true: 'var(--color-success, #16a34a)',
  out_false: 'var(--color-error, #dc2626)',
};

const LABEL_CLASSES: Record<string, string> = {
  out_true: 'if-node__port-label--true',
  out_false: 'if-node__port-label--false',
};

const conditions = computed(() => {
  const raw = node.data.config?.conditions;
  if (Array.isArray(raw) && raw.length > 0) return raw as { left: string; operator: string; right: string }[];
  return [];
});

const conditionCount = computed(() => Math.max(conditions.value.length, 1));

const logicType = computed(() => {
  return (node.data.config?.logic as string) || 'AND';
});

const preview = computed(() => {
  if (conditions.value.length > 0) {
    const c = conditions.value[0];
    const left = c.left?.replace(/^\{(.+)\}$/, '$1') || c.left || '?';
    const right = c.right?.replace(/^\{(.+)\}$/, '$1') || c.right || '?';
    return { left, operator: c.operator || '==', right };
  }
  // 旧格式回退
  const op = node.data.config?.operator || '==';
  const left = node.data.config?.left_default || '?';
  const right = node.data.config?.right_default || '?';
  return { left, operator: op, right };
});
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

.if-node__summary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px 3px;
}

.if-node__count {
  font-size: 10px;
  color: var(--text-secondary-color, #57534e);
}

.if-node__logic-badge {
  font-size: 9px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.if-node__logic-badge--AND {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 15%, transparent);
  color: var(--theme-primary, #5a8f4e);
}

.if-node__logic-badge--OR {
  background: color-mix(in srgb, var(--color-warning, #d97706) 15%, transparent);
  color: var(--color-warning, #d97706);
}

.if-node__preview {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 1px 10px 4px;
  overflow: hidden;
}

.if-node__preview-text {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 9px;
  color: var(--text-secondary-color, #57534e);
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.if-node__preview-op {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 9px;
  color: var(--color-warning, #d97706);
  font-weight: 600;
  flex-shrink: 0;
}
</style>
