import type {
  ClipboardAdapter,
  CredentialAdapter,
  FeedbackAdapter,
  FileAdapter,
  FilePickerOptions,
  LifecycleAdapter,
  LifecycleEvent,
  NotificationAdapter,
  NotificationPermissionState,
  PersistenceAdapter,
  PlatformAdapters,
  PlatformCapabilities,
  ShownNotification,
  SystemNotificationOptions,
} from './types';

export class UnsupportedPlatformCapabilityError extends Error {
  constructor(capability: string) {
    super(`Unsupported platform capability: ${capability}`);
    this.name = 'UnsupportedPlatformCapabilityError';
  }
}

function getStorageKeys(storage: Storage) {
  const keys: string[] = [];
  for (let index = 0; index < storage.length; index++) {
    const key = storage.key(index);
    if (key) keys.push(key);
  }
  return keys;
}

function getStorage(): Storage | null {
  if (typeof localStorage === 'undefined') return null;
  return localStorage;
}

export function createWebPersistenceAdapter(storage: Storage | null = getStorage()) {
  return {
    async getItem(key: string) {
      return storage?.getItem(key) ?? null;
    },
    async setItem(key: string, value: string) {
      if (!storage) throw new UnsupportedPlatformCapabilityError('persistence.localStorage');
      storage.setItem(key, value);
    },
    async removeItem(key: string) {
      storage?.removeItem(key);
    },
    async keys(prefix?: string) {
      if (!storage) return [];
      return getStorageKeys(storage).filter((key) => !prefix || key.startsWith(prefix));
    },
    async clear(prefix?: string) {
      if (!storage) return;
      if (!prefix) {
        storage.clear();
        return;
      }
      getStorageKeys(storage).forEach((key) => {
        if (key.startsWith(prefix)) storage.removeItem(key);
      });
    },
  } satisfies PersistenceAdapter;
}

function clickDownloadLink(url: string, filename?: string) {
  if (typeof document === 'undefined') {
    throw new UnsupportedPlatformCapabilityError('files.webDownload');
  }

  const link = document.createElement('a');
  link.href = url;
  if (filename) link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

export function createWebFileAdapter(fetcher: typeof fetch = fetch) {
  return {
    async pickFiles(options: FilePickerOptions = {}) {
      if (typeof document === 'undefined') {
        throw new UnsupportedPlatformCapabilityError('files.webFileInput');
      }

      return new Promise<File[]>((resolve) => {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = options.accept ?? '';
        input.multiple = options.multiple ?? false;
        input.style.display = 'none';
        input.addEventListener(
          'change',
          () => {
            const files = Array.from(input.files ?? []);
            input.remove();
            resolve(files);
          },
          { once: true }
        );
        document.body.appendChild(input);
        input.click();
      });
    },
    async downloadBlob(blob: Blob, filename: string) {
      const url = URL.createObjectURL(blob);
      try {
        clickDownloadLink(url, filename);
      } finally {
        URL.revokeObjectURL(url);
      }
    },
    async downloadUrl(url: string, filename?: string) {
      if (!filename) {
        clickDownloadLink(url);
        return;
      }

      const response = await fetcher(url);
      if (!response.ok) throw new Error('Failed to download file');
      const blob = await response.blob();
      await this.downloadBlob(blob, filename);
    },
  } satisfies FileAdapter;
}

export function createWebCredentialAdapter() {
  return {
    async getSecret() {
      return null;
    },
    async setSecret() {
      throw new UnsupportedPlatformCapabilityError('credentials.secureStorage');
    },
    async deleteSecret() {
      return;
    },
  } satisfies CredentialAdapter;
}

function getNotificationPermission(): NotificationPermissionState {
  if (typeof Notification === 'undefined') return 'unsupported';
  return Notification.permission as NotificationPermissionState;
}

export function createWebNotificationAdapter() {
  return {
    async getPermission() {
      return getNotificationPermission();
    },
    async requestPermission() {
      if (typeof Notification === 'undefined') return 'unsupported';
      return (await Notification.requestPermission()) as NotificationPermissionState;
    },
    async show(title: string, options?: SystemNotificationOptions) {
      if (typeof Notification === 'undefined') {
        throw new UnsupportedPlatformCapabilityError('notifications.web');
      }

      const permission = await this.getPermission();
      if (permission !== 'granted') return null;

      const notification = new Notification(title, options);
      return {
        close: () => notification.close(),
      } satisfies ShownNotification;
    },
  } satisfies NotificationAdapter;
}

function emitLifecycle(handler: (_event: LifecycleEvent) => void, type: LifecycleEvent['type']) {
  handler({ type, timestamp: Date.now() });
}

export function createWebLifecycleAdapter() {
  return {
    subscribe(handler: (_event: LifecycleEvent) => void) {
      if (typeof window === 'undefined' || typeof document === 'undefined') {
        return () => {};
      }

      const onFocus = () => emitLifecycle(handler, 'focus');
      const onBlur = () => emitLifecycle(handler, 'blur');
      const onVisibility = () =>
        emitLifecycle(handler, document.visibilityState === 'hidden' ? 'hidden' : 'visible');
      const onOnline = () => emitLifecycle(handler, 'online');
      const onOffline = () => emitLifecycle(handler, 'offline');

      window.addEventListener('focus', onFocus);
      window.addEventListener('blur', onBlur);
      window.addEventListener('online', onOnline);
      window.addEventListener('offline', onOffline);
      document.addEventListener('visibilitychange', onVisibility);

      return () => {
        window.removeEventListener('focus', onFocus);
        window.removeEventListener('blur', onBlur);
        window.removeEventListener('online', onOnline);
        window.removeEventListener('offline', onOffline);
        document.removeEventListener('visibilitychange', onVisibility);
      };
    },
    getSnapshot() {
      return {
        visible: typeof document === 'undefined' ? true : document.visibilityState !== 'hidden',
        online: typeof navigator === 'undefined' ? true : navigator.onLine,
      };
    },
  } satisfies LifecycleAdapter;
}

export function createWebClipboardAdapter() {
  return {
    async writeText(text: string) {
      if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
        return;
      }
      if (typeof document === 'undefined') {
        throw new UnsupportedPlatformCapabilityError('clipboard.writeText');
      }

      const textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.style.cssText = 'position:fixed;opacity:0';
      document.body.appendChild(textarea);
      textarea.select();
      if (!document.execCommand) {
        document.body.removeChild(textarea);
        throw new UnsupportedPlatformCapabilityError('clipboard.writeText');
      }
      document.execCommand('copy');
      document.body.removeChild(textarea);
    },
    async readText() {
      if (typeof navigator === 'undefined' || !navigator.clipboard?.readText) {
        throw new UnsupportedPlatformCapabilityError('clipboard.readText');
      }
      return navigator.clipboard.readText();
    },
  } satisfies ClipboardAdapter;
}

export function createWebFeedbackAdapter() {
  return {
    vibrate(pattern: number | number[]) {
      if (typeof navigator !== 'undefined' && navigator.vibrate) {
        navigator.vibrate(pattern);
      }
    },
  } satisfies FeedbackAdapter;
}

export function createPlatformAdapters(_capabilities: PlatformCapabilities) {
  return {
    persistence: createWebPersistenceAdapter(),
    files: createWebFileAdapter(),
    credentials: createWebCredentialAdapter(),
    notifications: createWebNotificationAdapter(),
    lifecycle: createWebLifecycleAdapter(),
    clipboard: createWebClipboardAdapter(),
    feedback: createWebFeedbackAdapter(),
  } satisfies PlatformAdapters;
}
