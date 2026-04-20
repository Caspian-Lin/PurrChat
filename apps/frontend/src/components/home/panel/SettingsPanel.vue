<template>
  <BasePanel
    panel-id="settings"
    :initial-sidebar-width="220"
    :min-sidebar-width="180"
    :max-sidebar-width="360"
  >
    <template #sidebar>
      <SettingsSidebar :active-category="activeCategory" @select="scrollToSection" />
    </template>

    <!-- 右侧设置内容 — 连续滚动长栏 -->
    <div ref="scrollContent" class="flex-1 overflow-y-auto">
      <div class="max-w-3xl mx-auto px-12 py-6 space-y-10">
        <!-- 账号设置 -->
        <SettingsAccount :user="authStore.currentUser" />

        <!-- 面板设置 -->
        <SettingsPanels
          :panel-settings="settingsStore.settings.panels"
          @update="settingsStore.updatePanelSettings"
        />

        <!-- 通知设置 -->
        <SettingsNotifications
          :notification-settings="settingsStore.settings.notifications"
          @update="settingsStore.updateNotificationSettings"
        />

        <!-- 通用设置 -->
        <SettingsGeneral
          :general-settings="settingsStore.settings.general"
          @update="settingsStore.updateGeneralSettings"
        />

        <!-- 关于 -->
        <SettingsAbout />
      </div>
    </div>

    <!-- 浮动保存按钮 -->
    <SaveButton
      :is-dirty="settingsStore.isDirty"
      :is-saving="settingsStore.isSaving"
      @save="settingsStore.save"
    />

    <!-- 未保存离开警告 -->
    <UnsavedWarningModal
      :show="showUnsavedWarning"
      @cancel="showUnsavedWarning = false"
      @discard="handleDiscard"
    />
  </BasePanel>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue';
import { onBeforeRouteLeave } from 'vue-router';
import { useSettingsStore } from '../../../stores/settings';
import { useAuthStore } from '../../../stores/auth';
import { useThemeStore } from '../../../stores/theme';
import type { SettingsCategoryId } from '../../../models/types';
import BasePanel from './BasePanel.vue';
import SettingsSidebar from './settings/SettingsSidebar.vue';
import SettingsAccount from './settings/SettingsAccount.vue';
import SettingsPanels from './settings/SettingsPanels.vue';
import SettingsNotifications from './settings/SettingsNotifications.vue';
import SettingsGeneral from './settings/SettingsGeneral.vue';
import SettingsAbout from './settings/SettingsAbout.vue';
import SaveButton from './settings/SaveButton.vue';
import UnsavedWarningModal from './settings/UnsavedWarningModal.vue';

const settingsStore = useSettingsStore();
const authStore = useAuthStore();
const themeStore = useThemeStore();

const scrollContent = ref<HTMLElement | null>(null);
const activeCategory = ref<SettingsCategoryId>('account');
const showUnsavedWarning = ref(false);
let pendingNavigation: ((val?: boolean) => void) | null = null;
let observer: IntersectionObserver | null = null;

// 所有分类 ID 顺序
const categoryIds: SettingsCategoryId[] = [
  'account',
  'panels',
  'notifications',
  'general',
  'about',
];

// 点击左侧导航 → 平滑滚动到对应小节
function scrollToSection(id: SettingsCategoryId) {
  const el = document.getElementById(`settings-${id}`);
  if (el && scrollContent.value) {
    const container = scrollContent.value;
    const targetTop = el.offsetTop - container.offsetTop;
    container.scrollTo({
      top: targetTop - 24,
      behavior: 'smooth',
    });
  }
}

// IntersectionObserver 监测当前可见小节
function setupScrollSpy() {
  if (!scrollContent.value) return;

  const container = scrollContent.value;

  observer = new IntersectionObserver(
    (entries) => {
      const visibleEntries = entries.filter((entry) => entry.isIntersecting);
      if (visibleEntries.length === 0) return;

      let topEntry = visibleEntries[0];
      let topOffset = Infinity;
      for (const entry of visibleEntries) {
        const rect = entry.boundingClientRect;
        if (rect.top < topOffset) {
          topOffset = rect.top;
          topEntry = entry;
        }
      }

      const id = topEntry.target.id.replace('settings-', '') as SettingsCategoryId;
      if (categoryIds.includes(id)) {
        activeCategory.value = id;
      }
    },
    {
      root: container,
      rootMargin: '-10% 0px -70% 0px',
      threshold: 0,
    }
  );

  categoryIds.forEach((id) => {
    const el = document.getElementById(`settings-${id}`);
    if (el) observer!.observe(el);
  });
}

// 离开守卫
onBeforeRouteLeave((_to, _from, next) => {
  if (settingsStore.isDirty) {
    showUnsavedWarning.value = true;
    pendingNavigation = next;
  } else {
    next();
  }
});

// 处理"不保存"操作 — 回滚主题并继续导航
function handleDiscard() {
  const savedGeneral = settingsStore.savedSettings.general;
  themeStore.setMode(savedGeneral.themeMode);
  themeStore.setColor(savedGeneral.themeColor);

  settingsStore.discard();
  showUnsavedWarning.value = false;

  if (pendingNavigation) {
    pendingNavigation();
    pendingNavigation = null;
  }
}

onMounted(async () => {
  await settingsStore.init();
  // 用 themeStore 中的当前主题值同步 settingsStore（即时提交，不产生脏状态）
  settingsStore.commitGeneralSettings({
    themeMode: themeStore.mode,
    themeColor: themeStore.color,
  });
  // 等待 DOM 更新后设置 scroll spy（BasePanel 可能需要一帧来完成布局）
  await nextTick();
  setupScrollSpy();
});

onUnmounted(() => {
  if (observer) {
    observer.disconnect();
    observer = null;
  }
});
</script>

<style scoped>
/* 设置小节通用样式 */
:deep(.settings-section) {
  scroll-margin-top: 24px;
}

:deep(.settings-section__title) {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 4px;
}

:deep(.settings-section__desc) {
  font-size: 0.875rem;
  color: var(--text-secondary-color);
  margin-bottom: 20px;
}

:deep(.settings-section__subtitle) {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 8px;
}

:deep(.settings-field) {
  padding: 12px 16px;
  border-radius: var(--radius-sm, 8px);
  background: var(--surface-secondary-color);
}

:deep(.settings-field__label) {
  display: block;
  font-size: 0.8rem;
  color: var(--text-tertiary-color);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

:deep(.settings-field__value) {
  font-size: 0.875rem;
  color: var(--text-color);
}

/* 自定义滚动条 */
:deep(.overflow-y-auto) {
  scrollbar-width: thin;
  scrollbar-color: var(--border-subtle-color) transparent;
}

:deep(.overflow-y-auto::-webkit-scrollbar) {
  width: 6px;
}

:deep(.overflow-y-auto::-webkit-scrollbar-track) {
  background: transparent;
}

:deep(.overflow-y-auto::-webkit-scrollbar-thumb) {
  background-color: var(--border-subtle-color);
  border-radius: 9999px;
}

:deep(.overflow-y-auto::-webkit-scrollbar-thumb:hover) {
  background-color: var(--border-color);
}
</style>
