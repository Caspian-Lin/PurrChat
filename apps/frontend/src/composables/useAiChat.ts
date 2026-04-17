import { ref } from 'vue';
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
  const isStreaming = ref(false);
  const error = ref<string | null>(null);
  const abortController = ref<AbortController | null>(null);

  const sendMessage = async (content: string) => {
    const store = useAiStore();
    const config = store.activeConfig;
    const conversation = store.activeConversation;

    if (!config) {
      error.value = '请先选择 AI 配置';
      return;
    }

    if (!conversation) {
      error.value = '请先创建对话';
      return;
    }

    // 1. 添加用户消息
    const userMessage: AiMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      createdAt: new Date().toISOString(),
    };
    store.addMessage(conversation.id, userMessage);

    // 2. 创建 AI 回复占位消息
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
    store.addMessage(conversation.id, assistantMessage);

    // 3. 构建请求消息列表
    const apiMessages: Array<{ role: AiMessageRole; content: string }> = [];
    for (const msg of conversation.messages) {
      if (!msg.isStreaming) {
        apiMessages.push({ role: msg.role, content: msg.content });
      }
    }
    apiMessages.push({ role: 'user', content });

    // 4. 发起请求
    isStreaming.value = true;
    error.value = null;
    abortController.value = new AbortController();

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

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${config.apiKey}`,
        },
        body: JSON.stringify(body),
        signal: abortController.value.signal,
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
                store.updateStreamingThinking(conversation.id, assistantId, fullThinking);
              }

              // 提取正式回复内容
              if (delta.content) {
                if (inThinkingPhase) {
                  inThinkingPhase = false;
                  store.setThinkingState(conversation.id, assistantId, false);
                }
                fullContent += delta.content;
                store.updateStreamingMessage(conversation.id, assistantId, fullContent);
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
            store.updateStreamingMessage(conversation.id, assistantId, fullContent);
          }
        } catch {
          // JSON 解析失败，保持空内容
        }
      }

      // 模型只有思维链无独立 content 阶段
      if (!fullContent && fullThinking && !hasContentDelta) {
        store.setThinkingState(conversation.id, assistantId, false);
        fullContent = fullThinking;
        store.updateStreamingMessage(conversation.id, assistantId, fullContent);
      }

      store.finalizeStreamingMessage(conversation.id, assistantId);
      store.saveConversations();
    } catch (err: unknown) {
      const e = err as Error;
      if (e.name === 'AbortError') {
        store.finalizeStreamingMessage(conversation.id, assistantId);
        store.saveConversations();
      } else {
        error.value = e.message || '发送消息失败';
        if (!fullContent && !fullThinking) {
          store.updateStreamingMessage(
            conversation.id,
            assistantId,
            `[错误] ${e.message || '未知错误'}`
          );
        }
        store.finalizeStreamingMessage(conversation.id, assistantId);
        store.saveConversations();
      }
    } finally {
      isStreaming.value = false;
      abortController.value = null;
    }
  };

  const stopGeneration = () => {
    abortController.value?.abort();
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
  };
};
