<template>
  <BaseDropdown
    ref="dropdownRef"
    align="right"
    width="480px"
    maxHeight="400px"
    offsetLeft="0px"
    offset-bottom="70px"
  >
    <template #trigger>
      <button
        class="relative p-2 flex items-center justify-center bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
        :title="title"
      >
        <BsEmojiSmile class="text-2xl" />
      </button>
    </template>
    <div class="emoji-picker-content">
      <Picker
        :data="emojiIndex"
        :native="true"
        :set="set"
        :emoji-size="emojiSize"
        :per-line="perLine"
        :emoji-tooltip="emojiTooltip"
        @select="handleEmojiSelect"
      />
    </div>
  </BaseDropdown>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { BsEmojiSmile } from 'vue-icons-plus/bs';
import { Picker, EmojiIndex } from 'emoji-mart-vue-fast/src';
import data from 'emoji-mart-vue-fast/data/all.json';
import 'emoji-mart-vue-fast/css/emoji-mart.css';
import BaseDropdown from './BaseDropdown.vue';

// 创建 emoji 数据索引
const emojiIndex = new EmojiIndex(data);

interface Props {
  modelValue?: string;
  title?: string;
  set?: 'apple' | 'google' | 'twitter' | 'facebook' | 'native';
  emojiSize?: number;
  perLine?: number;
  emojiTooltip?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: '表情',
  set: 'native',
  emojiSize: 24,
  perLine: 9,
  emojiTooltip: true,
});

const emit = defineEmits<{
  select: [emoji: string];
  'update:modelValue': [value: string];
}>();

const dropdownRef = ref<InstanceType<typeof BaseDropdown>>();

const handleEmojiSelect = (emoji: any) => {
  const emojiChar = emoji.native || emoji;
  emit('select', emojiChar);
  if (props.modelValue !== undefined) {
    emit('update:modelValue', props.modelValue + emojiChar);
  }
  // 关闭 dropdown
  dropdownRef.value?.close();
};
</script>

<style scoped>
.emoji-picker-content {
  overflow: auto;
}

.emoji-picker-content :deep(.emoji-mart) {
  background: var(--surface-color);
  border: none;
  color: var(--text-primary);
}

.emoji-picker-content :deep(.emoji-mart-anchor) {
  color: var(--text-tertiary);
}

.emoji-picker-content :deep(.emoji-mart-anchor:hover),
.emoji-picker-content :deep(.emoji-mart-anchor-selected) {
  color: var(--theme-primary);
}

.emoji-picker-content :deep(.emoji-mart-search) {
  background: var(--bg-quaternary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
}

.emoji-picker-content :deep(.emoji-mart-search input) {
  background: transparent;
  color: var(--text-primary);
}

.emoji-picker-content :deep(.emoji-mart-category-label) {
  background: transparent;
  color: var(--text-tertiary);
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.emoji-picker-content :deep(.emoji-mart-emoji) {
  background: transparent;
}

.emoji-picker-content :deep(.emoji-mart-emoji:hover) {
  background: var(--hover-bg);
}

.emoji-picker-content :deep(.emoji-mart-emoji span) {
  filter: none;
}

/* 深色主题适配 */
@media (prefers-color-scheme: dark) {
  .emoji-picker-content :deep(.emoji-mart) {
    background: var(--surface-color);
    color: var(--text-primary);
  }

  .emoji-picker-content :deep(.emoji-mart-search) {
    background: var(--bg-quaternary);
    border-color: var(--border-color);
    color: var(--text-primary);
  }

  .emoji-picker-content :deep(.emoji-mart-search input) {
    color: var(--text-primary);
  }

  .emoji-picker-content :deep(.emoji-mart-category-label) {
    color: var(--text-tertiary);
  }

  .emoji-picker-content :deep(.emoji-mart-anchor) {
    color: var(--text-tertiary);
  }
}
</style>
