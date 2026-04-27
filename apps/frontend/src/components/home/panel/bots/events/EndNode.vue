<template>
  <BaseNode tint="var(--theme-primary, #5a8f4e)" :inputs="INPUTS" :outputs="OUTPUTS" />
</template>

<script setup lang="ts">
import BaseNode from './BaseNode.vue';
import type { PortDef } from './BaseNode.vue';

const INPUTS: PortDef[] = [{ id: 'in_exec', name: '执行', dataType: 'trigger' }];

const OUTPUTS: PortDef[] = [];
</script>
