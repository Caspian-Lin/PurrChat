import { ref, computed } from 'vue';
import { useAiStore } from '../stores/ai';
import type { AiMessage, AiMessageRole } from '../models/types';

/**
 * SSE 流式 delta 的结构
 * 兼容不同 API 提供商的思维链字段
 */
interface StreamDelta {
  role?: string;
  content?: string | null;
  reasoning_content?: string | null; // DeepSeek / QwQ / vLLM
  reasoning?: string | null; // OpenAI 旧版 o-series
}

/**
 * 从 delta 中统一获取思维链文本
 */
function getThinkingText(delta: StreamDelta): string | null {
  return delta.reasoning_content ?? delta.reasoning ?? null;
}

export const useAiChat = () => {
  const error = ref<string | null>(null);
  // 每个会话独立的 AbortController，支持并发流式请求
  const abortControllers = new Map<string, AbortController>();

  // 基于当前活跃会话计算 isStreaming（切换会话时自动更新）
  const isStreaming = computed(() => {
    const store = useAiStore();
    return (
      store.activeConversationId != null &&
      store.streamingConversationIds.has(store.activeConversationId)
    );
  });

  /**
   * 构建发送给 API 的消息列表（排除占位和错误消息）
   */
  const buildApiMessages = (
    messages: AiMessage[],
    additionalContent?: string
  ): Array<{ role: AiMessageRole; content: string }> => {
    const apiMessages: Array<{ role: AiMessageRole; content: string }> = [];
    for (const msg of messages) {
      if (!msg.isStreaming && !msg.isError) {
        apiMessages.push({ role: msg.role, content: msg.content });
      }
    }
    // 附加额外内容（用于重发/编辑场景，此时新内容尚未在 messages 中）
    if (additionalContent) {
      apiMessages.push({ role: 'user', content: additionalContent });
    }
    return apiMessages;
  };

  /**
   * 核心流式请求逻辑（复用于 sendMessage / retryMessage / regenerateResponse / editAndResend）
   */
  const streamRequest = async (options: {
    conversationId: string;
    messages: AiMessage[];
    extraContent?: string;
  }) => {
    const store = useAiStore();
    const config = store.activeConfig;
    const convId = options.conversationId;

    if (!config) {
      error.value = '请先选择 AI 配置';
      return;
    }

    const conversation = store.conversations.find((c) => c.id === convId);
    if (!conversation) return;

    // 创建 AI 回复占位消息
    const assistantId = crypto.randomUUID();
    const assistantMessage: AiMessage = {
      id: assistantId,
      role: 'assistant',
      content: '',
      thinking: '',
      createdAt: new Date().toISOString(),
      isStreaming: true,
      isThinking: true,
    };
    store.addMessage(convId, assistantMessage);

    // 构建请求消息列表
    const apiMessages = buildApiMessages(options.messages, options.extraContent);

    // 发起请求
    store.setConversationStreaming(convId, true);
    error.value = null;
    abortControllers.set(convId, new AbortController());

    let fullContent = '';
    let fullThinking = '';
    let hasSSEData = false;
    let inThinkingPhase = true;
    let hasContentDelta = false;

    try {
      const url = config.apiUrl.replace(/\/+$/, '') + '/chat/completions';
      const body: Record<string, unknown> = {
        model: config.model,
        messages: apiMessages,
        temperature: config.temperature,
        stream: true,
      };
      if (config.maxTokens) {
        body.max_tokens = config.maxTokens;
      }

      // 推理参数：仅当启用时发送
      if (conversation.reasoningEnabled !== false && conversation.reasoningEffort) {
        body.reasoning_effort = conversation.reasoningEffort;
      }

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${config.apiKey}`,
        },
        body: JSON.stringify(body),
        signal: abortControllers.get(convId)!.signal,
      });

      if (!response.ok) {
        let errorMsg = `API 请求失败: ${response.status}`;
        try {
          const errBody = await response.json();
          errorMsg = errBody.error?.message || errBody.message || errorMsg;
        } catch {
          // ignore
        }
        throw new Error(errorMsg);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error('无法读取响应流');

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed || trimmed === 'data: [DONE]') continue;

          if (trimmed.startsWith('data: ')) {
            hasSSEData = true;
            try {
              const json = JSON.parse(trimmed.slice(6));
              const delta: StreamDelta = json.choices?.[0]?.delta;
              if (!delta) continue;

              if ('content' in delta) {
                hasContentDelta = true;
              }

              // 提取思维链内容
              const thinkingDelta = getThinkingText(delta);
              if (thinkingDelta) {
                fullThinking += thinkingDelta;
                store.updateStreamingThinking(convId, assistantId, fullThinking);
              }

              // 提取正式回复内容
              if (delta.content) {
                if (inThinkingPhase) {
                  inThinkingPhase = false;
                  store.setThinkingState(convId, assistantId, false);
                }
                fullContent += delta.content;
                store.updateStreamingMessage(convId, assistantId, fullContent);
              }
            } catch {
              // 忽略 JSON 解析错误
            }
          }
        }
      }

      // 非流式 fallback：API 可能忽略 stream: true 返回普通 JSON
      if (!hasSSEData && !fullContent) {
        try {
          const json = JSON.parse(buffer.trim());
          const messageContent = json.choices?.[0]?.message?.content;
          if (messageContent) {
            fullContent = messageContent;
            store.updateStreamingMessage(convId, assistantId, fullContent);
          }
        } catch {
          // JSON 解析失败，保持空内容
        }
      }

      // 模型只有思维链无独立 content 阶段
      if (!fullContent && fullThinking && !hasContentDelta) {
        store.setThinkingState(convId, assistantId, false);
        fullContent = fullThinking;
        store.updateStreamingMessage(convId, assistantId, fullContent);
      }

      store.finalizeStreamingMessage(convId, assistantId);
      store.saveConversations();
    } catch (err: unknown) {
      const e = err as Error;
      if (e.name === 'AbortError') {
        store.finalizeStreamingMessage(convId, assistantId);
        store.saveConversations();
      } else {
        error.value = e.message || '发送消息失败';
        if (!fullContent && !fullThinking) {
          store.updateStreamingMessage(convId, assistantId, `[错误] ${e.message || '未知错误'}`);
        }
        // 标记消息为错误状态
        const conv = store.conversations.find((c) => c.id === convId);
        if (conv) {
          const msg = conv.messages.find((m) => m.id === assistantId);
          if (msg) msg.isError = true;
        }
        store.finalizeStreamingMessage(convId, assistantId);
        store.saveConversations();
      }
    } finally {
      store.setConversationStreaming(convId, false);
      abortControllers.delete(convId);
    }
  };

  const sendMessage = async (content: string) => {
    const store = useAiStore();
    const conversation = store.activeConversation;

    if (!conversation) {
      error.value = '请先创建对话';
      return;
    }

    // 添加用户消息
    const userMessage: AiMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      createdAt: new Date().toISOString(),
    };
    store.addMessage(conversation.id, userMessage);

    // 发起流式请求
    await streamRequest({
      conversationId: conversation.id,
      messages: conversation.messages,
    });
  };

  // 重试错误/中断消息：找到该消息前的用户 prompt，删除错误消息后重发
  const retryMessage = async (messageId: string) => {
    const store = useAiStore();
    const conversation = store.activeConversation;
    if (!conversation) return;

    // 找到目标消息和其前的用户消息
    const msgIdx = conversation.messages.findIndex((m) => m.id === messageId);
    if (msgIdx < 0) return;

    let userContent: string | null = null;
    for (let i = msgIdx - 1; i >= 0; i--) {
      if (conversation.messages[i]!.role === 'user') {
        userContent = conversation.messages[i]!.content;
        break;
      }
    }
    if (!userContent) return;

    // 删除错误消息
    store.deleteMessage(conversation.id, messageId);

    // 重发（不额外添加用户消息，extraContent 用于在构建 API messages 时附加）
    await streamRequest({
      conversationId: conversation.id,
      messages: conversation.messages,
      extraContent: userContent,
    });
  };

  // 重新生成 AI 回复（可选分支）
  const regenerateResponse = async (messageId: string, branch: boolean) => {
    const store = useAiStore();
    const conversation = store.activeConversation;
    if (!conversation) return;

    const msgIdx = conversation.messages.findIndex((m) => m.id === messageId);
    if (msgIdx < 0) return;

    // 找到该 AI 消息前的用户消息
    let userContent: string | null = null;
    for (let i = msgIdx - 1; i >= 0; i--) {
      if (conversation.messages[i]!.role === 'user') {
        userContent = conversation.messages[i]!.content;
        break;
      }
    }
    if (!userContent) return;

    // 分支：保存当前版本为替代
    if (branch) {
      store.addAlternative(conversation.id, messageId);
    }

    // 删除当前 AI 消息
    store.deleteMessage(conversation.id, messageId);

    // 重发
    await streamRequest({
      conversationId: conversation.id,
      messages: conversation.messages,
      extraContent: userContent,
    });
  };

  // 编辑用户 prompt 并重发（可选分支）
  const editAndResend = async (messageId: string, newContent: string, branch: boolean) => {
    const store = useAiStore();
    const conversation = store.activeConversation;
    if (!conversation) return;

    const msgIdx = conversation.messages.findIndex((m) => m.id === messageId);
    if (msgIdx < 0) return;

    if (branch) {
      // 分支模式：保存该消息及之后的消息为分支快照
      store.createBranch(conversation.id, messageId);
    } else {
      // 覆盖模式：直接删除该消息及之后的所有消息
      store.removeMessagesFrom(conversation.id, messageId);
    }

    // 发送新消息（通过 sendMessage 会自动添加用户消息 + AI 占位）
    await sendMessage(newContent);
  };

  const stopGeneration = () => {
    const store = useAiStore();
    if (store.activeConversationId) {
      abortControllers.get(store.activeConversationId)?.abort();
    }
  };

  const clearError = () => {
    error.value = null;
  };

  return {
    isStreaming,
    error,
    sendMessage,
    stopGeneration,
    clearError,
    retryMessage,
    regenerateResponse,
    editAndResend,
  };
};
