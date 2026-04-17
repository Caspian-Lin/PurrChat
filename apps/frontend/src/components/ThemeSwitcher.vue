<template>
  <div>
    <button
      class="relative w-10 h-10 flex items-center justify-center rounded-lg bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
      @click="showModal = true"
    >
      <component :is="themeIcon" class="text-text-tertiary" :size="20" />
    </button>

    <BaseModal :show="showModal" title="主题设置" @update:show="showModal = $event">
      <div class="space-y-4">
        <div>
          <div class="px-3 py-2 text-sm font-semibold" style="color: var(--text-secondary-color)">
            主题模式
          </div>
          <div
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-hover-bg transition-colors rounded-lg"
            @click="handleSelect('light')"
          >
            <BsSun />
            <span class="flex-1">浅色</span>
            <div
              v-if="themeStore.mode === 'light'"
              class="w-2 h-2 rounded-full"
              style="background: var(--theme-primary)"
            />
          </div>
          <div
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-hover-bg transition-colors rounded-lg"
            @click="handleSelect('dark')"
          >
            <BsMoon />
            <span class="flex-1">深色</span>
            <div
              v-if="themeStore.mode === 'dark'"
              class="w-2 h-2 rounded-full"
              style="background: var(--theme-primary)"
            />
          </div>
        </div>

        <div class="border-t" style="border-color: var(--border-color)" />

        <div>
          <div class="px-3 py-2 text-sm font-semibold" style="color: var(--text-secondary-color)">
            主题色
          </div>
          <div
            v-for="(color, key) in themeColors"
            :key="key"
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-hover-bg transition-colors rounded-lg"
            @click="handleSelect(`color-${key}`)"
          >
            <div
              class="w-5 h-5 rounded-[var(--radius-xs)]"
              :style="{ background: color.primary }"
            />
            <span class="flex-1">{{ colorNames[key as ThemeColor] || key }}</span>
            <div
              v-if="themeStore.color === key"
              class="w-2 h-2 rounded-full"
              style="background: var(--theme-primary)"
            />
          </div>
        </div>
      </div>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { BsSun, BsMoon } from 'vue-icons-plus/bs';
import { useThemeStore } from '../stores/theme';
import { themeColors } from '../config/theme';
import type { ThemeColor } from '../config/theme';

const colorNames: Record<ThemeColor, string> = {
  sage: '鼠尾草',
  iris: '鸢尾',
  ocean: '海洋',
  ember: '余烬',
  rose: '玫瑰',
  slate: '石板',
  clay: '陶土',
  honey: '蜂蜜',
};
import BaseModal from './common/BaseModal.vue';

const themeStore = useThemeStore();
const showModal = ref(false);

const themeIcon = computed(() => {
  return themeStore.mode === 'light' ? BsSun : BsMoon;
});

const handleSelect = (key: string) => {
  if (key === 'light' || key === 'dark') {
    themeStore.setMode(key);
  } else if (key.startsWith('color-')) {
    const colorKey = key.replace('color-', '') as keyof typeof themeColors;
    themeStore.setColor(colorKey);
  }
  showModal.value = false;
};
</script>

<style scoped></style>
