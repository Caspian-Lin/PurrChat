import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Message } from '../models/types';
import { useMessageCache } from '../services/messageCache';

export const useMessageStore = defineStore('message', () => {
  // 消息状态
  const messages = ref<Map<string, Message[]>>(new Map()); // conversationId -> messages
  const loading = ref<Set<string>>(new Set()); // conversationId -> loading state
  const error = ref<Record<string, string>>({}); // conversationId -> error message

  // 获取消息缓存服务
  const messageCache = useMessageCache();

  // 计算属性
  const getMessages = computed(() => (conversationId: string) => {
    return messages.value.get(conversationId) || [];
  });

  const isLoading = computed(() => (conversationId: string) => {
    return loading.value.has(conversationId);
  });

  const getError = computed(() => (conversationId: string) => {
    return error.value[conversationId] || null;
  });

  const totalMessageCount = computed(() => {
    let total = 0;
    messages.value.forEach((msgList) => {
      total += msgList.length;
    });
    return total;
  });

  // 设置消息
  function setMessages(conversationId: string, newMessages: Message[]) {
    console.log(
      `[MessageStore] Setting ${newMessages.length} messages for conversation ${conversationId}`
    );
    messages.value.set(conversationId, newMessages);
  }

  // 添加消息
  function addMessage(conversationId: string, message: Message) {
    console.log(`[MessageStore] ===== 添加消息开始 =====`);
    console.log(`[MessageStore] 会话ID: ${conversationId}`);
    console.log(`[MessageStore] 消息ID: ${message.id}`);
    console.log(`[MessageStore] 消息内容: ${message.content}`);
    console.log(`[MessageStore] 发送者ID: ${message.sender_id}`);
    console.log(`[MessageStore] 创建时间: ${message.created_at}`);

    const currentMessages = messages.value.get(conversationId) || [];
    console.log(`[MessageStore] 当前消息数量: ${currentMessages.length}`);

    // 检查消息是否已存在
    const exists = currentMessages.some((m) => m.id === message.id);
    console.log(`[MessageStore] 消息是否已存在: ${exists}`);

    if (!exists) {
      const newMessages = [...currentMessages, message];
      messages.value.set(conversationId, newMessages);
      console.log(`[MessageStore] 消息已添加，新消息数量: ${newMessages.length}`);
      console.log(
        `[MessageStore] 所有消息ID:`,
        newMessages.map((m) => m.id)
      );

      // 缓存消息
      messageCache.addMessage(conversationId, message);
      console.log(`[MessageStore] 消息已缓存`);
    } else {
      console.log(`[MessageStore] 消息已存在，跳过添加`);
    }
    console.log(`[MessageStore] ===== 添加消息结束 =====`);
  }

  // 批量添加消息
  function addMessages(conversationId: string, newMessages: Message[]) {
    console.log(
      `[MessageStore] Adding ${newMessages.length} messages to conversation ${conversationId}`
    );
    const currentMessages = messages.value.get(conversationId) || [];
    // 只添加不存在的消息
    const messagesToAdd = newMessages.filter(
      (msg) => !currentMessages.some((m) => m.id === msg.id)
    );

    if (messagesToAdd.length > 0) {
      messages.value.set(conversationId, [...currentMessages, ...messagesToAdd]);
      // 缓存新消息
      messageCache.addMessages(conversationId, messagesToAdd);
    }
  }

  // 清除会话的消息
  function clearMessages(conversationId: string) {
    console.log(`[MessageStore] Clearing messages for conversation ${conversationId}`);
    messages.value.delete(conversationId);
  }

  // 清除所有消息
  function clearAllMessages() {
    console.log('[MessageStore] Clearing all messages');
    messages.value.clear();
  }

  // 设置加载状态
  function setLoading(conversationId: string, isLoading: boolean) {
    if (isLoading) {
      loading.value.add(conversationId);
    } else {
      loading.value.delete(conversationId);
    }
  }

  // 设置错误
  function setError(conversationId: string, errorMessage: string | null) {
    if (errorMessage) {
      error.value[conversationId] = errorMessage;
    } else {
      delete error.value[conversationId];
    }
  }

  // 更新会话的最后一条消息
  function updateLastMessage(conversationId: string, message: Message) {
    const currentMessages = messages.value.get(conversationId) || [];
    if (currentMessages.length > 0) {
      const lastMessage = currentMessages[currentMessages.length - 1];
      if (lastMessage && lastMessage.id !== message.id) {
        addMessage(conversationId, message);
      }
    }
  }

  // 从缓存加载消息
  async function loadFromCache(conversationId: string): Promise<Message[]> {
    console.log(`[MessageStore] Loading messages from cache for conversation ${conversationId}`);
    const cachedMessages = messageCache.getMessages(conversationId);
    if (cachedMessages.length > 0) {
      // 类型转换：CachedMessage[] -> Message[]
      const messagesAsMessage: Message[] = cachedMessages.map((msg) => ({
        ...msg,
        msg_type: msg.msg_type as 'text' | 'image',
        sender: msg.sender
          ? {
              id: msg.sender.id,
              uid: 0, // 缓存中没有uid，使用默认值
              username: msg.sender.username,
              avatar_url: msg.sender.avatar_url || '',
              email_verified: false,
              phone_verified: false,
              created_at: msg.created_at || '', // 使用缓存中的created_at
            }
          : undefined,
      }));
      setMessages(conversationId, messagesAsMessage);
      console.log(`[MessageStore] Loaded ${cachedMessages.length} messages from cache`);
      return messagesAsMessage;
    }
    return [];
  }

  // 检查并加载增量消息
  async function checkAndLoadIncremental(
    conversationId: string,
    sinceTimestamp: number
  ): Promise<number> {
    console.log(
      `[MessageStore] Checking incremental messages for conversation ${conversationId} since ${sinceTimestamp}`
    );
    // 检查是否有缓存
    if (messageCache.hasCache(conversationId)) {
      const lastUpdated = messageCache.getLastUpdated(conversationId);
      console.log(`[MessageStore] Cache found, last updated: ${lastUpdated}`);
      // 这里可以调用API获取增量消息
      // 实际实现需要在composable中完成
      return 0;
    } else {
      console.log(`[MessageStore] No cache found for conversation ${conversationId}`);
      return 0;
    }
  }

  // 更新消息状态
  function updateMessageStatus(
    conversationId: string,
    messageId: string,
    status: 'sending' | 'sent' | 'failed'
  ) {
    console.log(
      `[MessageStore] Updating message status for conversation ${conversationId}, message ${messageId} to ${status}`
    );
    const currentMessages = messages.value.get(conversationId);
    if (currentMessages) {
      const messageIndex = currentMessages.findIndex((m) => m.id === messageId);
      if (messageIndex !== -1 && currentMessages[messageIndex]) {
        currentMessages[messageIndex].sendStatus = status;
        console.log(`[MessageStore] Message status updated to ${status}`);
      }
    }
  }

  return {
    // 状态
    messages,
    loading,
    error,
    // 计算属性
    getMessages,
    isLoading,
    getError,
    totalMessageCount,
    // 方法
    setMessages,
    addMessage,
    addMessages,
    clearMessages,
    clearAllMessages,
    setLoading,
    setError,
    updateLastMessage,
    loadFromCache,
    checkAndLoadIncremental,
    updateMessageStatus,
  };
});
