// 主题配置
export type ThemeMode = 'light' | 'dark';
export type ThemeColor = 'purple' | 'blue' | 'green' | 'orange' | 'red' | 'pink' | 'cyan';

export interface ThemeConfig {
  mode: ThemeMode;
  color: ThemeColor;
}

// 主题色配置
export const themeColors: Record<ThemeColor, { primary: string; secondary: string }> = {
  purple: {
    primary: '#bf5eff',
    secondary: '#764ba2',
  },
  blue: {
    primary: '#1d79cb',
    secondary: '#57a6eb',
  },
  green: {
    primary: '#70e874',
    secondary: '#92de63',
  },
  orange: {
    primary: '#FF8E42',
    secondary: '#fee140',
  },
  red: {
    primary: '#d43f3f',
    secondary: '#ee5a5a',
  },
  pink: {
    primary: '#f093fb',
    secondary: '#f5576c',
  },
  cyan: {
    primary: '#4facfe',
    secondary: '#00f2fe',
  },
};

// 浅色主题配色（暖灰色系，selected/messageSent 由 JS 动态混入主题色）
export const lightTheme = {
  background: '#faf9f7',
  surface: '#f0efec',
  surfaceHover: '#e8e6e3',
  surfaceSecondary: '#f5f4f1',
  surfaceTertiary: '#eae8e5',
  text: '#1a1a1e',
  textSecondary: '#4a4a52',
  textTertiary: '#8a8a94',
  border: '#e0ddd8',
  shadow: 'rgba(0, 0, 0, 0.08)',
  modalOverlay: 'rgba(0, 0, 0, 0.4)',
  inputBackground: '#f0efec',
  inputBorder: '#ddd9d3',
  cardBackground: '#ffffff',
  hoverBackground: '#e8e6e3',
  selectedBackground: '#eae8e5',
  messageSent: '#f0efec',
  messageReceived: '#ffffff',
  success: '#22c55e',
  successBackground: '#dcfce7',
  warning: '#f59e0b',
  warningBackground: '#fef3c7',
  error: '#ef4444',
  errorBackground: '#fef2f2',
  info: '#3b82f6',
  infoBackground: '#eff6ff',
};

// 深色主题配色（增强对比层次，selected/messageSent 由 JS 动态混入主题色）
export const darkTheme = {
  background: '#121212',
  surface: '#1e1e1e',
  surfaceHover: '#2a2a2a',
  surfaceSecondary: '#181818',
  surfaceTertiary: '#252525',
  text: '#e8e6e3',
  textSecondary: '#a8a6a3',
  textTertiary: '#6e6c69',
  border: '#333333',
  shadow: 'rgba(0, 0, 0, 0.4)',
  modalOverlay: 'rgba(0, 0, 0, 0.75)',
  inputBackground: '#1a1a1a',
  inputBorder: '#383838',
  cardBackground: '#1e1e1e',
  hoverBackground: '#2a2a2a',
  selectedBackground: '#252525',
  messageSent: '#252525',
  messageReceived: '#2d2d2d',
  success: '#22c55e',
  successBackground: '#1a2e1a',
  warning: '#f59e0b',
  warningBackground: '#2e2510',
  error: '#ef4444',
  errorBackground: '#2e1a1a',
  info: '#3b82f6',
  infoBackground: '#1a2530',
};

// 默认主题配置
export const defaultThemeConfig: ThemeConfig = {
  mode: 'light',
  color: 'purple',
};
