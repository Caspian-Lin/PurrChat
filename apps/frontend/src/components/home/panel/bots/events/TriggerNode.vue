<template>
  <BaseNode tint="var(--theme-primary, #5a8f4e)" :inputs="INPUTS" :outputs="OUTPUTS" />
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const INPUTS: PortDef[] = [];

const OUTPUTS: PortDef[] = [
  { id: 'out_exec', name: '执行', dataType: 'trigger' },
  { id: 'out_input', name: '用户消息', dataType: 'string' },
  { id: 'out_username', name: '发送者', dataType: 'string' },
  { id: 'out_time', name: '时间', dataType: 'string' },
  { id: 'out_args', name: '参数', dataType: 'string' },
];
</script>
