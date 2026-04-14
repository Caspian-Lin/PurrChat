/**
 * 本地存储命名空间工具
 * 所有需要按用户隔离的业务数据，统一通过此模块生成 storage key
 */

// 获取当前登录用户的 ID
export function getCurrentUserId(): string | null {
  const userStr = localStorage.getItem('user');
  if (!userStr) return null;
  try {
    const user = JSON.parse(userStr);
    return user.id || null;
  } catch {
    return null;
  }
}

// ===== 消息缓存相关 =====

/** 消息缓存 key 前缀：msg_{userId}_ */
export function messageKeyPrefix(userId: string): string {
  return `msg_${userId}_`;
}

/** 消息缓存 key：msg_{userId}_{conversationId} */
export function messageKey(userId: string, conversationId: string): string {
  return `msg_${userId}_${conversationId}`;
}

/** 加密密钥 key：msg_key_{userId} */
export function messageEncryptionKey(userId: string): string {
  return `msg_key_${userId}`;
}

// ===== 会话状态缓存相关 =====

/** 会话状态 key 前缀：conv_state_{userId}_ */
export function convStateKeyPrefix(userId: string): string {
  return `conv_state_${userId}_`;
}

/** 会话状态 key：conv_state_{userId}_{conversationId} */
export function convStateKey(userId: string, conversationId: string): string {
  return `conv_state_${userId}_${conversationId}`;
}

// ===== AI 相关 =====

/** AI 配置 key：ai_cfg_{userId} */
export function aiConfigsKey(userId: string): string {
  return `ai_cfg_${userId}`;
}

/** AI 会话 key：ai_conv_{userId} */
export function aiConversationsKey(userId: string): string {
  return `ai_conv_${userId}`;
}

/** AI 激活配置 key：ai_act_cfg_{userId} */
export function aiActiveConfigKey(userId: string): string {
  return `ai_act_cfg_${userId}`;
}

/** AI 激活会话 key：ai_act_conv_{userId} */
export function aiActiveConversationKey(userId: string): string {
  return `ai_act_conv_${userId}`;
}

// ===== 数据管理 =====

/** 清理指定用户的所有缓存数据 */
export function clearUserData(userId: string) {
  const keys = Object.keys(localStorage);
  keys.forEach((key) => {
    if (key.startsWith(`msg_${userId}_`) || key === `msg_key_${userId}`) {
      localStorage.removeItem(key);
    }
    if (key.startsWith(`conv_state_${userId}_`)) {
      localStorage.removeItem(key);
    }
    if (
      key === aiConfigsKey(userId) ||
      key === aiConversationsKey(userId) ||
      key === aiActiveConfigKey(userId) ||
      key === aiActiveConversationKey(userId)
    ) {
      localStorage.removeItem(key);
    }
  });
}

/** 从 localStorage key 中扫描所有有缓存数据的用户 ID */
export function getAllUserIds(): string[] {
  const keys = Object.keys(localStorage);
  const userIdSet = new Set<string>();

  keys.forEach((key) => {
    // 匹配 msg_{userId}_ 前缀
    const msgMatch = key.match(/^msg_([^_]+(?:_[^_]+)*?)_/);
    if (msgMatch) {
      userIdSet.add(msgMatch[1]!);
    }
    // 匹配 msg_key_{userId}
    const keyMatch = key.match(/^msg_key_(.+)$/);
    if (keyMatch) {
      userIdSet.add(keyMatch[1]!);
    }
    // 匹配 conv_state_{userId}_ 前缀
    const convMatch = key.match(/^conv_state_([^_]+(?:_[^_]+)*?)_/);
    if (convMatch) {
      userIdSet.add(convMatch[1]!);
    }
    // 匹配 ai_cfg_{userId}, ai_conv_{userId}, ai_act_cfg_{userId}, ai_act_conv_{userId}
    const aiMatch = key.match(/^ai_(?:cfg|conv|act_cfg|act_conv)_(.+)$/);
    if (aiMatch) {
      userIdSet.add(aiMatch[1]!);
    }
  });

  return Array.from(userIdSet);
}

/** 兼容迁移：清除旧格式（无用户前缀）的缓存数据 */
export function migrateLegacyData() {
  const keys = Object.keys(localStorage);
  keys.forEach((key) => {
    // 旧的消息缓存格式
    if (key.startsWith('message_cache_')) {
      localStorage.removeItem(key);
    }
    // 旧的加密密钥
    if (key === 'message_encryption_key') {
      localStorage.removeItem(key);
    }
    // 旧的会话状态格式
    if (key.startsWith('conversation_state_')) {
      localStorage.removeItem(key);
    }
    // 旧的 AI 存储
    if (
      key === 'purr-chat-ai-configs' ||
      key === 'purr-chat-ai-conversations' ||
      key === 'purr-chat-ai-active-config' ||
      key === 'purr-chat-ai-active-conversation'
    ) {
      localStorage.removeItem(key);
    }
  });
}
