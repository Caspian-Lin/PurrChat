import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { AiConfig, AiConversation, AiMessage } from '../models/types';
import {
  aiConfigsKey,
  aiConversationsKey,
  aiActiveConfigKey,
  aiActiveConversationKey,
} from '../utils/storageNamespace';

export const useAiStore = defineStore('ai', () => {
  // 当前用户 ID，用于生成隔离的 storage key
  const currentUserId = ref<string | null>(null);

  // 动态生成 storage key
  const getStorageKeys = () => {
    const uid = currentUserId.value;
    if (!uid) {
      // 未初始化时回退到旧 key（避免破坏性变更）
      return {
        configs: 'purr-chat-ai-configs',
        conversations: 'purr-chat-ai-conversations',
        activeConfig: 'purr-chat-ai-active-config',
        activeConversation: 'purr-chat-ai-active-conversation',
      };
    }
    return {
      configs: aiConfigsKey(uid),
      conversations: aiConversationsKey(uid),
      activeConfig: aiActiveConfigKey(uid),
      activeConversation: aiActiveConversationKey(uid),
    };
  };

  // ===== 配置管理 =====
  const configs = ref<AiConfig[]>([]);
  const activeConfigId = ref<string | null>(null);

  const activeConfig = computed(() => {
    return configs.value.find((c) => c.id === activeConfigId.value) || null;
  });

  const hasConfigs = computed(() => configs.value.length > 0);

  const loadConfigs = () => {
    try {
      const { configs: key } = getStorageKeys();
      const saved = localStorage.getItem(key);
      if (saved) {
        configs.value = JSON.parse(saved) as AiConfig[];
      }
    } catch (error) {
      console.error('[AiStore] Failed to load configs:', error);
    }
  };

  const saveConfigs = () => {
    try {
      const { configs: key } = getStorageKeys();
      localStorage.setItem(key, JSON.stringify(configs.value));
    } catch (error) {
      console.error('[AiStore] Failed to save configs:', error);
    }
  };

  const addConfig = (config: Omit<AiConfig, 'id' | 'createdAt' | 'updatedAt'>): AiConfig => {
    const newConfig: AiConfig = {
      ...config,
      id: crypto.randomUUID(),
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    configs.value.push(newConfig);
    saveConfigs();
    return newConfig;
  };

  const updateConfig = (id: string, updates: Partial<AiConfig>) => {
    const config = configs.value.find((c) => c.id === id);
    if (config) {
      Object.assign(config, updates, { updatedAt: new Date().toISOString() });
      saveConfigs();
    }
  };

  const deleteConfig = (id: string) => {
    configs.value = configs.value.filter((c) => c.id !== id);
    // 删除该配置下的所有会话
    conversations.value = conversations.value.filter((c) => c.configId !== id);
    saveConfigs();
    saveConversations();
    const { activeConfig: actCfgKey, activeConversation: actConvKey } = getStorageKeys();
    // 如果删除的是当前激活的配置
    if (activeConfigId.value === id) {
      activeConfigId.value = configs.value[0]?.id ?? null;
      localStorage.setItem(actCfgKey, activeConfigId.value || '');
    }
    // 如果当前会话属于被删除的配置
    if (
      activeConversationId.value &&
      !conversations.value.find((c) => c.id === activeConversationId.value)
    ) {
      activeConversationId.value = null;
      localStorage.setItem(actConvKey, '');
    }
  };

  const setActiveConfig = (id: string) => {
    activeConfigId.value = id;
    const { activeConfig: key } = getStorageKeys();
    localStorage.setItem(key, id);
  };

  // ===== 会话管理 =====
  const conversations = ref<AiConversation[]>([]);
  const activeConversationId = ref<string | null>(null);
  // 流式更新版本号，每次流式更新时递增以触发 computed 重新计算
  const streamingVersion = ref(0);

  const activeConversation = computed(() => {
    return conversations.value.find((c) => c.id === activeConversationId.value) || null;
  });

  const activeMessages = computed(() => {
    streamingVersion.value;
    const conv = activeConversation.value;
    // 返回数组浅拷贝，确保每次 streamingVersion 变化时子组件收到新的引用
    return conv ? conv.messages.slice() : [];
  });

  const loadConversations = () => {
    try {
      const { conversations: key } = getStorageKeys();
      const saved = localStorage.getItem(key);
      if (saved) {
        conversations.value = JSON.parse(saved) as AiConversation[];
      }
    } catch (error) {
      console.error('[AiStore] Failed to load conversations:', error);
    }
  };

  const saveConversations = () => {
    try {
      const { conversations: key } = getStorageKeys();
      localStorage.setItem(key, JSON.stringify(conversations.value));
    } catch (error) {
      console.error('[AiStore] Failed to save conversations:', error);
    }
  };

  const createConversation = (configId: string): AiConversation => {
    const conv: AiConversation = {
      id: crypto.randomUUID(),
      configId,
      title: '新对话',
      messages: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    conversations.value.unshift(conv);
    activeConversationId.value = conv.id;
    const { activeConversation: key } = getStorageKeys();
    localStorage.setItem(key, conv.id);
    saveConversations();
    return conv;
  };

  const addMessage = (conversationId: string, message: AiMessage) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (conv) {
      conv.messages.push(message);
      conv.updatedAt = new Date().toISOString();
      // 自动生成标题（使用第一条用户消息）
      if (message.role === 'user' && conv.title === '新对话') {
        conv.title = message.content.slice(0, 30) + (message.content.length > 30 ? '...' : '');
      }
      saveConversations();
    }
  };

  const updateStreamingMessage = (conversationId: string, messageId: string, content: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (conv) {
      const msg = conv.messages.find((m) => m.id === messageId);
      if (msg) {
        msg.content = content;
        streamingVersion.value++;
      }
    }
  };

  const updateStreamingThinking = (conversationId: string, messageId: string, thinking: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (conv) {
      const msg = conv.messages.find((m) => m.id === messageId);
      if (msg) {
        msg.thinking = thinking;
        streamingVersion.value++;
      }
    }
  };

  const setThinkingState = (conversationId: string, messageId: string, isThinking: boolean) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (conv) {
      const msg = conv.messages.find((m) => m.id === messageId);
      if (msg) {
        msg.isThinking = isThinking;
        streamingVersion.value++;
      }
    }
  };

  const finalizeStreamingMessage = (conversationId: string, messageId: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (conv) {
      const msg = conv.messages.find((m) => m.id === messageId);
      if (msg) {
        msg.isStreaming = false;
        msg.isThinking = false;
        conv.updatedAt = new Date().toISOString();
        saveConversations();
      }
    }
  };

  const setActiveConversation = (id: string | null) => {
    activeConversationId.value = id;
    const { activeConversation: key } = getStorageKeys();
    localStorage.setItem(key, id || '');
  };

  const deleteConversation = (id: string) => {
    conversations.value = conversations.value.filter((c) => c.id !== id);
    saveConversations();
    const { activeConversation: key } = getStorageKeys();
    if (activeConversationId.value === id) {
      activeConversationId.value = conversations.value[0]?.id ?? null;
      localStorage.setItem(key, activeConversationId.value || '');
    }
  };

  // ===== 初始化 =====
  /**
   * 初始化 store，加载指定用户的数据
   * 不会删除其他用户的数据
   */
  const initStore = (userId?: string) => {
    if (userId) {
      currentUserId.value = userId;
    } else {
      currentUserId.value = null;
    }

    // 清空内存中的状态
    configs.value = [];
    conversations.value = [];
    activeConfigId.value = null;
    activeConversationId.value = null;

    // 从 localStorage 加载当前用户的数据
    loadConfigs();
    loadConversations();
    const { activeConfig: actCfgKey, activeConversation: actConvKey } = getStorageKeys();
    // 恢复上次激活的配置
    const savedActiveConfig = localStorage.getItem(actCfgKey);
    if (savedActiveConfig && configs.value.some((c) => c.id === savedActiveConfig)) {
      activeConfigId.value = savedActiveConfig;
    } else if (configs.value.length > 0) {
      activeConfigId.value = configs.value[0]!.id;
    }
    // 恢复上次激活的会话
    const savedActiveConv = localStorage.getItem(actConvKey);
    if (savedActiveConv && conversations.value.some((c) => c.id === savedActiveConv)) {
      activeConversationId.value = savedActiveConv;
    }
  };

  return {
    // 配置状态
    configs,
    activeConfigId,
    activeConfig,
    hasConfigs,
    // 会话状态
    conversations,
    activeConversationId,
    activeConversation,
    activeMessages,
    // 配置方法
    addConfig,
    updateConfig,
    deleteConfig,
    setActiveConfig,
    // 会话方法
    createConversation,
    addMessage,
    updateStreamingMessage,
    updateStreamingThinking,
    setThinkingState,
    finalizeStreamingMessage,
    setActiveConversation,
    deleteConversation,
    // 初始化
    initStore,
    saveConversations,
  };
});
