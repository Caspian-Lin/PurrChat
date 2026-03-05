// 主题配置
export type ThemeMode = 'light' | 'dark';
export type ThemeColor = 'purple' | 'blue' | 'green' | 'orange' | 'red' | 'pink' | 'cyan';

export interface ThemeConfig {
  mode: ThemeMode;
  color: ThemeColor;
}

// 主题色配置
export const themeColors: Record<
  ThemeColor,
  { primary: string; secondary: string; gradient: string }
> = {
  purple: {
    primary: '#bf5eff',
    secondary: '#764ba2',
    gradient: 'linear-gradient(135deg, #899cf0 0%, #764ba2 100%)',
  },
  blue: {
    primary: '#1d79cb',
    secondary: '#57a6eb',
    gradient: 'linear-gradient(135deg, #1d79cb 0%, #57a6eb 100%)',
  },
  green: {
    primary: '#70e874',
    secondary: '#92de63',
    gradient: 'linear-gradient(135deg, #70e874 0%, #92de63 100%)',
  },
  orange: {
    primary: '#fa709a',
    secondary: '#fee140',
    gradient: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
  },
  red: {
    primary: '#d43f3f',
    secondary: '#ee5a5a',
    gradient: 'linear-gradient(135deg, #d43f3f 0%, #ee5a5a 100%)',
  },
  pink: {
    primary: '#f093fb',
    secondary: '#f5576c',
    gradient: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  },
  cyan: {
    primary: '#4facfe',
    secondary: '#00f2fe',
    gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  },
};

// 浅色主题配色
export const lightTheme = {
  background: '#ffffff',
  surface: '#f9f9f9',
  surfaceHover: '#e8e8e8',
  surfaceSecondary: '#f4f4f4',
  surfaceTertiary: '#f2f2f2',
  text: '#000000',
  textSecondary: '#000000cc',
  textTertiary: '#00000099',
  border: '#d9d9d9',
  shadow: 'rgba(0, 0, 0, 0.1)',
  modalOverlay: 'rgba(0, 0, 0, 0.5)',
  inputBackground: '#f2f2f2',
  inputBorder: '#d9d9d9',
  messageSent: 'linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-secondary) 100%)',
  messageReceived: '#fffffffa',
  cardBackground: '#f9f9f9',
  hoverBackground: '#f2f2f2',
  selectedBackground: '#e8f4fd',
};

// 深色主题配色
export const darkTheme = {
  background: '#1a1a1a',
  surface: '#2d2d2d',
  surfaceHover: '#3d3d3d',
  surfaceSecondary: '#252525',
  surfaceTertiary: '#1f1f1f',
  text: '#ffffff',
  textSecondary: '#e0e0e0',
  textTertiary: '#b0b0b0',
  border: '#404040',
  shadow: 'rgba(0, 0, 0, 0.3)',
  modalOverlay: 'rgba(0, 0, 0, 0.7)',
  inputBackground: '#1f1f1f',
  inputBorder: '#404040',
  messageSent: 'linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-secondary) 100%)',
  messageReceived: '#3d3d3d',
  cardBackground: '#2d2d2d',
  hoverBackground: '#3d3d3d',
  selectedBackground: '#1e3a5f',
};

// 默认主题配置
export const defaultThemeConfig: ThemeConfig = {
  mode: 'light',
  color: 'purple',
};
