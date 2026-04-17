<template>
  <nav class="flex flex-col py-4 px-3 h-full">
    <!-- 标题 -->
    <h2 class="text-lg font-semibold text-text-primary px-3 mb-4">设置</h2>

    <!-- 分类列表 -->
    <div class="flex flex-col gap-1">
      <button
        v-for="category in categories"
        :key="category.id"
        :class="[
          'w-full flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-sm,8px)] text-sm transition-colors duration-200',
          activeCategory === category.id
            ? 'bg-[var(--selected-background)] text-text-primary font-medium'
            : 'text-text-secondary hover:bg-[var(--hover-background)] hover:text-text-primary',
        ]"
        @click="$emit('select', category.id)"
      >
        <component :is="category.icon" :size="18" />
        <span>{{ category.label }}</span>
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { BsPersonCircle, BsLayoutSidebar, BsBell, BsPalette, BsInfoCircle } from 'vue-icons-plus/bs';
import type { SettingsCategoryId } from '../../../../models/types';

interface Props {
  activeCategory: SettingsCategoryId;
}

defineProps<Props>();

defineEmits<{
  select: [id: SettingsCategoryId];
}>();

const categories: { id: SettingsCategoryId; label: string; icon: any }[] = [
  { id: 'account', label: '账号', icon: BsPersonCircle },
  { id: 'panels', label: '面板', icon: BsLayoutSidebar },
  { id: 'notifications', label: '通知', icon: BsBell },
  { id: 'general', label: '通用', icon: BsPalette },
  { id: 'about', label: '关于', icon: BsInfoCircle },
];
</script>
