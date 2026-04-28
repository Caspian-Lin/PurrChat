import { ref, computed } from 'vue';
import appConfig from '../config/app';

/**
 * 平台检测 composable
 * 基于运行时环境 + appConfig.client 综合判断
 * 使用懒初始化，避免模块级副作用
 */

const isMobile = ref(false);
let initialized = false;

function detectMobile(): boolean {
  // 1. 环境变量显式声明
  if (appConfig.client === 'mobile') return true;

  // 2. Tauri Mobile 运行时检测
  //    Tauri 2 Mobile 在 Android WebView 中运行，__TAURI_INTERNALS__ 存在
  //    且 userAgent 包含 'wv' (WebView) 或 'Mobile'
  if (appConfig.isTauri && typeof navigator !== 'undefined') {
    const ua = navigator.userAgent.toLowerCase();
    if (ua.includes('android') || ua.includes('mobile') || ua.includes('wv')) {
      return true;
    }
  }

  // 3. 响应式视口检测（小屏幕设备）
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(max-width: 768px)').matches;
  }

  return false;
}

function initialize() {
  if (initialized) return;
  initialized = true;

  // 初始化检测
  isMobile.value = detectMobile();

  // 监听窗口尺寸变化（桌面端调整窗口大小时也能响应）
  if (typeof window !== 'undefined' && window.matchMedia) {
    const mql = window.matchMedia('(max-width: 768px)');
    mql.addEventListener('change', (e) => {
      isMobile.value = e.matches;
    });
  }
}

export function usePlatform() {
  initialize();

  const isDesktop = computed(() => !isMobile.value);

  return {
    isMobile: computed(() => isMobile.value),
    isDesktop,
  };
}
