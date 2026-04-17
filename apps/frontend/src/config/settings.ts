import type { UserSettings } from '../models/types';
import pkg from '../../package.json';

export const APP_VERSION = pkg.version;

export const DEFAULT_SETTINGS: UserSettings = {
  panels: {
    visiblePanels: ['chat', 'friends', 'ai'],
  },
  notifications: {
    messageNotification: true,
    friendRequestNotification: true,
    groupInviteNotification: true,
    systemNotification: true,
    soundEnabled: true,
    desktopNotificationEnabled: false,
  },
  general: {
    themeMode: 'light',
    themeColor: 'sage',
    language: 'zh-CN',
    fontSize: 'medium',
  },
};

export const SETTINGS_STORAGE_KEY = 'purr-chat-settings';
