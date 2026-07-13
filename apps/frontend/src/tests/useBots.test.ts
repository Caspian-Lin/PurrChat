import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useBots } from '../composables/useBots';
import { useNotification } from '../composables/useNotification';

vi.mock('../models/api', () => ({
  api: {
    updateBotInstallation: vi.fn(),
    createBotInstallation: vi.fn(),
    getBotDeployments: vi.fn(),
    getBots: vi.fn(),
  },
}));

import { api } from '../models/api';

const mockedApi = vi.mocked(api);

describe('useBots — installation error handling', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    useNotification().clearAllNotifications();
  });

  it('shows server-provided error message when updateInstallation fails', async () => {
    mockedApi.updateBotInstallation.mockResolvedValueOnce({
      success: false,
      code: 'granted_exceeds_requested',
      message: '授予的权限超出了 Bot 声明的权限范围',
    });

    const { updateInstallation } = useBots();
    const result = await updateInstallation('inst-1', {
      granted_capabilities: ['secrets:use'],
    });

    expect(result).toBeNull();
    const { notifications } = useNotification();
    expect(notifications.value).toHaveLength(1);
    expect(notifications.value[0].type).toBe('error');
    expect(notifications.value[0].message).toBe('授予的权限超出了 Bot 声明的权限范围');
  });

  it('shows forbidden message for unauthorized update', async () => {
    mockedApi.updateBotInstallation.mockResolvedValueOnce({
      success: false,
      code: 'forbidden',
      message: '无权管理此安装',
    });

    const { updateInstallation } = useBots();
    const result = await updateInstallation('inst-1', { status: 'paused' });

    expect(result).toBeNull();
    const { notifications } = useNotification();
    expect(notifications.value[0].message).toBe('无权管理此安装');
  });

  it('falls back to generic message when server provides none', async () => {
    mockedApi.updateBotInstallation.mockResolvedValueOnce({
      success: false,
    });

    const { updateInstallation } = useBots();
    await updateInstallation('inst-1', { status: 'paused' });

    const { notifications } = useNotification();
    expect(notifications.value[0].message).toBe('更新 Bot 权限失败');
  });

  it('returns installation and refreshes store on success', async () => {
    const installation = {
      id: 'inst-1',
      app_id: 'bot-1',
      installed_by: 'user-1',
      target_type: 'user' as const,
      target_id: 'user-1',
      granted_capabilities: ['messages:read_trigger'],
      status: 'active' as const,
      diagnostics_consent: 'denied' as const,
      installed_at: '2026-07-13T00:00:00Z',
      updated_at: '2026-07-13T00:00:00Z',
    };
    mockedApi.updateBotInstallation.mockResolvedValueOnce({
      success: true,
      data: { installation },
    });
    mockedApi.getBotDeployments.mockResolvedValueOnce({ success: true, data: [] });

    const { updateInstallation } = useBots();
    const result = await updateInstallation('inst-1', {
      granted_capabilities: ['messages:read_trigger'],
    });

    expect(result).toEqual(installation);
    expect(mockedApi.getBotDeployments).toHaveBeenCalled();
    const { notifications } = useNotification();
    expect(notifications.value[0].type).toBe('success');
  });
});
