import { ref } from 'vue';
import { useAiStore } from '../stores/ai';
import type { AiMessage, AiMessageRole } from '../models/types';

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
    const assistantMessage: AiMessage = {
      id: crypto.randomUUID(),
      role: 'assistant',
      content: '',
      createdAt: new Date().toISOString(),
      isStreaming: true,
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

    // 4. 发起 SSE 流式请求
    isStreaming.value = true;
    error.value = null;
    abortController.value = new AbortController();

    let fullContent = '';

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
          if (!trimmed.startsWith('data: ')) continue;

          try {
            const json = JSON.parse(trimmed.slice(6));
            const delta = json.choices?.[0]?.delta?.content;
            if (delta) {
              fullContent += delta;
              store.updateStreamingMessage(conversation.id, assistantMessage.id, fullContent);
            }
          } catch {
            // 忽略解析错误（可能是不完整的 JSON）
          }
        }
      }

      store.finalizeStreamingMessage(conversation.id, assistantMessage.id);
      store.saveConversations();
    } catch (err: unknown) {
      const e = err as Error;
      if (e.name === 'AbortError') {
        store.finalizeStreamingMessage(conversation.id, assistantMessage.id);
        store.saveConversations();
      } else {
        error.value = e.message || '发送消息失败';
        if (!fullContent) {
          store.updateStreamingMessage(
            conversation.id,
            assistantMessage.id,
            `[错误] ${e.message || '未知错误'}`
          );
        }
        store.finalizeStreamingMessage(conversation.id, assistantMessage.id);
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
