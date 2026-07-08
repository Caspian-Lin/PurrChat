import appConfig, { type AppClient, type AppEnv } from '../config/app';
import type {
  DeviceType,
  LayoutMode,
  NotificationPermissionState,
  OperatingSystem,
  PlatformCapabilities,
  PointerKind,
  ViewportClass,
} from './types';

const COMPACT_VIEWPORT_MAX = 767;
const MEDIUM_VIEWPORT_MAX = 1023;
const DEFAULT_VIEWPORT_WIDTH = 1200;
const DEFAULT_VIEWPORT_HEIGHT = 800;

interface TauriWindow extends Window {
  __TAURI_INTERNALS__?: unknown;
}

interface NavigatorWithUserAgentData extends Navigator {
  userAgentData?: {
    platform?: string;
    mobile?: boolean;
  };
}

export interface PlatformDetectionInput {
  env?: AppEnv;
  client?: AppClient;
  userAgent?: string;
  userAgentPlatform?: string;
  userAgentMobile?: boolean;
  maxTouchPoints?: number;
  hasTauriInternals?: boolean;
  width?: number;
  height?: number;
  pointerCoarse?: boolean;
  pointerFine?: boolean;
  canHover?: boolean;
  hasNotificationApi?: boolean;
  notificationPermission?: NotificationPermissionState;
  hasClipboardApi?: boolean;
  online?: boolean;
  visible?: boolean;
}

function queryMedia(query: string): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false;
  return window.matchMedia(query).matches;
}

export function readBrowserPlatformInput(): PlatformDetectionInput {
  const nav =
    typeof navigator === 'undefined' ? undefined : (navigator as NavigatorWithUserAgentData);
  const win = typeof window === 'undefined' ? undefined : (window as TauriWindow);
  const doc = typeof document === 'undefined' ? undefined : document;
  const notificationPermission =
    typeof Notification === 'undefined'
      ? 'unsupported'
      : (Notification.permission as NotificationPermissionState);

  return {
    env: appConfig.env,
    client: appConfig.client,
    userAgent: nav?.userAgent,
    userAgentPlatform: nav?.userAgentData?.platform ?? nav?.platform,
    userAgentMobile: nav?.userAgentData?.mobile,
    maxTouchPoints: nav?.maxTouchPoints ?? 0,
    hasTauriInternals: Boolean(win?.__TAURI_INTERNALS__),
    width: win?.innerWidth,
    height: win?.innerHeight,
    pointerCoarse: queryMedia('(pointer: coarse)'),
    pointerFine: queryMedia('(pointer: fine)'),
    canHover: queryMedia('(hover: hover)'),
    hasNotificationApi: typeof Notification !== 'undefined',
    notificationPermission,
    hasClipboardApi: Boolean(nav?.clipboard),
    online: nav?.onLine ?? true,
    visible: doc?.visibilityState !== 'hidden',
  };
}

function detectOs(input: PlatformDetectionInput): OperatingSystem {
  const platform = input.userAgentPlatform?.toLowerCase() ?? '';
  const ua = input.userAgent?.toLowerCase() ?? '';
  const source = `${platform} ${ua}`;

  if (source.includes('android')) return 'android';
  if (
    source.includes('iphone') ||
    source.includes('ipad') ||
    source.includes('ipod') ||
    source.includes('ios') ||
    (platform.includes('mac') && (input.maxTouchPoints ?? 0) > 1)
  ) {
    return 'ios';
  }
  if (source.includes('win')) return 'windows';
  if (source.includes('mac')) return 'macos';
  if (source.includes('linux') || source.includes('x11')) return 'linux';

  return 'unknown';
}

function getViewportClass(width: number): ViewportClass {
  if (width <= COMPACT_VIEWPORT_MAX) return 'compact';
  if (width <= MEDIUM_VIEWPORT_MAX) return 'medium';
  return 'expanded';
}

function getPointerKind(input: PlatformDetectionInput, hasTouch: boolean): PointerKind {
  if (input.pointerCoarse && input.pointerFine) return 'hybrid';
  if (input.pointerCoarse || (hasTouch && !input.pointerFine)) return 'coarse';
  if (input.pointerFine) return 'fine';
  return 'unknown';
}

function getDeviceType(
  os: OperatingSystem,
  input: PlatformDetectionInput,
  viewportClass: ViewportClass
): DeviceType {
  const mobileHint = input.client === 'mobile' || input.userAgentMobile === true;
  const mobileOs = os === 'android' || os === 'ios';

  if (!mobileHint && !mobileOs) return 'desktop';
  if (viewportClass === 'compact') return 'phone';
  return 'tablet';
}

function getLayoutMode(deviceType: DeviceType): LayoutMode {
  if (deviceType === 'phone') return 'mobile';
  if (deviceType === 'tablet') return 'tablet';
  return 'desktop';
}

export function detectPlatform(input: PlatformDetectionInput = readBrowserPlatformInput()) {
  const env = input.env ?? appConfig.env;
  const client = input.client ?? appConfig.client;
  const runtimeKind = client === 'tauri' || input.hasTauriInternals ? 'tauri' : 'web';
  const osName = detectOs(input);
  const osIsMobile = osName === 'android' || osName === 'ios';
  const osIsDesktop = osName === 'windows' || osName === 'macos' || osName === 'linux';
  const width = input.width ?? DEFAULT_VIEWPORT_WIDTH;
  const height = input.height ?? DEFAULT_VIEWPORT_HEIGHT;
  const viewportClass = getViewportClass(width);
  const maxTouchPoints = input.maxTouchPoints ?? 0;
  const hasTouch = maxTouchPoints > 0 || input.pointerCoarse === true;
  const pointer = getPointerKind(input, hasTouch);
  const deviceType = getDeviceType(osName, input, viewportClass);
  const layoutMode = getLayoutMode(deviceType);
  const isNative = runtimeKind === 'tauri';
  const nativeDesktop = isNative && (osIsDesktop || osName === 'unknown');
  const nativeMobile = isNative && osIsMobile;

  return {
    runtime: {
      kind: runtimeKind,
      client,
      env,
      isNative,
      isWeb: runtimeKind === 'web',
    },
    os: {
      name: osName,
      isDesktop: osIsDesktop,
      isMobile: osIsMobile,
    },
    viewport: {
      width,
      height,
      class: viewportClass,
      isCompact: viewportClass === 'compact',
      isMedium: viewportClass === 'medium',
      isExpanded: viewportClass === 'expanded',
    },
    input: {
      hasTouch,
      pointer,
      canHover: input.canHover === true,
    },
    window: {
      layoutMode,
      deviceType,
      canResize: deviceType === 'desktop',
      hasSafeArea: osIsMobile,
      supportsMultiWindow: nativeDesktop,
    },
    files: {
      webFileInput: true,
      webDownload: true,
      dragAndDrop: deviceType !== 'phone',
      nativeOpenDialog: isNative,
      nativeSaveDialog: isNative,
      nativeReveal: nativeDesktop,
    },
    notifications: {
      web: input.hasNotificationApi === true,
      native: isNative,
      permission: input.notificationPermission ?? 'unsupported',
    },
    tray: {
      supported: nativeDesktop,
    },
    clipboard: {
      writeText: input.hasClipboardApi === true || isNative,
      readText: input.hasClipboardApi === true || isNative,
      native: isNative,
    },
    lifecycle: {
      visibility: true,
      onlineStatus: true,
      nativeResume: isNative,
      deepLink: isNative,
    },
    haptics: {
      supported: nativeMobile || (hasTouch && deviceType !== 'desktop'),
    },
  } satisfies PlatformCapabilities;
}
