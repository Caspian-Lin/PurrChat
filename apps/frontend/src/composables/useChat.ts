import { ref, nextTick } from 'vue';
import { api } from '../models/api';
import { useMessage } from './useMessage';
import { useMessageCache } from '../services/messageCache';
import type { Message } from '../models/types';

export const useChat = () => {
  const messages = ref<Message[]>([]);
  const message = useMessage();
  const messageCache = useMessageCache();
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
        messages.value = [...response.data].reverse();
        scrollToBottom();
      }
    } catch (error) {
      console.error('Failed to load messages:', error);
    }
  };

  /**
   * 发送消息
   * @param conversationId - 会话ID
   * @param content - 消息内容
   * @returns 是否发送成功
   */
  const sendMessage = async (conversationId: string, content: string): Promise<boolean> => {
    if (!content.trim()) return false;

    try {
      const response = await api.sendMessage({
        conversation_id: conversationId,
        content,
        msg_type: 'text',
      });

      if (response.success && response.data) {
        messages.value.push(response.data);
        scrollToBottom();

        // 缓存发送的消息
        messageCache.addMessage(conversationId, response.data);
        return true;
      }
      return false;
    } catch (error) {
      console.error('Failed to send message:', error);
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
    sendMessage,
    exportMessages,
    addMessage,
    clearMessages,
  };
};
