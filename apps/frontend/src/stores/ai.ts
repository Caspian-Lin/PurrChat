import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { AiConfig, AiConversation, AiMessage } from '../models/types';

const AI_CONFIGS_STORAGE_KEY = 'purr-chat-ai-configs';
const AI_CONVERSATIONS_STORAGE_KEY = 'purr-chat-ai-conversations';
const AI_ACTIVE_CONFIG_KEY = 'purr-chat-ai-active-config';
const AI_ACTIVE_CONVERSATION_KEY = 'purr-chat-ai-active-conversation';

export const useAiStore = defineStore('ai', () => {
  // ===== 配置管理 =====
  const configs = ref<AiConfig[]>([]);
  const activeConfigId = ref<string | null>(null);

  const activeConfig = computed(() => {
    return configs.value.find((c) => c.id === activeConfigId.value) || null;
  });

  const hasConfigs = computed(() => configs.value.length > 0);

  const loadConfigs = () => {
    try {
      const saved = localStorage.getItem(AI_CONFIGS_STORAGE_KEY);
      if (saved) {
        configs.value = JSON.parse(saved) as AiConfig[];
      }
    } catch (error) {
      console.error('[AiStore] Failed to load configs:', error);
    }
  };

  const saveConfigs = () => {
    try {
      localStorage.setItem(AI_CONFIGS_STORAGE_KEY, JSON.stringify(configs.value));
    } catch (error) {
      console.error('[AiStore] Failed to save configs:', error);
    }
  };

  const addConfig = (
    config: Omit<AiConfig, 'id' | 'createdAt' | 'updatedAt'>
  ): AiConfig => {
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
    // 如果删除的是当前激活的配置
    if (activeConfigId.value === id) {
      activeConfigId.value = configs.value[0]?.id ?? null;
      localStorage.setItem(AI_ACTIVE_CONFIG_KEY, activeConfigId.value || '');
    }
    // 如果当前会话属于被删除的配置
    if (
      activeConversationId.value &&
      !conversations.value.find((c) => c.id === activeConversationId.value)
    ) {
      activeConversationId.value = null;
      localStorage.setItem(AI_ACTIVE_CONVERSATION_KEY, '');
    }
  };

  const setActiveConfig = (id: string) => {
    activeConfigId.value = id;
    localStorage.setItem(AI_ACTIVE_CONFIG_KEY, id);
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
    // eslint-disable-next-line @typescript-eslint/no-unused-expressions
    streamingVersion.value;
    return activeConversation.value?.messages || [];
  });

  const loadConversations = () => {
    try {
      const saved = localStorage.getItem(AI_CONVERSATIONS_STORAGE_KEY);
      if (saved) {
        conversations.value = JSON.parse(saved) as AiConversation[];
      }
    } catch (error) {
      console.error('[AiStore] Failed to load conversations:', error);
    }
  };

  const saveConversations = () => {
    try {
      localStorage.setItem(AI_CONVERSATIONS_STORAGE_KEY, JSON.stringify(conversations.value));
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
    localStorage.setItem(AI_ACTIVE_CONVERSATION_KEY, conv.id);
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

  const finalizeStreamingMessage = (conversationId: string, messageId: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (conv) {
      const msg = conv.messages.find((m) => m.id === messageId);
      if (msg) {
        msg.isStreaming = false;
        conv.updatedAt = new Date().toISOString();
        saveConversations();
      }
    }
  };

  const setActiveConversation = (id: string | null) => {
    activeConversationId.value = id;
    localStorage.setItem(AI_ACTIVE_CONVERSATION_KEY, id || '');
  };

  const deleteConversation = (id: string) => {
    conversations.value = conversations.value.filter((c) => c.id !== id);
    saveConversations();
    if (activeConversationId.value === id) {
      activeConversationId.value = conversations.value[0]?.id ?? null;
      localStorage.setItem(AI_ACTIVE_CONVERSATION_KEY, activeConversationId.value || '');
    }
  };

  // ===== 初始化 =====
  const initStore = () => {
    loadConfigs();
    loadConversations();
    // 恢复上次激活的配置
    const savedActiveConfig = localStorage.getItem(AI_ACTIVE_CONFIG_KEY);
    if (savedActiveConfig && configs.value.some((c) => c.id === savedActiveConfig)) {
      activeConfigId.value = savedActiveConfig;
    } else if (configs.value.length > 0) {
      activeConfigId.value = configs.value[0]!.id;
    }
    // 恢复上次激活的会话
    const savedActiveConv = localStorage.getItem(AI_ACTIVE_CONVERSATION_KEY);
    if (
      savedActiveConv &&
      conversations.value.some((c) => c.id === savedActiveConv)
    ) {
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
    finalizeStreamingMessage,
    setActiveConversation,
    deleteConversation,
    // 初始化
    initStore,
    saveConversations,
  };
});
