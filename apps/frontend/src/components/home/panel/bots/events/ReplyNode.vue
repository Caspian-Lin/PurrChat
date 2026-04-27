<template>
  <BaseNode tint="#409eff" :inputs="INPUTS" :outputs="OUTPUTS" />
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const INPUTS: PortDef[] = [
  { id: 'in_exec', name: '执行', dataType: 'trigger' },
  { id: 'in_content', name: '内容', dataType: 'string' },
];

const OUTPUTS: PortDef[] = [{ id: 'out_exec', name: '执行', dataType: 'trigger' }];
</script>
