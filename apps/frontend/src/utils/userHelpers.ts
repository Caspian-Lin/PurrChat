import type { User, Conversation } from '../models/types';

/**
 * 安全地获取用户头像URL
 * @param user - 用户对象
 * @returns 头像URL或undefined
 */
export const getUserAvatar = (user: User | undefined): string | undefined => {
  return user?.avatar_url;
};

/**
 * 安全地获取用户昵称
 * @param user - 用户对象
 * @returns 用户昵称或默认值
 */
export const getUserUsername = (user: User | undefined): string => {
  return user?.username || '未知用户';
};

/**
 * 获取会话中的对方用户
 * @param conversation - 会话对象
 * @param currentUserId - 当前用户ID
 * @returns 对方用户或undefined
 */
export const getOtherUser = (
  conversation: Conversation,
  currentUserId: string | undefined
): User | undefined => {
  if (!currentUserId || !conversation.members) return undefined;
  // 在members数组中找到不是当前用户的成员
  return conversation.members.find((m) => m.user_id !== currentUserId)?.user;
};

/**
 * 格式化好友状态
 * @param status - 好友状态
 * @returns 格式化后的状态文本
 */
export const formatFriendshipStatus = (status: string): string => {
  const statusMap: Record<string, string> = {
    pending: '等待验证',
    accepted: '已添加',
    blocked: '已拉黑',
  };
  return statusMap[status] || status;
};

/**
 * 获取好友状态的颜色
 * @param status - 好友状态
 * @returns 对应的CSS类名
 */
export const getFriendshipStatusColor = (status: string): string => {
  const colorMap: Record<string, string> = {
    pending: 'text-orange-500',
    accepted: 'text-green-500',
    blocked: 'text-red-500',
  };
  return colorMap[status] || 'text-gray-500';
};
