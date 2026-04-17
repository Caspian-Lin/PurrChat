<template>
  <section id="settings-general" class="settings-section">
    <h2 class="settings-section__title">通用</h2>

    <!-- 主题模式 -->
    <div class="space-y-3">
      <h3 class="settings-section__subtitle">外观</h3>
      <div class="flex gap-3">
        <button
          v-for="mode in themeModes"
          :key="mode.value"
          :class="[
            'px-4 py-2.5 rounded-[var(--radius-sm,8px)] text-sm font-medium transition-all duration-200',
            generalSettings.themeMode === mode.value
              ? 'text-white'
              : 'text-text-secondary hover:text-text-primary',
          ]"
          :style="{
            backgroundColor: generalSettings.themeMode === mode.value
              ? 'var(--theme-primary)'
              : 'var(--surface-tertiary-color)',
          }"
          @click="updateSetting('themeMode', mode.value)"
        >
          {{ mode.label }}
        </button>
      </div>
    </div>

    <!-- 主题颜色 -->
    <div class="space-y-3">
      <h3 class="settings-section__subtitle">主题色</h3>
      <div class="flex gap-3 flex-wrap">
        <button
          v-for="color in themeColors"
          :key="color.name"
          class="w-10 h-10 rounded-full transition-all duration-200 border-2"
          :style="{
            backgroundColor: color.value,
            borderColor: generalSettings.themeColor === color.name
              ? 'var(--text-primary)'
              : 'transparent',
            transform: generalSettings.themeColor === color.name ? 'scale(1.15)' : 'scale(1)',
          }"
          :title="color.label"
          @click="updateSetting('themeColor', color.name)"
        />
      </div>
    </div>

    <!-- 字号 -->
    <div class="space-y-3">
      <h3 class="settings-section__subtitle">字号</h3>
      <div class="flex gap-3">
        <button
          v-for="size in fontSizes"
          :key="size.value"
          :class="[
            'px-4 py-2.5 rounded-[var(--radius-sm,8px)] text-sm font-medium transition-all duration-200',
            generalSettings.fontSize === size.value
              ? 'text-white'
              : 'text-text-secondary hover:text-text-primary',
          ]"
          :style="{
            backgroundColor: generalSettings.fontSize === size.value
              ? 'var(--theme-primary)'
              : 'var(--surface-tertiary-color)',
          }"
          @click="updateSetting('fontSize', size.value)"
        >
          {{ size.label }}
        </button>
      </div>
    </div>

    <!-- 本地存储 -->
    <div class="space-y-3">
      <h3 class="settings-section__subtitle">存储</h3>
      <div class="p-4 rounded-[var(--radius-sm,8px)]" style="background: var(--surface-secondary-color)">
        <div class="flex items-center justify-between">
          <span class="text-sm text-text-secondary">本地存储占用</span>
          <span class="text-sm text-text-primary">{{ storageText }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useThemeStore } from '../../../../stores/theme';
import { themeColors } from '../../../../config/theme';
import type { GeneralSettings, ThemeColor } from '../../../../models/types';

interface Props {
  generalSettings: GeneralSettings;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [settings: Partial<GeneralSettings>];
}>();

const themeStore = useThemeStore();

const themeModes = [
  { value: 'light' as const, label: '浅色' },
  { value: 'dark' as const, label: '深色' },
];

const themeColorsList = [
  { name: 'sage' as ThemeColor, label: '鼠尾草', value: themeColors.sage.primary },
  { name: 'iris' as ThemeColor, label: '鸢尾', value: themeColors.iris.primary },
  { name: 'ocean' as ThemeColor, label: '海洋', value: themeColors.ocean.primary },
  { name: 'ember' as ThemeColor, label: '余烬', value: themeColors.ember.primary },
  { name: 'rose' as ThemeColor, label: '玫瑰', value: themeColors.rose.primary },
  { name: 'slate' as ThemeColor, label: '石板', value: themeColors.slate.primary },
  { name: 'clay' as ThemeColor, label: '陶土', value: themeColors.clay.primary },
  { name: 'honey' as ThemeColor, label: '蜂蜜', value: themeColors.honey.primary },
];

const fontSizes = [
  { value: 'small' as const, label: '小' },
  { value: 'medium' as const, label: '中' },
  { value: 'large' as const, label: '大' },
];

// 本地存储估算
const storageUsage = ref<number>(0);
const storageQuota = ref<number>(0);

onMounted(async () => {
  if ('storage' in navigator && 'estimate' in navigator.storage) {
    try {
      const estimate = await navigator.storage.estimate();
      storageUsage.value = estimate.usage || 0;
      storageQuota.value = estimate.quota || 0;
    } catch {
      // 不支持，忽略
    }
  }
});

const storageText = computed(() => {
  if (storageUsage.value === 0) return '计算中...';
  const mb = (storageUsage.value / (1024 * 1024)).toFixed(2);
  return `${mb} MB`;
});

function updateSetting<K extends keyof GeneralSettings>(key: K, value: GeneralSettings[K]) {
  emit('update', { [key]: value });

  // 主题即时预览
  if (key === 'themeMode') {
    themeStore.setMode(value as 'light' | 'dark');
  } else if (key === 'themeColor') {
    themeStore.setColor(value as ThemeColor);
  }
}
</script>
