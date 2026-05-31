import { ref, nextTick } from 'vue';
import { api } from '../models/api';
import { useNotification } from './useNotification';
import { useMessageCache } from '../services/messageCache';
import { useMessageStore } from '../stores/message';
import { useAuthStore } from '../stores/auth';
import type { Message, SendMessageRequest, FileMessageContent } from '../models/types';

export const useChat = () => {
  const notify = useNotification();
  const messageCache = useMessageCache();
  const messageStore = useMessageStore();
  const messagesContainer = ref<HTMLElement | null>(null);

  /**
   * 获取消息
   * @param conversationId - 会话ID
   */
  const loadMessages = async (conversationId: string) => {
    try {
      const response = await api.getMessages(conversationId);
      if (response.success && response.data) {
        // 后端返回的消息是按created_at DESC排序的（从新到旧）
        // 需要反转顺序，让最新的消息在最下面
        const reversedMessages = [...response.data].reverse();

        // 更新message store（唯一数据源）
        messageStore.setMessages(conversationId, reversedMessages);
        scrollToBottom();

        // 缓存消息
        await messageCache.addMessages(conversationId, response.data);
        console.log(
          `[useChat] Loaded and cached ${response.data.length} messages for conversation ${conversationId}`
        );
      }
    } catch (error) {
      console.error('[useChat] Failed to load messages:', error);
    }
  };

  /**
   * 增量获取消息（从指定时间之后）
   * @param conversationId - 会话ID
   * @param sinceTimestamp - 起始时间戳（毫秒）
   */
  const loadMessagesIncremental = async (conversationId: string, sinceTimestamp: number) => {
    try {
      console.log(
        `[useChat] Loading incremental messages for conversation ${conversationId} since ${sinceTimestamp}`
      );
      const response = await api.getMessagesIncremental(conversationId, sinceTimestamp);
      if (response.success && response.data && response.data.length > 0) {
        // 构建服务器消息 ID 集合，用于校准本地缓存
        const serverMessageIds = new Set(response.data.map((msg) => msg.id));

        // 校准本地缓存：移除服务器上不存在的消息（被撤回/删除的）
        const removedCount = await messageCache.reconcileWithServer(
          conversationId,
          serverMessageIds
        );
        if (removedCount > 0) {
          console.log(
            `[useChat] Reconciled ${removedCount} removed messages for conversation ${conversationId}`
          );
          // 从 store 中移除已删除的消息
          const currentStoreMessages = messageStore.getMessages(conversationId);
          messageStore.setMessages(
            conversationId,
            currentStoreMessages.filter((m) => serverMessageIds.has(m.id))
          );
        }

        // 增量消息是按created_at ASC排序的（从旧到新）
        const newMessages: Message[] = [];
        const currentStoreMessages = messageStore.getMessages(conversationId);
        response.data.forEach((msg) => {
          // 检查消息是否已存在
          const exists = currentStoreMessages.some((m) => m.id === msg.id);
          if (!exists) {
            newMessages.push(msg);
          }
        });

        if (newMessages.length > 0) {
          messageStore.addMessages(conversationId, newMessages);
        }
        scrollToBottom();

        // 缓存新消息
        await messageCache.addMessages(conversationId, response.data);
        console.log(
          `[useChat] Loaded and cached ${response.data.length} incremental messages for conversation ${conversationId}`
        );
      } else {
        console.log(
          `[useChat] No new messages for conversation ${conversationId} since ${sinceTimestamp}`
        );
      }
      return response.success && response.data ? response.data.length : 0;
    } catch (error) {
      console.error('[useChat] Failed to load incremental messages:', error);
      return 0;
    }
  };

  /**
   * 检查并加载会话的增量消息
   * @param conversationId - 会话ID
   */
  const checkAndLoadIncremental = async (conversationId: string) => {
    // 检查是否有缓存
    if (messageCache.hasCache(conversationId)) {
      const lastUpdated = messageCache.getLastUpdated(conversationId);
      console.log(
        `[useChat] Checking incremental messages for conversation ${conversationId}, last updated: ${lastUpdated}`
      );
      const newMessageCount = await loadMessagesIncremental(conversationId, lastUpdated);
      return newMessageCount;
    } else {
      console.log(
        `[useChat] No cache found for conversation ${conversationId}, loading all messages`
      );
      await loadMessages(conversationId);
      return 0;
    }
  };

  /**
   * 发送消息
   * @param conversationId - 会话ID
   * @param content - 消息内容
   * @returns 是否发送成功
   */
  const sendMessage = async (conversationId: string, content: string): Promise<boolean> => {
    console.log(
      '[useChat] sendMessage called with conversationId:',
      conversationId,
      'content:',
      content
    );
    if (!content.trim()) {
      console.log('[useChat] Content is empty, returning false');
      return false;
    }

    try {
      const authStore = useAuthStore();
      const currentUser = authStore.currentUser;

      // 创建临时消息ID（用于匹配WebSocket返回的消息）
      const tempId = `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

      // 先添加一条带"发送中"状态的消息到store，携带完整的发送者信息
      const tempMessage: Message = {
        id: tempId,
        conversation_id: conversationId,
        sender_id: currentUser?.id || '',
        content: content,
        msg_type: 'text',
        created_at: new Date().toISOString(),
        sendStatus: 'sending',
        sender: currentUser
          ? {
              id: currentUser.id,
              uid: currentUser.uid,
              username: currentUser.username,
              avatar_url: currentUser.avatar_url,
              email_verified: currentUser.email_verified,
              phone_verified: currentUser.phone_verified,
              created_at: currentUser.created_at,
            }
          : undefined,
      };

      console.log('[useChat] Adding temporary message with sending status:', tempMessage);
      messageStore.addMessage(conversationId, tempMessage);
      scrollToBottom();

      // 发送API请求
      const requestData: SendMessageRequest = {
        conversation_id: conversationId,
        content,
        msg_type: 'text',
      };
      console.log('[useChat] Sending message with data:', JSON.stringify(requestData, null, 2));
      const response = await api.sendMessage(requestData);

      console.log('[useChat] sendMessage response:', response);
      if (response.success && response.data) {
        // 不在此处标记为已发送，等待 WebSocket 事件携带完整消息替换临时消息
        // WebSocket 匹配逻辑依赖 sendStatus === 'sending' 来查找临时消息

        // 缓存发送的消息
        console.log('[useChat] Caching message');
        try {
          await messageCache.addMessage(conversationId, response.data);
          console.log(`[useChat] Message sent and cached for conversation ${conversationId}`);
        } catch (error) {
          console.error('[useChat] Error caching message:', error);
        }
        return true;
      }
      console.log('[useChat] sendMessage response not successful or no data');
      messageStore.updateMessageStatus(conversationId, tempId, 'failed');
      return false;
    } catch (error) {
      console.error('[useChat] Failed to send message:', error);
      notify.error('发送消息失败');
      // 更新临时消息的状态为"发送失败"
      const currentMessages = messageStore.getMessages(conversationId);
      const tempMessage = currentMessages.find((m) => m.id.startsWith('temp-'));
      if (tempMessage) {
        messageStore.updateMessageStatus(conversationId, tempMessage.id, 'failed');
      }
      return false;
    }
  };

  /**
   * 导出消息
   * @param conversationId - 会话ID
   */
  const exportMessages = async (conversationId: string) => {
    try {
      const response = await api.exportMessages(conversationId);
      if (response.success && response.data) {
        // 将消息数据转换为JSON字符串
        const jsonData = JSON.stringify(response.data, null, 2);

        // 创建Blob并下载
        const blob = new Blob([jsonData], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = `messages_${conversationId}_${new Date().toISOString().split('T')[0]}.json`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);

        notify.success(`成功导出 ${response.data.length} 条消息`);
      } else {
        notify.error('没有可导出的消息');
      }
    } catch (error) {
      console.error('Failed to export messages:', error);
      notify.error('导出消息失败');
    }
  };

  /**
   * 发送文件消息
   * @param conversationId - 会话ID
   * @param fileContent - 文件消息内容
   * @returns 是否发送成功
   */
  const sendFileMessage = async (
    conversationId: string,
    fileContent: FileMessageContent
  ): Promise<boolean> => {
    const contentJson = JSON.stringify(fileContent);

    try {
      const authStore = useAuthStore();
      const currentUser = authStore.currentUser;

      const tempId = `temp-file-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

      const tempMessage: Message = {
        id: tempId,
        conversation_id: conversationId,
        sender_id: currentUser?.id || '',
        content: contentJson,
        msg_type: 'file',
        created_at: new Date().toISOString(),
        sendStatus: 'sending',
        sender: currentUser
          ? {
              id: currentUser.id,
              uid: currentUser.uid,
              username: currentUser.username,
              avatar_url: currentUser.avatar_url,
              email_verified: currentUser.email_verified,
              phone_verified: currentUser.phone_verified,
              created_at: currentUser.created_at,
            }
          : undefined,
      };

      messageStore.addMessage(conversationId, tempMessage);
      scrollToBottom();

      const requestData: SendMessageRequest = {
        conversation_id: conversationId,
        content: contentJson,
        msg_type: 'file',
      };
      const response = await api.sendMessage(requestData);

      if (response.success && response.data) {
        // 不在此处标记为已发送，等待 WebSocket 事件替换临时消息
        try {
          await messageCache.addMessage(conversationId, response.data);
        } catch (error) {
          console.error('[useChat] Error caching file message:', error);
        }
        return true;
      }

      messageStore.updateMessageStatus(conversationId, tempId, 'failed');
      return false;
    } catch (error) {
      console.error('[useChat] Failed to send file message:', error);
      const currentMessages = messageStore.getMessages(conversationId);
      const tempMessage = currentMessages.find(
        (m) => m.id.startsWith('temp-file-') && m.sendStatus === 'sending'
      );
      if (tempMessage) {
        messageStore.updateMessageStatus(conversationId, tempMessage.id, 'failed');
      }
      return false;
    }
  };

  /**
   * 滚动到底部
   */
  const scrollToBottom = async () => {
    await nextTick();
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
    }
  };

  /**
   * 添加新消息到store（用于WebSocket实时接收）
   * @param newMessage - 新消息
   */
  const addMessage = (newMessage: Message) => {
    messageStore.addMessage(newMessage.conversation_id, newMessage);
    scrollToBottom();
  };

  /**
   * 清空指定会话的消息
   * @param conversationId - 会话ID
   */
  const clearMessages = (conversationId?: string) => {
    if (conversationId) {
      messageStore.clearMessages(conversationId);
    }
  };

  return {
    messagesContainer,
    loadMessages,
    loadMessagesIncremental,
    checkAndLoadIncremental,
    sendMessage,
    sendFileMessage,
    exportMessages,
    addMessage,
    clearMessages,
  };
};
