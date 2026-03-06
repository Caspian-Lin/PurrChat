import { ref } from 'vue';
import { api } from '../models/api';
import type { Conversation } from '../models/types';

export const useConversations = () => {
  const conversations = ref<Conversation[]>([]);

  /**
   * 获取会话列表
   */
  const loadConversations = async () => {
    console.log('[useConversations] loadConversations 开始');
    try {
      const response = await api.getConversations();
      console.log('[useConversations] getConversations 响应', response);
      if (response.success && response.data) {
        conversations.value = response.data;
        console.log('[useConversations] 会话列表加载成功', conversations.value.length, '个会话');
      } else {
        console.log('[useConversations] 会话列表加载失败', response.message);
      }
    } catch (error) {
      console.error('[useConversations] Failed to load conversations:', error);
    }
  };

  /**
   * 创建会话
   * @param targetUserId - 目标用户ID
   * @returns 创建的会话对象或null
   */
  const createConversation = async (targetUserId: string) => {
    try {
      const response = await api.createConversation({
        target_user_id: targetUserId,
      });

      if (response.success && response.data) {
        await loadConversations();
        return conversations.value.find((c) => c.id === response.data?.id) || null;
      }
      return null;
    } catch (error) {
      console.error('Failed to create conversation:', error);
      return null;
    }
  };

  /**
   * 删除会话
   * @param conversationId - 会话ID
   * @returns 是否删除成功
   */
  const deleteConversation = async (conversationId: string) => {
    try {
      const response = await api.deleteConversation(conversationId);
      if (response.success) {
        await loadConversations();
        return true;
      }
      return false;
    } catch (error) {
      console.error('Failed to delete conversation:', error);
      return false;
    }
  };

  return {
    conversations,
    loadConversations,
    createConversation,
    deleteConversation,
  };
};
