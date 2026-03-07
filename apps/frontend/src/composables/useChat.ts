import { ref, nextTick } from 'vue';
import { api } from '../models/api';
import { useMessage } from './useMessage';
import { useMessageCache } from '../services/messageCache';
import { useMessageStore } from '../stores/message';
import type { Message, SendMessageRequest } from '../models/types';

export const useChat = () => {
  const messages = ref<Message[]>([]);
  const message = useMessage();
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
        messages.value = reversedMessages;
        scrollToBottom();

        // 更新message store
        messageStore.setMessages(conversationId, reversedMessages);

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
        // 增量消息是按created_at ASC排序的（从旧到新）
        // 直接添加到消息列表
        const newMessages: Message[] = [];
        response.data.forEach((msg) => {
          // 检查消息是否已存在
          const exists = messages.value.some((m) => m.id === msg.id);
          if (!exists) {
            messages.value.push(msg);
            newMessages.push(msg);
          }
        });
        scrollToBottom();

        // 更新message store
        messageStore.addMessages(conversationId, newMessages);

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
      const requestData: SendMessageRequest = {
        conversation_id: conversationId,
        content,
        msg_type: 'text',
      };
      console.log('[useChat] Sending message with data:', JSON.stringify(requestData, null, 2));
      const response = await api.sendMessage(requestData);

      console.log('[useChat] sendMessage response:', response);
      if (response.success && response.data) {
        console.log('[useChat] Response successful, adding message to messages.value');
        messages.value.push(response.data);
        scrollToBottom();

        // 更新message store
        console.log('[useChat] Adding message to messageStore');
        messageStore.addMessage(conversationId, response.data);

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
      return false;
    } catch (error) {
      console.error('[useChat] Failed to send message:', error);
      message.error('发送消息失败');
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

        message.success(`成功导出 ${response.data.length} 条消息`);
      } else {
        message.error('没有可导出的消息');
      }
    } catch (error) {
      console.error('Failed to export messages:', error);
      message.error('导出消息失败');
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
   * 添加新消息到列表（用于WebSocket实时接收）
   * @param newMessage - 新消息
   */
  const addMessage = (newMessage: Message) => {
    messages.value.push(newMessage);
    scrollToBottom();

    // 更新message store
    messageStore.addMessage(newMessage.conversation_id, newMessage);
  };

  /**
   * 清空消息列表
   */
  const clearMessages = () => {
    messages.value = [];
  };

  return {
    messages,
    messagesContainer,
    loadMessages,
    loadMessagesIncremental,
    checkAndLoadIncremental,
    sendMessage,
    exportMessages,
    addMessage,
    clearMessages,
  };
};
