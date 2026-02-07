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
    primary: '#899cf0',
    secondary: '#764ba2',
    gradient: 'linear-gradient(135deg, #899cf0 0%, #764ba2 100%)',
  },
  blue: {
    primary: '#1d79cb',
    secondary: '#57a6eb',
    gradient: 'linear-gradient(135deg, #1d79cb 0%, #57a6eb 100%)',
  },
  green: {
    primary: '#43da75',
    secondary: '#a4f86f',
    gradient: 'linear-gradient(135deg, #43da75 0%, #a4f86f 100%)',
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
  surface: '#f5f5f5',
  surfaceHover: '#e8e8e8',
  text: '#333333',
  textSecondary: '#666666',
  textTertiary: '#999999',
  border: '#e0e0e0',
  shadow: 'rgba(0, 0, 0, 0.1)',
  modalOverlay: 'rgba(0, 0, 0, 0.5)',
  inputBackground: '#ffffff',
  inputBorder: '#dddddd',
  messageSent: 'linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-secondary) 100%)',
  messageReceived: '#f0f0f0',
  cardBackground: '#fafafa',
};

// 深色主题配色
export const darkTheme = {
  background: '#1a1a1a',
  surface: '#2d2d2d',
  surfaceHover: '#3d3d3d',
  text: '#ffffff',
  textSecondary: '#b0b0b0',
  textTertiary: '#808080',
  border: '#404040',
  shadow: 'rgba(0, 0, 0, 0.3)',
  modalOverlay: 'rgba(0, 0, 0, 0.7)',
  inputBackground: '#2d2d2d',
  inputBorder: '#404040',
  messageSent: 'linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-secondary) 100%)',
  messageReceived: '#3d3d3d',
  cardBackground: '#252525',
};

// 默认主题配置
export const defaultThemeConfig: ThemeConfig = {
  mode: 'light',
  color: 'purple',
};
