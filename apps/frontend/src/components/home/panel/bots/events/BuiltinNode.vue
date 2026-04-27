<template>
  <BaseNode tint="#e6a23c" :inputs="INPUTS" :outputs="OUTPUTS" />
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const INPUTS: PortDef[] = [
  { id: 'in_exec', name: '执行', dataType: 'trigger' },
  { id: 'in_input', name: '输入', dataType: 'string' },
];

const OUTPUTS: PortDef[] = [
  { id: 'out_exec', name: '执行', dataType: 'trigger' },
  { id: 'out_output', name: '输出', dataType: 'string' },
];
</script>
