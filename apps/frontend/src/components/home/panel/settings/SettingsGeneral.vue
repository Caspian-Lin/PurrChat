<template>
  <section id="settings-general" class="settings-section space-y-5">
    <h2 class="settings-section__title">通用</h2>

    <!-- 主题模式 -->
    <div class="space-y-3">
      <h3 class="settings-section__subtitle">外观</h3>
      <div class="flex gap-3">
        <button
          v-for="mode in themeModes"
          :key="mode.value"
          :class="[
            'px-4 py-2.5 rounded-[var(--radius-sm)] text-sm font-medium transition-all duration-200',
            generalSettings.themeMode === mode.value
              ? 'text-white'
              : 'text-text-secondary hover:text-text-primary',
          ]"
          :style="{
            backgroundColor:
              generalSettings.themeMode === mode.value
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
          v-for="color in themeColorsList"
          :key="color.name"
          class="w-5 h-5 rounded-md transition-all duration-200 border-2"
          :style="{
            backgroundColor: color.value,
            borderColor:
              generalSettings.themeColor === color.name ? 'var(--text-primary)' : 'transparent',
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
            'px-4 py-2.5 rounded-[var(--radius-sm)] text-sm font-medium transition-all duration-200',
            generalSettings.fontSize === size.value
              ? 'text-white'
              : 'text-text-secondary hover:text-text-primary',
          ]"
          :style="{
            backgroundColor:
              generalSettings.fontSize === size.value
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
      <div
        class="p-4 rounded-[var(--radius-sm)] space-y-3"
        style="background: var(--surface-secondary-color)"
      >
        <div class="space-y-2">
          <div
            v-for="item in storageBreakdown"
            :key="item.label"
            class="flex items-center justify-between text-sm"
          >
            <span style="color: var(--text-secondary-color)">{{ item.label }}</span>
            <span style="color: var(--text-primary-color)">{{ item.size }}</span>
          </div>
          <div class="border-t" style="border-color: var(--border-color)" />
          <div class="flex items-center justify-between text-sm font-medium">
            <span style="color: var(--text-primary-color)">总计</span>
            <span style="color: var(--text-primary-color)">{{ storageText }}</span>
          </div>
        </div>

        <button
          :disabled="clearingCache"
          class="w-full mt-2 px-4 py-2 text-sm rounded-[var(--radius-sm)] transition-all duration-200 disabled:opacity-50"
          style="color: var(--text-secondary-color); background: var(--surface-tertiary-color)"
          @click="handleClearCache"
        >
          {{ clearingCache ? '清除中...' : '清除缓存数据' }}
        </button>
        <p class="text-xs" style="color: var(--text-tertiary-color)">
          清除聊天记录缓存和会话状态缓存，不影响个人设置和账号数据
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useThemeStore } from '../../../../stores/theme';
import { themeColors } from '../../../../config/theme';
import { getCurrentUserId, clearUserData } from '../../../../utils/storageNamespace';
import { messageCacheService } from '../../../../services/messageCache';
import { conversationStateCacheService } from '../../../../services/conversationStateCache';
import { useMessage } from '../../../../composables/useMessage';
import type { GeneralSettings } from '../../../../models/types';
import type { ThemeColor } from '../../../../config/theme';

interface Props {
  generalSettings: GeneralSettings;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [settings: Partial<GeneralSettings>];
}>();

const themeStore = useThemeStore();
const { success, error: showError } = useMessage();

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

// ===== 本地存储统计 =====
const storageUsage = ref<number>(0);
const clearingCache = ref(false);

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
}

const storageBreakdown = computed(() => {
  const breakdown: { label: string; size: string }[] = [];
  const userId = getCurrentUserId();

  if (userId) {
    const keys = Object.keys(localStorage);
    const calcSize = (prefix: string | string[]): number => {
      const prefixes = Array.isArray(prefix) ? prefix : [prefix];
      let bytes = 0;
      for (const key of keys) {
        if (prefixes.some((p) => key.startsWith(p) || key === p)) {
          const val = localStorage.getItem(key);
          if (val) bytes += (key.length + val.length) * 2; // UTF-16
        }
      }
      return bytes;
    };

    const msgSize = calcSize([`msg_${userId}_`, `msg_key_${userId}`]);
    const convSize = calcSize(`conv_state_${userId}_`);
    const aiSize = calcSize([
      `ai_cfg_${userId}`,
      `ai_conv_${userId}`,
      `ai_act_cfg_${userId}`,
      `ai_act_conv_${userId}`,
    ]);

    if (msgSize > 0) breakdown.push({ label: '聊天记录缓存', size: formatBytes(msgSize) });
    if (convSize > 0) breakdown.push({ label: '会话状态', size: formatBytes(convSize) });
    if (aiSize > 0) breakdown.push({ label: 'AI 对话数据', size: formatBytes(aiSize) });
  }

  // 设置数据
  const settingsData = localStorage.getItem('purr-chat-settings');
  const settingsSize = settingsData ? settingsData.length * 2 : 0;
  if (settingsSize > 0) breakdown.push({ label: '应用设置', size: formatBytes(settingsSize) });

  // 账号信息（Cookie token 不在 localStorage 中）
  const userData = localStorage.getItem('user');
  const authSize = userData ? userData.length * 2 : 0;
  if (authSize > 0) breakdown.push({ label: '账号信息', size: formatBytes(authSize) });

  return breakdown;
});

const storageText = computed(() => formatBytes(storageUsage.value));

async function refreshStorageEstimate() {
  if ('storage' in navigator && 'estimate' in navigator.storage) {
    try {
      const estimate = await navigator.storage.estimate();
      storageUsage.value = estimate.usage || 0;
    } catch {
      // 不支持，忽略
    }
  }
}

onMounted(refreshStorageEstimate);

async function handleClearCache() {
  const userId = getCurrentUserId();
  if (!userId) {
    showError('未找到用户信息');
    return;
  }

  clearingCache.value = true;
  try {
    clearUserData(userId);
    // 重新初始化缓存服务
    messageCacheService.init(userId);
    conversationStateCacheService.init(userId);
    // 刷新存储统计
    await refreshStorageEstimate();
    success('缓存已清除');
  } catch {
    showError('清除缓存失败');
  } finally {
    clearingCache.value = false;
  }
}

// ===== 设置更新 =====
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
