import { describe, expect, it, vi } from 'vitest';
import {
  UnsupportedPlatformCapabilityError,
  createPlatformAdapters,
  createWebPersistenceAdapter,
} from '../platform';
import { detectPlatform } from '../platform/detection';

describe('platform adapters', () => {
  it('uses localStorage through the web persistence adapter', async () => {
    const storage = new Map<string, string>();
    const adapter = createWebPersistenceAdapter({
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
      key: (index: number) => Array.from(storage.keys())[index] ?? null,
      get length() {
        return storage.size;
      },
    } as Storage);

    await adapter.setItem('purr:a', '1');
    await adapter.setItem('other:b', '2');

    await expect(adapter.getItem('purr:a')).resolves.toBe('1');
    await expect(adapter.keys('purr:')).resolves.toEqual(['purr:a']);

    await adapter.clear('purr:');
    await expect(adapter.getItem('purr:a')).resolves.toBeNull();
    await expect(adapter.getItem('other:b')).resolves.toBe('2');
  });

  it('keeps secure credential storage unsupported for web defaults', async () => {
    const platform = detectPlatform({
      env: 'production',
      client: 'web',
      width: 1280,
      height: 800,
      hasNotificationApi: false,
      notificationPermission: 'unsupported',
      hasClipboardApi: false,
    });
    const adapters = createPlatformAdapters(platform);

    await expect(adapters.credentials.getSecret('refresh-token')).resolves.toBeNull();
    await expect(adapters.credentials.setSecret('refresh-token', 'secret')).rejects.toBeInstanceOf(
      UnsupportedPlatformCapabilityError
    );
  });

  it('falls back to textarea copy when Clipboard API is unavailable', async () => {
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: () => true,
    });
    const execCommand = vi.spyOn(document, 'execCommand').mockReturnValue(true);
    const originalClipboard = navigator.clipboard;

    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: undefined,
    });

    const adapters = createPlatformAdapters(
      detectPlatform({
        env: 'production',
        client: 'web',
        width: 1280,
        height: 800,
        hasClipboardApi: false,
      })
    );

    await adapters.clipboard.writeText('hello');

    expect(execCommand).toHaveBeenCalledWith('copy');

    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: originalClipboard,
    });
    execCommand.mockRestore();
  });
});
