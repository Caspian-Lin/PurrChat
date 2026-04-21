<template>
  <div class="space-y-4">
    <!-- 类型选择 -->
    <div class="flex gap-2">
      <button
        v-for="mode in modes"
        :key="mode.value"
        class="px-3 py-1.5 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
        :class="
          localConfig.type === mode.value
            ? 'text-white'
            : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
        "
        :style="localConfig.type === mode.value ? { background: 'var(--theme-primary)' } : {}"
        @click="
          localConfig.type = mode.value;
          emitUpdate();
        "
      >
        {{ mode.label }}
      </button>
    </div>

    <!-- 规则触发 -->
    <div v-if="localConfig.type === 'rule'" class="space-y-3">
      <div v-for="(rule, index) in localConfig.rules" :key="index" class="flex items-center gap-2">
        <!-- 规则类型 -->
        <select
          v-model="rule.type"
          class="px-2 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
          @change="emitUpdate()"
        >
          <option value="keyword">关键词</option>
          <option value="regex">正则</option>
          <option value="command">命令</option>
          <option value="equals">精确匹配</option>
        </select>

        <!-- 模式 -->
        <input
          v-model="rule.pattern"
          type="text"
          :placeholder="getRulePlaceholder(rule.type)"
          class="flex-1 px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
          @input="emitUpdate()"
        />

        <!-- 大小写 -->
        <label class="flex items-center gap-1.5 text-xs text-text-tertiary whitespace-nowrap">
          <input
            v-model="rule.case_sensitive"
            type="checkbox"
            class="accent-[var(--theme-primary)]"
            @change="emitUpdate()"
          />
          Aa
        </label>

        <!-- 删除 -->
        <button
          class="p-1.5 rounded-lg hover:bg-red-500/10 text-text-tertiary hover:text-red-500 transition-colors"
          aria-label="删除规则"
          @click="removeRule(index)"
        >
          <BsX :size="14" />
        </button>
      </div>

      <!-- 添加规则 -->
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm,8px)] hover:bg-hover-bg transition-colors"
        @click="addRule"
      >
        <BsPlus :size="12" />
        添加规则
      </button>

      <p class="text-xs text-text-quaternary">无规则时，Bot 会对每条消息触发。</p>
    </div>

    <!-- 概率触发 -->
    <div v-if="localConfig.type === 'probability'" class="space-y-3">
      <div class="flex items-center gap-4">
        <input
          v-model.number="localConfig.probability"
          type="range"
          min="0"
          max="1"
          step="0.05"
          class="flex-1 accent-[var(--theme-primary)]"
          @input="emitUpdate()"
        />
        <span class="text-sm font-mono text-text-primary w-12 text-right">
          {{ Math.round((localConfig.probability || 0) * 100) }}%
        </span>
      </div>
      <p class="text-xs text-text-quaternary">
        Bot 会以设定概率随机触发回复。每个 Bot 最多一个概率机制。
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue';
import { BsPlus, BsX } from 'vue-icons-plus/bs';
import type { TriggerSpec } from '../../../../models/types';

interface Props {
  config: TriggerSpec;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [config: TriggerSpec];
}>();

const modes = [
  { value: 'rule' as const, label: '规则' },
  { value: 'probability' as const, label: '概率' },
];

const localConfig = reactive<TriggerSpec>({
  type: props.config?.type || 'rule',
  rules: props.config?.rules ? [...props.config.rules.map((r) => ({ ...r }))] : [],
  probability: props.config?.probability ?? 0.3,
});

watch(
  () => props.config,
  (newConfig) => {
    if (!newConfig) return;
    localConfig.type = newConfig.type || 'rule';
    localConfig.rules = newConfig.rules ? [...newConfig.rules.map((r) => ({ ...r }))] : [];
    localConfig.probability = newConfig.probability ?? 0.3;
  },
  { deep: true }
);

function emitUpdate() {
  emit('update', { ...localConfig, rules: localConfig.rules ? [...localConfig.rules] : [] });
}

function addRule() {
  if (!localConfig.rules) localConfig.rules = [];
  localConfig.rules.push({
    type: 'keyword',
    pattern: '',
    case_sensitive: false,
  });
  emitUpdate();
}

function removeRule(index: number) {
  localConfig.rules!.splice(index, 1);
  emitUpdate();
}

function getRulePlaceholder(type: string): string {
  const placeholders: Record<string, string> = {
    keyword: '输入关键词',
    regex: '输入正则表达式',
    command: '输入命令前缀，如 /help',
    equals: '精确匹配文本',
  };
  return placeholders[type] || '';
}
</script>
