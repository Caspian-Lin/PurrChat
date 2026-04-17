// 主题配置 — PurrChat Design System: Soft Architecture
// Brand: Intimate · Refined · Alive

export type ThemeMode = 'light' | 'dark';
export type ThemeColor = 'sage' | 'iris' | 'ocean' | 'ember' | 'rose' | 'slate' | 'clay' | 'honey';

export interface ThemeConfig {
  mode: ThemeMode;
  color: ThemeColor;
}

// 主题色配置（低饱和矿物感）
// primary: 用于前景（文字、图标、active 状态），需满足 WCAG AA
// secondary: 用于背景色（选中项、标签、浅色装饰）
export const themeColors: Record<ThemeColor, { primary: string; secondary: string }> = {
  sage: {
    primary: '#4A7C3F',
    secondary: '#E8F0E5',
  },
  iris: {
    primary: '#7358A8',
    secondary: '#EDE8F5',
  },
  ocean: {
    primary: '#3A6D8C',
    secondary: '#E5EEF5',
  },
  ember: {
    primary: '#A86A2E',
    secondary: '#F5ECE0',
  },
  rose: {
    primary: '#A85A6E',
    secondary: '#F5E5E9',
  },
  slate: {
    primary: '#546478',
    secondary: '#E8EDF2',
  },
  clay: {
    primary: '#9E5A40',
    secondary: '#F5E5DE',
  },
  honey: {
    primary: '#8A7430',
    secondary: '#F5F0E0',
  },
};

// 旧颜色名映射（向后兼容已保存的 localStorage 配置）
export const legacyColorMap: Record<string, ThemeColor> = {
  purple: 'iris',
  blue: 'ocean',
  green: 'sage',
  orange: 'ember',
  red: 'clay',
  pink: 'rose',
  cyan: 'ocean',
};

// 浅色主题（温暖亚麻调）
export const lightTheme = {
  background: '#F7F5F2',
  surface: '#EEEAE5',
  surfaceHover: '#E2DDD7',
  surfaceSecondary: '#F4F1EC',
  surfaceTertiary: '#E8E4DE',
  text: '#1C1917',
  textSecondary: '#57534E',
  textTertiary: '#A8A29E',
  border: '#D6D3CE',
  shadow: 'rgba(28, 25, 23, 0.04)',
  modalOverlay: 'rgba(0, 0, 0, 0.4)',
  inputBackground: '#FFFFFF',
  inputBorder: '#D6D3CE',
  cardBackground: '#FFFFFF',
  hoverBackground: '#E2DDD7',
  selectedBackground: '#E2EDE0',
  messageSent: '#F0EDE8',
  messageReceived: '#FFFFFF',
  success: '#16A34A',
  successBackground: '#E8F5E9',
  warning: '#D97706',
  warningBackground: '#FFF8E1',
  error: '#DC2626',
  errorBackground: '#FEE2E2',
  info: '#2563EB',
  infoBackground: '#E3F2FD',
};

// 深色主题（蓝调暮色灰）
export const darkTheme = {
  background: '#101015',
  surface: '#161620',
  surfaceHover: '#282838',
  surfaceSecondary: '#161620',
  surfaceTertiary: '#1D1C26',
  text: '#ECEBE8',
  textSecondary: '#A1A0A8',
  textTertiary: '#65656E',
  border: '#2A2A36',
  shadow: 'rgba(0, 0, 0, 0.15)',
  modalOverlay: 'rgba(0, 0, 0, 0.6)',
  inputBackground: '#1E1E28',
  inputBorder: '#2A2A36',
  cardBackground: '#1E1E28',
  hoverBackground: '#282838',
  selectedBackground: '#1A2418',
  messageSent: '#1A1A22',
  messageReceived: '#2C2C3A',
  success: '#16A34A',
  successBackground: '#1A2E1A',
  warning: '#D97706',
  warningBackground: '#2E2510',
  error: '#DC2626',
  errorBackground: '#2E1A1A',
  info: '#2563EB',
  infoBackground: '#1A2530',
};

// 默认主题配置
export const defaultThemeConfig: ThemeConfig = {
  mode: 'light',
  color: 'sage',
};
