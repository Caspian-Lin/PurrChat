<template>
  <div class="end-condition-config">
    <div v-for="(cond, index) in conditions" :key="index" class="condition-item">
      <select v-model="cond.type" class="condition-select">
        <option value="message_match">消息匹配</option>
        <option value="max_rounds">最大轮次</option>
        <option value="timeout">超时 (分钟)</option>
      </select>

      <input
        v-if="cond.type === 'message_match'"
        v-model="cond.pattern"
        type="text"
        class="condition-input"
        placeholder="匹配文本"
      />
      <input
        v-else
        v-model.number="cond.value"
        type="number"
        class="condition-input condition-input--short"
        :min="1"
        :placeholder="cond.type === 'max_rounds' ? '轮次' : '分钟'"
      />

      <button class="condition-remove" title="删除" @click="removeCondition(index)">
        <BsX :size="14" />
      </button>
    </div>

    <button class="condition-add" @click="addCondition">
      <BsPlus :size="14" />
      添加条件
    </button>
  </div>
</template>

<script setup lang="ts">
import { BsX, BsPlus } from 'vue-icons-plus/bs';
import type { SpecialModeEndCondition } from '../../../../models/types';

interface Props {
  conditions: SpecialModeEndCondition[];
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [conditions: SpecialModeEndCondition[]];
}>();

function addCondition() {
  const updated = [...props.conditions, { type: 'max_rounds' as const, value: 50 }];
  emit('update', updated);
}

function removeCondition(index: number) {
  const updated = props.conditions.filter((_, i) => i !== index);
  emit('update', updated);
}

function updateCondition(index: number, field: string, value: any) {
  const updated = props.conditions.map((cond, i) =>
    i === index ? { ...cond, [field]: value } : cond
  );
  emit('update', updated);
}
</script>

<style scoped>
.end-condition-config {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.condition-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.condition-select,
.condition-input {
  padding: 6px 10px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--bg-quaternary, #f8f7f5);
  color: var(--text-primary, #1a1a1a);
  outline: none;
}

.condition-select {
  width: 130px;
  flex-shrink: 0;
}

.condition-input {
  flex: 1;
}

.condition-input--short {
  width: 100px;
  flex: none;
}

.condition-remove {
  padding: 4px;
  border: none;
  background: none;
  color: var(--text-tertiary, #999);
  cursor: pointer;
  border-radius: var(--radius-xs, 4px);
  flex-shrink: 0;
}
.condition-remove:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
}

.condition-add {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-size: 12px;
  border: 1px dashed var(--border-subtle, rgba(0, 0, 0, 0.12));
  border-radius: var(--radius-xs, 4px);
  background: none;
  color: var(--text-tertiary, #999);
  cursor: pointer;
  transition: all 0.15s;
}
.condition-add:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}
</style>
