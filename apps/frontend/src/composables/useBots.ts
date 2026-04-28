import { ref } from 'vue';
import { api } from '../models/api';
import { useBotStore } from '../stores/bot';
import { useNotification } from './useNotification';
import type {
  Bot,
  CreateBotRequest,
  UpdateBotRequest,
  DeployBotRequest,
  UpdateDeploymentStatusRequest,
  PublicBotDetail,
  DeployableConversation,
} from '../models/types';

export const useBots = () => {
  const notify = useNotification();
  const botStore = useBotStore();
  const loading = ref(false);

  // 创建 Bot
  async function createBot(data: CreateBotRequest): Promise<Bot | null> {
    loading.value = true;
    try {
      const response = await api.createBot(data);
      if (response.success && response.data) {
        notify.success('Bot 创建成功');
        await botStore.loadBots();
        return response.data;
      }
      notify.error(response.message || '创建 Bot 失败');
      return null;
    } catch (err) {
      console.error('[useBots] 创建 Bot 失败:', err);
      notify.error('创建 Bot 失败');
      return null;
    } finally {
      loading.value = false;
    }
  }

  // 更新 Bot
  async function updateBot(botId: string, data: UpdateBotRequest): Promise<boolean> {
    loading.value = true;
    try {
      const response = await api.updateBot(botId, data);
      if (response.success) {
        notify.success('Bot 已更新');
        await botStore.loadBots();
        return true;
      }
      notify.error(response.message || '更新 Bot 失败');
      return false;
    } catch (err) {
      console.error('[useBots] 更新 Bot 失败:', err);
      notify.error('更新 Bot 失败');
      return false;
    } finally {
      loading.value = false;
    }
  }

  // 删除 Bot
  async function deleteBot(botId: string): Promise<boolean> {
    loading.value = true;
    try {
      const response = await api.deleteBot(botId);
      if (response.success) {
        notify.success('Bot 已删除');
        if (botStore.activeBotId === botId) {
          botStore.setActiveBot(null);
        }
        await botStore.loadBots();
        return true;
      }
      notify.error(response.message || '删除 Bot 失败');
      return false;
    } catch (err) {
      console.error('[useBots] 删除 Bot 失败:', err);
      notify.error('删除 Bot 失败');
      return false;
    } finally {
      loading.value = false;
    }
  }

  // 部署 Bot 到会话
  async function deployBot(botId: string, conversationId: string): Promise<boolean> {
    try {
      const data: DeployBotRequest = { conversation_id: conversationId };
      const response = await api.deployBot(botId, data);
      if (response.success) {
        notify.success('Bot 已部署');
        await botStore.loadDeployments();
        return true;
      }
      notify.error(response.message || '部署 Bot 失败');
      return false;
    } catch (err) {
      console.error('[useBots] 部署 Bot 失败:', err);
      notify.error('部署 Bot 失败');
      return false;
    }
  }

  // 从会话移除 Bot
  async function undeployBot(botId: string, conversationId: string): Promise<boolean> {
    try {
      const response = await api.undeployBot(botId, conversationId);
      if (response.success) {
        notify.success('Bot 已移除');
        await botStore.loadDeployments();
        return true;
      }
      notify.error(response.message || '移除 Bot 失败');
      return false;
    } catch (err) {
      console.error('[useBots] 移除 Bot 失败:', err);
      notify.error('移除 Bot 失败');
      return false;
    }
  }

  // 更新部署状态
  async function updateDeploymentStatus(
    botId: string,
    data: UpdateDeploymentStatusRequest
  ): Promise<boolean> {
    try {
      const response = await api.updateDeploymentStatus(botId, data);
      if (response.success) {
        await botStore.loadDeployments();
        return true;
      }
      return false;
    } catch (err) {
      console.error('[useBots] 更新部署状态失败:', err);
      return false;
    }
  }

  // 创建与 Bot 的私聊会话
  async function createBotConversation(botId: string) {
    try {
      const response = await api.createBotConversation(botId);
      if (response.success && response.data) {
        return response.data;
      }
      notify.error(response.message || '创建对话失败');
      return null;
    } catch (err) {
      console.error('[useBots] 创建 Bot 对话失败:', err);
      notify.error('创建对话失败');
      return null;
    }
  }

  // 搜索公开 Bot（分页）
  async function searchBots(
    query: string,
    limit = 20,
    offset = 0
  ): Promise<{
    bots: PublicBotDetail[];
    total: number;
    hasMore: boolean;
  }> {
    try {
      const response = await api.searchBots(query, limit, offset);
      if (response.success && response.data) {
        const { bots, total } = response.data;
        return {
          bots,
          total,
          hasMore: offset + bots.length < total,
        };
      }
      return { bots: [], total: 0, hasMore: false };
    } catch (err) {
      console.error('[useBots] 搜索 Bot 失败:', err);
      return { bots: [], total: 0, hasMore: false };
    }
  }

  // 获取可部署 Bot 的群聊列表
  async function getDeployableConversations(botId: string): Promise<DeployableConversation[]> {
    try {
      const response = await api.getDeployableConversations(botId);
      if (response.success && response.data) {
        return response.data;
      }
      return [];
    } catch (err) {
      console.error('[useBots] 获取可部署会话失败:', err);
      return [];
    }
  }

  return {
    loading,
    createBot,
    updateBot,
    deleteBot,
    deployBot,
    undeployBot,
    updateDeploymentStatus,
    createBotConversation,
    searchBots,
    getDeployableConversations,
  };
};
