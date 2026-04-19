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
            placeholder="给你的 Bot 起个名字"
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

const emit = defineEmits<{
  create: [data: { name: string; description?: string }];
  close: [];
}>();

const name = ref('');
const description = ref('');

function handleCreate() {
  if (!name.value.trim()) return;
  emit('create', {
    name: name.value.trim(),
    description: description.value.trim() || undefined,
  });
}
</script>
