<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center" @click.self="$emit('close')">
    <!-- 遮罩 -->
    <div class="absolute inset-0 bg-black/30 backdrop-blur-sm" />

    <!-- 弹窗 -->
    <div
      class="relative w-full max-w-md mx-4 bg-bg-primary rounded-[var(--radius-lg,16px)] shadow-lg overflow-hidden"
    >
      <!-- 头部 -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-border-subtle">
        <h2 class="text-base font-semibold text-text-primary">创建 Bot</h2>
        <button
          class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
          @click="$emit('close')"
        >
          <BsX :size="18" />
        </button>
      </div>

      <!-- 类型选择 -->
      <div class="flex gap-1 px-6 pt-4">
        <button
          v-for="t in botTypes"
          :key="t.value"
          class="flex-1 px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
          :class="
            botType === t.value
              ? 'text-white'
              : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
          "
          :style="botType === t.value ? { background: 'var(--theme-primary)' } : {}"
          @click="botType = t.value"
        >
          {{ t.label }}
        </button>
      </div>

      <!-- 表单 -->
      <div class="px-6 py-4 space-y-4">
        <div>
          <label class="block text-xs text-text-secondary mb-1.5"
            >名称 <span class="text-red-500">*</span></label
          >
          <input
            v-model="name"
            type="text"
            maxlength="40"
            :placeholder="botType === 'external' ? '你的 OneBot Bot 名称' : '给你的 Bot 起个名字'"
            class="w-full px-3 py-2.5 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all"
            @keydown.enter="handleCreate"
          />
          <p class="text-xs text-text-quaternary mt-1">{{ name.length }}/40</p>
        </div>

        <div>
          <label class="block text-xs text-text-secondary mb-1.5">描述</label>
          <textarea
            v-model="description"
            maxlength="500"
            rows="2"
            placeholder="描述 Bot 的用途..."
            class="w-full px-3 py-2.5 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all resize-none"
          />
        </div>

        <!-- OneBot API 提示 -->
        <div
          v-if="botType === 'external'"
          class="p-3 rounded-[var(--radius-sm,8px)] bg-[var(--theme-primary)]/5 text-xs text-text-secondary space-y-1"
        >
          <p class="font-medium text-text-primary">OneBot API Bot</p>
          <p>创建后可获取鉴权 Token，通过 OneBot 协议接入外部 Bot 平台。</p>
          <p>创建成功后在编辑面板查看完整接入指南。</p>
        </div>
      </div>

      <!-- 底部 -->
      <div class="flex justify-end gap-3 px-6 py-4 border-t border-border-subtle">
        <button
          class="px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-secondary hover:bg-hover-bg transition-colors"
          @click="$emit('close')"
        >
          取消
        </button>
        <button
          class="px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] text-white transition-colors disabled:opacity-50"
          style="background: var(--theme-primary)"
          :disabled="!name.trim()"
          @click="handleCreate"
        >
          创建
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { BsX } from 'vue-icons-plus/bs';
import type { BotType } from '../../../../models/types';

const emit = defineEmits<{
  create: [data: { name: string; description?: string; bot_type: BotType }];
  close: [];
}>();

const botTypes: { value: BotType; label: string }[] = [
  { value: 'workflow', label: 'Workflow Bot' },
  { value: 'external', label: 'OneBot API' },
];

const botType = ref<BotType>('workflow');
const name = ref('');
const description = ref('');

function handleCreate() {
  if (!name.value.trim()) return;
  emit('create', {
    name: name.value.trim(),
    description: description.value.trim() || undefined,
    bot_type: botType.value,
  });
}
</script>
