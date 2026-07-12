/**
 * 应用配置管理
 * 根据环境变量自动选择合适的 API 配置
 */

// 环境类型
export type AppEnv = 'development' | 'production';
export type AppClient = 'web' | 'tauri' | 'mobile';

// 获取环境变量
const getEnvVar = (key: string, defaultValue: string = ''): string => {
  return import.meta.env[key] || defaultValue;
};

// 应用配置
export const appConfig = {
  // 环境标识
  env: (getEnvVar('VITE_APP_ENV', 'development') as AppEnv) || 'development',
  client: (getEnvVar('VITE_APP_CLIENT', 'web') as AppClient) || 'web',

  // API 基础 URL
  get apiBaseUrl(): string {
    const baseUrl = getEnvVar('VITE_API_BASE_URL', 'http://localhost:8080');
    return baseUrl;
  },

  // 是否为开发环境
  get isDevelopment(): boolean {
    return this.env === 'development';
  },

  // 是否为生产环境
  get isProduction(): boolean {
    return this.env === 'production';
  },

  // 客户端类型判断
  get isWeb(): boolean {
    return this.client === 'web';
  },

  get isTauri(): boolean {
    return this.client === 'tauri';
  },

  get isMobile(): boolean {
    return this.client === 'mobile';
  },
};

// 获取完整的 API 基础 URL（处理相对路径）
export const getApiBaseUrl = (): string => {
  const baseUrl = appConfig.apiBaseUrl;

  // 如果是相对路径，使用当前协议和主机
  if (baseUrl.startsWith('/')) {
    const protocol = window.location.protocol;
    const host = window.location.host;
    return `${protocol}//${host}${baseUrl}`;
  }

  return baseUrl;
};

// 获取存储服务 API 基础 URL
// 生产环境通过 nginx 代理（/api/files/ → storage），与 API Base URL 相同
// 开发环境需要单独配置（如 http://localhost:8081）
export const getStorageApiBaseUrl = (): string => {
  const storageUrl = getEnvVar('VITE_STORAGE_BASE_URL', '');
  if (storageUrl) return storageUrl;
  return getApiBaseUrl();
};

// 获取 Bot 微服务 URL（可选，用于 XState 引擎调试）
export const getBotEngineUrl = (): string => {
  return getEnvVar('VITE_BOT_ENGINE_URL', '');
};

// 获取 WebSocket URL（通过 Cookie/子协议认证，不传递 token 或 user_id）
export const getWebSocketUrl = (): string => {
  const baseUrl = appConfig.apiBaseUrl;
  let wsUrl: string;

  if (baseUrl.startsWith('/')) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const basePath = baseUrl === '/' ? '' : baseUrl;
    wsUrl = `${protocol}//${host}${basePath}/api/ws`;
  } else {
    const url = new URL(baseUrl);
    const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = url.host;
    wsUrl = `${protocol}//${host}/api/ws`;
  }

  return wsUrl;
};

// 日志配置
export const logger = {
  log: (...args: any[]) => {
    if (appConfig.isDevelopment) {
      console.log('[App]', ...args);
    }
  },
  warn: (...args: any[]) => {
    console.warn('[App]', ...args);
  },
  error: (...args: any[]) => {
    console.error('[App]', ...args);
  },
  info: (...args: any[]) => {
    if (appConfig.isDevelopment) {
      console.info('[App]', ...args);
    }
  },
};

// 导出配置对象
export default appConfig;
