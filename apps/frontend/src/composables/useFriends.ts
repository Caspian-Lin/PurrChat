import { ref } from 'vue';
import { api } from '../models/api';
import { useNotification } from './useNotification';
import type { Friendship } from '../models/types';

export const useFriends = () => {
  const friends = ref<Friendship[]>([]);
  const pendingRequests = ref<Friendship[]>([]);
  const notify = useNotification();

  /**
   * 获取好友列表
   */
  const loadFriends = async () => {
    console.log('[useFriends] loadFriends 开始');
    try {
      const response = await api.getFriends();
      console.log('[useFriends] getFriends 响应', response);
      if (response.success && response.data) {
        friends.value = response.data;
        console.log('[useFriends] 好友列表加载成功', friends.value.length, '个好友');
      } else {
        console.log('[useFriends] 好友列表加载失败', response.message);
      }
    } catch (error) {
      console.error('[useFriends] Failed to load friends:', error);
    }
  };

  /**
   * 获取待处理的好友请求
   */
  const loadPendingRequests = async () => {
    console.log('[useFriends] loadPendingRequests 开始');
    try {
      const response = await api.getPendingFriendRequests();
      console.log('[useFriends] getPendingFriendRequests 响应', response);
      if (response.success && response.data) {
        pendingRequests.value = response.data;
        console.log('[useFriends] 待处理好友请求加载成功', pendingRequests.value.length, '个请求');
      } else {
        console.log('[useFriends] 待处理好友请求加载失败', response.message);
      }
    } catch (error) {
      console.error('[useFriends] Failed to load pending requests:', error);
    }
  };

  /**
   * 发送好友请求
   * @param targetUserId - 目标用户ID
   * @returns 是否发送成功
   */
  const sendFriendRequest = async (targetUserId: string): Promise<boolean> => {
    console.log('[useFriends] sendFriendRequest 开始', { targetUserId });
    try {
      const response = await api.sendFriendRequest({
        target_user_id: targetUserId,
      });

      if (response.success) {
        console.log('[useFriends] 好友请求发送成功');
        notify.success('好友请求已发送');
        return true;
      } else {
        console.log('[useFriends] 好友请求发送失败', response.message);
        notify.error('发送好友请求失败');
        return false;
      }
    } catch (error) {
      console.error('[useFriends] Failed to send friend request:', error);
      notify.error('发送好友请求失败');
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
    console.log('[useFriends] handleFriendRequest 开始', { conversationId, action });
    try {
      const response = await api.handleFriendRequest({
        conversation_id: conversationId,
        action,
      });

      if (response.success) {
        const actionText = action === 'accept' ? '已接受' : '已拒绝';
        console.log('[useFriends] 好友请求处理成功', action);
        notify.success(`好友请求${actionText}`);
        // 重新加载待处理请求列表
        await loadPendingRequests();
        return true;
      } else {
        console.log('[useFriends] 好友请求处理失败', response.message);
        notify.error('处理好友请求失败');
        return false;
      }
    } catch (error) {
      console.error('[useFriends] Failed to handle friend request:', error);
      notify.error('处理好友请求失败');
      return false;
    }
  };

  return {
    friends,
    pendingRequests,
    loadFriends,
    loadPendingRequests,
    sendFriendRequest,
    handleFriendRequest,
  };
};
