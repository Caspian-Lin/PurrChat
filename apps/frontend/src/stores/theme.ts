import { defineStore } from 'pinia';
import { ref, watch } from 'vue';
import type { ThemeConfig, ThemeMode, ThemeColor } from '../config/theme';
import { defaultThemeConfig, themeColors, lightTheme, darkTheme, legacyColorMap } from '../config/theme';

const THEME_STORAGE_KEY = 'purr-chat-theme';

// 将 hex 颜色转为 rgba 字符串
function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

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
        if (config.color) {
          // 向后兼容：将旧颜色名映射到新颜色名
          if (themeColors[config.color as ThemeColor]) {
            color.value = config.color as ThemeColor;
          } else if (legacyColorMap[config.color]) {
            color.value = legacyColorMap[config.color];
          }
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

    // 设置 Tailwind 深色主题类
    if (mode.value === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }

    // 设置主题色
    root.style.setProperty('--theme-primary', colorConfig.primary);
    root.style.setProperty('--theme-secondary', colorConfig.secondary);
    root.style.setProperty('--theme-gradient', colorConfig.primary);

    // 设置表面色
    root.style.setProperty('--background-color', theme.background);
    root.style.setProperty('--surface-color', theme.surface);
    root.style.setProperty('--surface-hover-color', theme.surfaceHover);
    root.style.setProperty('--surface-secondary-color', theme.surfaceSecondary);
    root.style.setProperty('--surface-tertiary-color', theme.surfaceTertiary);
    root.style.setProperty('--strong-background-color',
      mode.value === 'light' ? '#FFFFFF' : '#22222C'
    );

    // 设置文本色
    root.style.setProperty('--text-color', theme.text);
    root.style.setProperty('--text-secondary-color', theme.textSecondary);
    root.style.setProperty('--text-tertiary-color', theme.textTertiary);

    // 设置边框色
    root.style.setProperty('--border-color', theme.border);
    root.style.setProperty('--border-subtle-color',
      mode.value === 'light' ? '#E7E5E0' : '#1F1F2A'
    );

    // 设置其他变量
    root.style.setProperty('--shadow-color', theme.shadow);
    root.style.setProperty('--modal-overlay-color', theme.modalOverlay);
    root.style.setProperty('--input-background', theme.inputBackground);
    root.style.setProperty('--input-border', theme.inputBorder);
    root.style.setProperty('--card-background', theme.cardBackground);
    root.style.setProperty('--hover-background', theme.hoverBackground);

    // 动态混入主题色到 sent 气泡和选中态
    const tintColor = colorConfig.primary;
    root.style.setProperty(
      '--message-sent-background',
      hexToRgba(tintColor, mode.value === 'light' ? 0.08 : 0.15)
    );
    root.style.setProperty(
      '--selected-background',
      hexToRgba(tintColor, mode.value === 'light' ? 0.1 : 0.15)
    );

    // 固定接收消息背景
    root.style.setProperty(
      '--message-received-background',
      mode.value === 'light' ? '#FFFFFF' : '#2C2C3A'
    );

    // 设置语义色
    root.style.setProperty('--color-success', theme.success);
    root.style.setProperty('--color-success-bg', theme.successBackground);
    root.style.setProperty('--color-warning', theme.warning);
    root.style.setProperty('--color-warning-bg', theme.warningBackground);
    root.style.setProperty('--color-error', theme.error);
    root.style.setProperty('--color-error-bg', theme.errorBackground);
    root.style.setProperty('--color-info', theme.info);
    root.style.setProperty('--color-info-bg', theme.infoBackground);

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
