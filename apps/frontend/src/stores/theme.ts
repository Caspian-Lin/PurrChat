import { defineStore } from 'pinia';
import { ref, watch } from 'vue';
import type { ThemeConfig, ThemeMode, ThemeColor } from '../config/theme';
import { defaultThemeConfig, themeColors, lightTheme, darkTheme } from '../config/theme';

const THEME_STORAGE_KEY = 'purr-chat-theme';

export const useThemeStore = defineStore('theme', () => {
  // 状态
  const mode = ref<ThemeMode>(defaultThemeConfig.mode);
  const color = ref<ThemeColor>(defaultThemeConfig.color);

  // 从 localStorage 加载主题配置
  const loadTheme = () => {
    try {
      const saved = localStorage.getItem(THEME_STORAGE_KEY);
      if (saved) {
        const config = JSON.parse(saved) as ThemeConfig;
        if (config.mode && (config.mode === 'light' || config.mode === 'dark')) {
          mode.value = config.mode;
        }
        if (config.color && themeColors[config.color]) {
          color.value = config.color;
        }
      }
    } catch (error) {
      console.error('Failed to load theme:', error);
    }
  };

  // 保存主题配置到 localStorage
  const saveTheme = () => {
    try {
      const config: ThemeConfig = {
        mode: mode.value,
        color: color.value,
      };
      localStorage.setItem(THEME_STORAGE_KEY, JSON.stringify(config));
    } catch (error) {
      console.error('Failed to save theme:', error);
    }
  };

  // 切换主题模式
  const toggleMode = () => {
    mode.value = mode.value === 'light' ? 'dark' : 'light';
    applyTheme();
  };

  // 设置主题模式
  const setMode = (newMode: ThemeMode) => {
    mode.value = newMode;
    applyTheme();
  };

  // 设置主题色
  const setColor = (newColor: ThemeColor) => {
    color.value = newColor;
    applyTheme();
  };

  // 应用主题到 DOM
  const applyTheme = () => {
    const root = document.documentElement;
    const theme = mode.value === 'light' ? lightTheme : darkTheme;
    const colorConfig = themeColors[color.value];

    // 设置主题模式
    root.setAttribute('data-theme', mode.value);

    // 设置主题色
    root.style.setProperty('--theme-primary', colorConfig.primary);
    root.style.setProperty('--theme-secondary', colorConfig.secondary);
    root.style.setProperty('--theme-gradient', colorConfig.gradient);

    // 设置主题变量
    root.style.setProperty('--background-color', theme.background);
    root.style.setProperty('--surface-color', theme.surface);
    root.style.setProperty('--surface-hover-color', theme.surfaceHover);
    root.style.setProperty('--text-color', theme.text);
    root.style.setProperty('--text-secondary-color', theme.textSecondary);
    root.style.setProperty('--text-tertiary-color', theme.textTertiary);
    root.style.setProperty('--border-color', theme.border);
    root.style.setProperty('--shadow-color', theme.shadow);
    root.style.setProperty('--modal-overlay-color', theme.modalOverlay);
    root.style.setProperty('--input-background', theme.inputBackground);
    root.style.setProperty('--input-border', theme.inputBorder);
    root.style.setProperty('--message-sent-background', theme.messageSent);
    root.style.setProperty('--message-received-background', theme.messageReceived);
    root.style.setProperty('--card-background', theme.cardBackground);

    // 保存到 localStorage
    saveTheme();
  };

  // 初始化主题
  const initTheme = () => {
    loadTheme();
    applyTheme();
  };

  // 监听主题变化
  watch([mode, color], () => {
    applyTheme();
  });

  return {
    mode,
    color,
    toggleMode,
    setMode,
    setColor,
    applyTheme,
    initTheme,
  };
});
