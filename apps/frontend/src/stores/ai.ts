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
  // 正在流式生成的会话 ID 集合（支持多会话并发）
  const streamingConversationIds = ref<Set<string>>(new Set());

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

  const setConversationStreaming = (conversationId: string, streaming: boolean) => {
    const newSet = new Set(streamingConversationIds.value);
    if (streaming) {
      newSet.add(conversationId);
    } else {
      newSet.delete(conversationId);
    }
    streamingConversationIds.value = newSet;
  };

  const setActiveConversation = (id: string | null) => {
    activeConversationId.value = id;
    const { activeConversation: key } = getStorageKeys();
    localStorage.setItem(key, id || '');
  };

  // 移除会话中最后一条助手消息（用于重新生成）
  const removeLastAssistantMessage = (conversationId: string): string | null => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return null;
    for (let i = conv.messages.length - 1; i >= 0; i--) {
      if (conv.messages[i]!.role === 'assistant') {
        const removed = conv.messages.splice(i, 1)[0]!;
        conv.updatedAt = new Date().toISOString();
        saveConversations();
        return removed.id;
      }
    }
    return null;
  };

  // 删除指定消息
  const deleteMessage = (conversationId: string, messageId: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return;
    conv.messages = conv.messages.filter((m) => m.id !== messageId);
    conv.updatedAt = new Date().toISOString();
    saveConversations();
    streamingVersion.value++;
  };

  // 删除指定消息及其之后的所有消息
  const removeMessagesFrom = (conversationId: string, messageId: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return;
    const idx = conv.messages.findIndex((m) => m.id === messageId);
    if (idx >= 0) {
      conv.messages.splice(idx);
      conv.updatedAt = new Date().toISOString();
      saveConversations();
      streamingVersion.value++;
    }
  };

  // 更新消息内容
  const updateMessageContent = (conversationId: string, messageId: string, content: string) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return;
    const msg = conv.messages.find((m) => m.id === messageId);
    if (msg) {
      msg.content = content;
      conv.updatedAt = new Date().toISOString();
      saveConversations();
      streamingVersion.value++;
    }
  };

  // 将当前 AI 消息内容保存为替代版本（用于分支重新生成）
  const addAlternative = (conversationId: string, messageId: string): boolean => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return false;
    const msg = conv.messages.find((m) => m.id === messageId);
    if (!msg || msg.role !== 'assistant' || !msg.content) return false;
    if (!msg.alternatives) msg.alternatives = [];
    msg.alternatives.push({
      id: crypto.randomUUID(),
      content: msg.content,
      thinking: msg.thinking,
      createdAt: msg.createdAt,
    });
    saveConversations();
    streamingVersion.value++;
    return true;
  };

  // 切换到指定替代版本（当前版本推入 alternatives，目标版本弹出设为当前）
  const switchAlternative = (conversationId: string, messageId: string, altIndex: number) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return;
    const msg = conv.messages.find((m) => m.id === messageId);
    if (!msg || !msg.alternatives || altIndex < 0 || altIndex >= msg.alternatives.length) return;
    const target = msg.alternatives.splice(altIndex, 1)[0]!;
    msg.alternatives.push({
      id: crypto.randomUUID(),
      content: msg.content,
      thinking: msg.thinking,
      createdAt: msg.createdAt,
    });
    msg.content = target.content;
    msg.thinking = target.thinking;
    saveConversations();
    streamingVersion.value++;
  };

  // 创建会话级分支：保存 fromMessageId 及之后的消息，然后删除
  const createBranch = (conversationId: string, fromMessageId: string): string | null => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return null;
    const idx = conv.messages.findIndex((m) => m.id === fromMessageId);
    if (idx < 0) return null;
    const branchMessages = conv.messages.splice(idx);
    if (!conv.branches) conv.branches = [];
    const branchId = crypto.randomUUID();
    conv.branches.push({
      id: branchId,
      fromMessageId,
      messages: branchMessages,
      createdAt: new Date().toISOString(),
    });
    conv.updatedAt = new Date().toISOString();
    saveConversations();
    streamingVersion.value++;
    return branchId;
  };

  // 恢复会话级分支：当前尾部保存为新分支，然后恢复目标分支
  const restoreBranch = (conversationId: string, branchId: string): boolean => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv || !conv.branches) return false;
    const branchIdx = conv.branches.findIndex((b) => b.id === branchId);
    if (branchIdx < 0) return false;
    // 保存当前尾部为新分支
    if (conv.messages.length > 0) {
      const lastMsg = conv.messages[conv.messages.length - 1]!;
      const newBranchId = crypto.randomUUID();
      conv.branches.push({
        id: newBranchId,
        fromMessageId: lastMsg.id,
        messages: [...conv.messages],
        createdAt: new Date().toISOString(),
      });
    }
    // 恢复目标分支
    const branch = conv.branches.splice(branchIdx, 1)[0]!;
    conv.messages = branch.messages;
    conv.updatedAt = new Date().toISOString();
    saveConversations();
    streamingVersion.value++;
    return true;
  };

  // 设置推理参数
  const setReasoningSettings = (
    conversationId: string,
    enabled: boolean,
    effort: string,
  ) => {
    const conv = conversations.value.find((c) => c.id === conversationId);
    if (!conv) return;
    conv.reasoningEnabled = enabled;
    conv.reasoningEffort = effort;
    saveConversations();
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

    // 清理被页面刷新中断的流式状态（HTTP 连接已断开，但消息仍标记为 streaming/thinking）
    let needsSave = false;
    for (const conv of conversations.value) {
      for (const msg of conv.messages) {
        if (msg.isStreaming || msg.isThinking) {
          msg.isStreaming = false;
          msg.isThinking = false;
          msg.isInterrupted = true;
          needsSave = true;
        }
        // 检测旧版错误消息（[错误] 前缀），标记 isError
        if (msg.content.startsWith('[错误]') && !msg.isError) {
          msg.isError = true;
          needsSave = true;
        }
      }
    }
    if (needsSave) {
      saveConversations();
    }

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
    streamingConversationIds,
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
    setConversationStreaming,
    setActiveConversation,
    removeLastAssistantMessage,
    deleteMessage,
    removeMessagesFrom,
    updateMessageContent,
    addAlternative,
    switchAlternative,
    createBranch,
    restoreBranch,
    setReasoningSettings,
    deleteConversation,
    // 初始化
    initStore,
    saveConversations,
  };
});
