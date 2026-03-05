import { ref } from 'vue';
import { api } from '../models/api';
import { useMessage } from './useMessage';
import type { Friendship } from '../models/types';

export const useFriends = () => {
  const friends = ref<Friendship[]>([]);
  const message = useMessage();

  /**
   * 获取好友列表
   */
  const loadFriends = async () => {
    try {
      const response = await api.getFriends();
      if (response.success && response.data) {
        friends.value = response.data;
      }
    } catch (error) {
      console.error('Failed to load friends:', error);
    }
  };

  /**
   * 发送好友请求
   * @param targetUserId - 目标用户ID
   * @returns 是否发送成功
   */
  const sendFriendRequest = async (targetUserId: string): Promise<boolean> => {
    try {
      const response = await api.sendFriendRequest({
        target_user_id: targetUserId,
      });

      if (response.success) {
        message.success('好友请求已发送');
        return true;
      } else {
        message.error('发送好友请求失败');
        return false;
      }
    } catch (error) {
      console.error('Failed to send friend request:', error);
      message.error('发送好友请求失败');
      return false;
    }
  };

  /**
   * 处理好友请求
   * @param conversationId - 会话ID
   * @param action - 操作类型（accept 或 reject）
   * @returns 是否处理成功
   */
  const handleFriendRequest = async (
    conversationId: string,
    action: 'accept' | 'reject'
  ): Promise<boolean> => {
    try {
      const response = await api.handleFriendRequest({
        conversation_id: conversationId,
        action,
      });

      if (response.success) {
        const actionText = action === 'accept' ? '已接受' : '已拒绝';
        message.success(`好友请求${actionText}`);
        return true;
      } else {
        message.error('处理好友请求失败');
        return false;
      }
    } catch (error) {
      console.error('Failed to handle friend request:', error);
      message.error('处理好友请求失败');
      return false;
    }
  };

  return {
    friends,
    loadFriends,
    sendFriendRequest,
    handleFriendRequest,
  };
};
