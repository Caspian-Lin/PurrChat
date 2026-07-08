import type { AppClient, AppEnv } from '../config/app';

export type RuntimeKind = 'web' | 'tauri';
export type OperatingSystem = 'windows' | 'macos' | 'linux' | 'android' | 'ios' | 'unknown';
export type DeviceType = 'phone' | 'tablet' | 'desktop';
export type LayoutMode = 'mobile' | 'tablet' | 'desktop';
export type ViewportClass = 'compact' | 'medium' | 'expanded';
export type PointerKind = 'coarse' | 'fine' | 'hybrid' | 'unknown';
export type NotificationPermissionState = 'default' | 'granted' | 'denied' | 'unsupported';

export interface PlatformRuntime {
  kind: RuntimeKind;
  client: AppClient;
  env: AppEnv;
  isNative: boolean;
  isWeb: boolean;
}

export interface PlatformOs {
  name: OperatingSystem;
  isDesktop: boolean;
  isMobile: boolean;
}

export interface ViewportInfo {
  width: number;
  height: number;
  class: ViewportClass;
  isCompact: boolean;
  isMedium: boolean;
  isExpanded: boolean;
}

export interface InputCapabilities {
  hasTouch: boolean;
  pointer: PointerKind;
  canHover: boolean;
}

export interface WindowCapabilities {
  layoutMode: LayoutMode;
  deviceType: DeviceType;
  canResize: boolean;
  hasSafeArea: boolean;
  supportsMultiWindow: boolean;
}

export interface FileCapabilities {
  webFileInput: boolean;
  webDownload: boolean;
  dragAndDrop: boolean;
  nativeOpenDialog: boolean;
  nativeSaveDialog: boolean;
  nativeReveal: boolean;
}

export interface NotificationCapabilities {
  web: boolean;
  native: boolean;
  permission: NotificationPermissionState;
}

export interface TrayCapabilities {
  supported: boolean;
}

export interface ClipboardCapabilities {
  writeText: boolean;
  readText: boolean;
  native: boolean;
}

export interface LifecycleCapabilities {
  visibility: boolean;
  onlineStatus: boolean;
  nativeResume: boolean;
  deepLink: boolean;
}

export interface HapticsCapabilities {
  supported: boolean;
}

export interface PlatformCapabilities {
  runtime: PlatformRuntime;
  os: PlatformOs;
  viewport: ViewportInfo;
  input: InputCapabilities;
  window: WindowCapabilities;
  files: FileCapabilities;
  notifications: NotificationCapabilities;
  tray: TrayCapabilities;
  clipboard: ClipboardCapabilities;
  lifecycle: LifecycleCapabilities;
  haptics: HapticsCapabilities;
}

export interface PersistenceAdapter {
  getItem(_key: string): Promise<string | null>;
  setItem(_key: string, _value: string): Promise<void>;
  removeItem(_key: string): Promise<void>;
  keys(_prefix?: string): Promise<string[]>;
  clear(_prefix?: string): Promise<void>;
}

export interface FilePickerOptions {
  accept?: string;
  multiple?: boolean;
}

export interface FileAdapter {
  pickFiles(_options?: FilePickerOptions): Promise<File[]>;
  downloadBlob(_blob: Blob, _filename: string): Promise<void>;
  downloadUrl(_url: string, _filename?: string): Promise<void>;
}

export interface CredentialAdapter {
  getSecret(_key: string): Promise<string | null>;
  setSecret(_key: string, _value: string): Promise<void>;
  deleteSecret(_key: string): Promise<void>;
}

export interface SystemNotificationOptions {
  body?: string;
  tag?: string;
  icon?: string;
  data?: unknown;
}

export interface ShownNotification {
  close(): void;
}

export interface NotificationAdapter {
  getPermission(): Promise<NotificationPermissionState>;
  requestPermission(): Promise<NotificationPermissionState>;
  show(_title: string, _options?: SystemNotificationOptions): Promise<ShownNotification | null>;
}

export type LifecycleEventType =
  | 'focus'
  | 'blur'
  | 'visible'
  | 'hidden'
  | 'online'
  | 'offline'
  | 'resume';

export interface LifecycleEvent {
  type: LifecycleEventType;
  timestamp: number;
}

export interface LifecycleAdapter {
  subscribe(_handler: (_event: LifecycleEvent) => void): () => void;
  getSnapshot(): {
    visible: boolean;
    online: boolean;
  };
}

export interface ClipboardAdapter {
  writeText(_text: string): Promise<void>;
  readText(): Promise<string>;
}

export interface FeedbackAdapter {
  vibrate(_pattern: number | number[]): void;
}

export interface PlatformAdapters {
  persistence: PersistenceAdapter;
  files: FileAdapter;
  credentials: CredentialAdapter;
  notifications: NotificationAdapter;
  lifecycle: LifecycleAdapter;
  clipboard: ClipboardAdapter;
  feedback: FeedbackAdapter;
}
