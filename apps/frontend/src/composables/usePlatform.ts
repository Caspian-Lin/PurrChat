import { computed, ref } from 'vue';
import { detectPlatform, readBrowserPlatformInput, type PlatformCapabilities } from '../platform';

/**
 * 平台检测 composable
 * 分离运行时、设备类型和 viewport，避免窄桌面窗口被误判为移动端。
 * 使用懒初始化，避免模块级副作用
 */

const capabilities = ref<PlatformCapabilities>(detectPlatform(readBrowserPlatformInput()));
let initialized = false;
let cleanup: (() => void) | null = null;

function refreshCapabilities() {
  capabilities.value = detectPlatform(readBrowserPlatformInput());
}

function initialize() {
  if (initialized) return;
  initialized = true;

  refreshCapabilities();

  if (typeof window === 'undefined') return;

  window.addEventListener('resize', refreshCapabilities);
  window.addEventListener('online', refreshCapabilities);
  window.addEventListener('offline', refreshCapabilities);

  cleanup = () => {
    window.removeEventListener('resize', refreshCapabilities);
    window.removeEventListener('online', refreshCapabilities);
    window.removeEventListener('offline', refreshCapabilities);
  };
}

export function usePlatform() {
  initialize();

  const layoutMode = computed(() => capabilities.value.window.layoutMode);
  const deviceType = computed(() => capabilities.value.window.deviceType);
  const viewport = computed(() => capabilities.value.viewport);
  const isMobile = computed(() => layoutMode.value === 'mobile');
  const isTablet = computed(() => layoutMode.value === 'tablet');
  const isDesktop = computed(() => layoutMode.value === 'desktop');

  return {
    capabilities: computed(() => capabilities.value),
    deviceType,
    layoutMode,
    viewport,
    isMobile,
    isTablet,
    isDesktop,
    isCompactViewport: computed(() => capabilities.value.viewport.isCompact),
    isTauri: computed(() => capabilities.value.runtime.kind === 'tauri'),
    isNative: computed(() => capabilities.value.runtime.isNative),
    isTouch: computed(() => capabilities.value.input.hasTouch),
    refresh: refreshCapabilities,
  };
}

export function resetPlatformForTests() {
  cleanup?.();
  cleanup = null;
  initialized = false;
  refreshCapabilities();
}
