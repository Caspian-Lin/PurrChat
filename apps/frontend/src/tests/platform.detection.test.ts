import { describe, expect, it } from 'vitest';
import { detectPlatform } from '../platform/detection';

describe('platform detection', () => {
  it('keeps a narrow desktop window in desktop layout', () => {
    const platform = detectPlatform({
      env: 'production',
      client: 'web',
      userAgentPlatform: 'Win32',
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
      width: 390,
      height: 840,
      pointerFine: true,
      canHover: true,
      maxTouchPoints: 0,
      hasClipboardApi: true,
      hasNotificationApi: true,
      notificationPermission: 'default',
    });

    expect(platform.viewport.class).toBe('compact');
    expect(platform.window.deviceType).toBe('desktop');
    expect(platform.window.layoutMode).toBe('desktop');
    expect(platform.input.pointer).toBe('fine');
  });

  it('detects a phone from mobile client and compact viewport', () => {
    const platform = detectPlatform({
      env: 'production',
      client: 'mobile',
      userAgentPlatform: 'Linux armv8l',
      userAgent: 'Mozilla/5.0 (Linux; Android 14; Pixel)',
      width: 390,
      height: 844,
      pointerCoarse: true,
      maxTouchPoints: 5,
      hasNotificationApi: true,
      notificationPermission: 'granted',
    });

    expect(platform.os.name).toBe('android');
    expect(platform.window.deviceType).toBe('phone');
    expect(platform.window.layoutMode).toBe('mobile');
    expect(platform.window.hasSafeArea).toBe(true);
    expect(platform.haptics.supported).toBe(true);
  });

  it('separates tablet layout from phone layout', () => {
    const platform = detectPlatform({
      env: 'production',
      client: 'mobile',
      userAgentPlatform: 'iPad',
      userAgent: 'Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)',
      width: 900,
      height: 1180,
      pointerCoarse: true,
      maxTouchPoints: 5,
      hasNotificationApi: true,
      notificationPermission: 'default',
    });

    expect(platform.os.name).toBe('ios');
    expect(platform.viewport.class).toBe('medium');
    expect(platform.window.deviceType).toBe('tablet');
    expect(platform.window.layoutMode).toBe('tablet');
  });

  it('enables native desktop capability flags for Tauri desktop', () => {
    const platform = detectPlatform({
      env: 'production',
      client: 'tauri',
      userAgentPlatform: 'Win32',
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
      width: 1280,
      height: 800,
      pointerFine: true,
      canHover: true,
      maxTouchPoints: 0,
      hasTauriInternals: true,
      hasClipboardApi: true,
      hasNotificationApi: true,
      notificationPermission: 'default',
    });

    expect(platform.runtime.kind).toBe('tauri');
    expect(platform.runtime.isNative).toBe(true);
    expect(platform.files.nativeOpenDialog).toBe(true);
    expect(platform.files.nativeReveal).toBe(true);
    expect(platform.tray.supported).toBe(true);
    expect(platform.lifecycle.nativeResume).toBe(true);
  });
});
