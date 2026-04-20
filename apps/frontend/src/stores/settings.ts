import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type {
  UserSettings,
  PanelVisibilitySettings,
  NotificationSettings,
  GeneralSettings,
} from '../models/types';
import { DEFAULT_SETTINGS, SETTINGS_STORAGE_KEY } from '../config/settings';
import { api } from '../models/api';

export const useSettingsStore = defineStore('settings', () => {
  // ===== 状态 =====

  // 当前（可能含未保存修改的）设置
  const settings = ref<UserSettings>(deepClone(DEFAULT_SETTINGS));
  // 上次保存的设置快照（dirty 检测基准）
  const savedSettings = ref<UserSettings>(deepClone(DEFAULT_SETTINGS));
  // 加载状态
  const isLoading = ref(false);
  // 保存状态
  const isSaving = ref(false);
  // 错误信息
  const error = ref<string | null>(null);

  // ===== 计算属性 =====

  const isDirty = computed(() => {
    return JSON.stringify(settings.value) !== JSON.stringify(savedSettings.value);
  });

  // ===== 工具函数 =====

  // 安全深拷贝：JSON 序列化可自动脱去 Vue reactive Proxy
  function deepClone<T>(obj: T): T {
    return JSON.parse(JSON.stringify(obj));
  }

  function deepMerge<T extends Record<string, any>>(target: T, source: Partial<T>): T {
    const result = { ...target };
    for (const key of Object.keys(source) as (keyof T)[]) {
      const srcVal = source[key];
      const tgtVal = target[key];
      if (
        srcVal &&
        typeof srcVal === 'object' &&
        !Array.isArray(srcVal) &&
        tgtVal &&
        typeof tgtVal === 'object' &&
        !Array.isArray(tgtVal)
      ) {
        result[key] = deepMerge(tgtVal as Record<string, any>, srcVal as Partial<typeof tgtVal>);
      } else if (srcVal !== undefined) {
        result[key] = srcVal;
      }
    }
    return result;
  }

  // ===== 初始化 =====

  // 从 localStorage 加载缓存
  const loadFromCache = (): boolean => {
    try {
      const cached = localStorage.getItem(SETTINGS_STORAGE_KEY);
      if (cached) {
        const parsed = JSON.parse(cached) as Partial<UserSettings>;
        settings.value = deepMerge(deepClone(DEFAULT_SETTINGS), parsed);
        savedSettings.value = deepClone(settings.value);
        return true;
      }
    } catch (e) {
      console.error('[settingsStore] Failed to load from cache:', e);
    }
    return false;
  };

  // 保存到 localStorage 缓存
  const saveToCache = () => {
    try {
      localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings.value));
    } catch (e) {
      console.error('[settingsStore] Failed to save to cache:', e);
    }
  };

  // 从服务端同步
  const fetchFromServer = async () => {
    isLoading.value = true;
    error.value = null;
    try {
      const response = await api.getSettings();
      if (response.success && response.data) {
        const serverSettings = response.data as Partial<UserSettings>;
        const merged = deepMerge(deepClone(DEFAULT_SETTINGS), serverSettings);
        settings.value = merged;
        savedSettings.value = deepClone(merged);
        saveToCache();
      }
    } catch (e: any) {
      error.value = e.response?.data?.message || '获取设置失败';
      console.error('[settingsStore] Failed to fetch from server:', e);
    } finally {
      isLoading.value = false;
    }
  };

  // 初始化：先缓存后同步
  const init = async () => {
    loadFromCache();
    await fetchFromServer();
  };

  // ===== 更新方法 =====

  const updatePanelSettings = (update: Partial<PanelVisibilitySettings>) => {
    settings.value.panels = { ...settings.value.panels, ...update };
  };

  const updateNotificationSettings = (update: Partial<NotificationSettings>) => {
    settings.value.notifications = { ...settings.value.notifications, ...update };
  };

  const updateGeneralSettings = (update: Partial<GeneralSettings>) => {
    settings.value.general = { ...settings.value.general, ...update };
  };

  // 即时提交：同时更新 settings、savedSettings 和本地缓存，不产生脏状态
  // 用于侧边栏主题切换等"即时生效、无需保存"的场景
  const commitGeneralSettings = (update: Partial<GeneralSettings>) => {
    settings.value.general = { ...settings.value.general, ...update };
    savedSettings.value.general = { ...savedSettings.value.general, ...update };
    saveToCache();
  };

  // ===== 保存到服务端 =====

  const save = async () => {
    if (!isDirty.value) return;

    isSaving.value = true;
    error.value = null;
    try {
      const response = await api.updateSettings({ settings: settings.value });
      if (response.success) {
        savedSettings.value = deepClone(settings.value);
        saveToCache();
      } else {
        error.value = response.message || '保存设置失败';
      }
    } catch (e: any) {
      error.value = e.response?.data?.message || '保存设置失败';
      console.error('[settingsStore] Failed to save:', e);
    } finally {
      isSaving.value = false;
    }
  };

  // ===== 丢弃修改 =====

  const discard = () => {
    settings.value = deepClone(savedSettings.value);
  };

  return {
    settings,
    savedSettings,
    isLoading,
    isSaving,
    isDirty,
    error,
    init,
    updatePanelSettings,
    updateNotificationSettings,
    updateGeneralSettings,
    commitGeneralSettings,
    save,
    discard,
  };
});
