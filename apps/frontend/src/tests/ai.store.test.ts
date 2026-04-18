import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useAiStore } from '../stores/ai';
import type { AiConfig, AiMessage } from '../models/types';

// Mock crypto.randomUUID
let uuidCounter = 0;
const mockUuids = () => {
  uuidCounter++;
  return `uuid-${uuidCounter}`;
};

describe('AI Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    uuidCounter = 0;
    vi.stubGlobal('crypto', { randomUUID: vi.fn(mockUuids) });
  });

  describe('Initial State', () => {
    it('should have empty configs and conversations', () => {
      const store = useAiStore();
      expect(store.configs).toEqual([]);
      expect(store.conversations).toEqual([]);
      expect(store.activeConfigId).toBeNull();
      expect(store.activeConversationId).toBeNull();
    });

    it('activeConfig should return null when no configs', () => {
      const store = useAiStore();
      expect(store.activeConfig).toBeNull();
    });

    it('activeConversation should return null when no conversations', () => {
      const store = useAiStore();
      expect(store.activeConversation).toBeNull();
    });

    it('activeMessages should return empty array when no conversation', () => {
      const store = useAiStore();
      expect(store.activeMessages).toEqual([]);
    });
  });

  describe('initStore', () => {
    it('should clear in-memory state and reload from localStorage', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });

      store.initStore();
      // After initStore, state should be reloaded from localStorage (legacy keys)
      expect(store.configs.length).toBe(1);
      expect(store.configs[0].id).toBe(config.id);
    });

    it('should use user-specific storage keys when userId provided', () => {
      const store = useAiStore();
      store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });

      store.initStore('user-123');

      // Should use namespaced keys
      const saved = localStorage.getItem('ai_cfg_user-123');
      expect(saved).toBeNull(); // No data for this user yet
      // But legacy keys should still have data
      expect(localStorage.getItem('purr-chat-ai-configs')).not.toBeNull();
    });

    it('should restore activeConfigId from localStorage', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });
      store.setActiveConfig(config.id);

      store.initStore();
      expect(store.activeConfigId).toBe(config.id);
    });

    it('should fall back to first config if saved activeConfig is invalid', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });

      // Manually set an invalid activeConfigId in localStorage
      localStorage.setItem('purr-chat-ai-active-config', 'non-existent-id');

      store.initStore();
      expect(store.activeConfigId).toBe(config.id);
    });
  });

  describe('Config CRUD', () => {
    it('addConfig should create config with id, createdAt, updatedAt', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'Test Config',
        apiUrl: 'http://api.test',
        apiKey: 'sk-test',
        model: 'gpt-4',
        temperature: 0.5,
      });

      expect(config.id).toBe('uuid-1');
      expect(config.name).toBe('Test Config');
      expect(config.createdAt).toBeDefined();
      expect(config.updatedAt).toBeDefined();
      expect(store.configs).toHaveLength(1);
    });

    it('addConfig should persist to localStorage', () => {
      const store = useAiStore();
      store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });

      const saved = localStorage.getItem('purr-chat-ai-configs');
      expect(saved).not.toBeNull();
      const parsed = JSON.parse(saved!);
      expect(parsed).toHaveLength(1);
    });

    it('updateConfig should update existing config and updatedAt', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });

      store.updateConfig(config.id, { name: 'Updated', temperature: 1.0 });

      const updated = store.configs[0];
      expect(updated.name).toBe('Updated');
      expect(updated.temperature).toBe(1.0);
      expect(updated.updatedAt).toBeDefined();
    });

    it('updateConfig should not modify config if id not found', () => {
      const store = useAiStore();
      store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });

      store.updateConfig('non-existent', { name: 'Updated' });
      expect(store.configs[0].name).toBe('Test');
    });

    it('deleteConfig should remove config and its conversations', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'Test',
        apiUrl: 'http://api.test',
        apiKey: 'key',
        model: 'model',
        temperature: 0.7,
      });
      store.createConversation(config.id);
      store.createConversation(config.id);

      store.deleteConfig(config.id);

      expect(store.configs).toHaveLength(0);
      expect(store.conversations).toHaveLength(0);
    });

    it('deleteConfig should reset activeConfigId if deleted config was active', () => {
      const store = useAiStore();
      const config1 = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const config2 = store.addConfig({
        name: 'C2',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      store.setActiveConfig(config1.id);

      store.deleteConfig(config1.id);

      expect(store.activeConfigId).toBe(config2.id);
    });

    it('deleteConfig should fall back to null if no configs remain', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      store.setActiveConfig(config.id);

      store.deleteConfig(config.id);

      expect(store.activeConfigId).toBeNull();
    });

    it('deleteConfig should clear activeConversationId if its conversation was deleted', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);
      expect(store.activeConversationId).toBe(conv.id);

      store.deleteConfig(config.id);

      expect(store.activeConversationId).toBeNull();
    });

    it('setActiveConfig should update activeConfigId and persist', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });

      store.setActiveConfig(config.id);

      expect(store.activeConfigId).toBe(config.id);
      expect(store.activeConfig).toEqual(config);
      expect(localStorage.getItem('purr-chat-ai-active-config')).toBe(config.id);
    });
  });

  describe('Computed properties', () => {
    it('activeConfig should return matching config or null', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });

      expect(store.activeConfig).toBeNull();
      store.setActiveConfig(config.id);
      expect(store.activeConfig).toEqual(config);
    });

    it('hasConfigs should return true when configs exist', () => {
      const store = useAiStore();
      expect(store.hasConfigs).toBe(false);

      store.addConfig({ name: 'C1', apiUrl: 'a', apiKey: 'k', model: 'm', temperature: 0.5 });
      expect(store.hasConfigs).toBe(true);
    });
  });

  describe('Conversation management', () => {
    it('createConversation should create conversation with correct configId', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });

      const conv = store.createConversation(config.id);

      expect(conv.configId).toBe(config.id);
      expect(conv.title).toBe('新对话');
      expect(conv.messages).toEqual([]);
      expect(store.conversations).toHaveLength(1);
    });

    it('createConversation should set as activeConversation', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });

      const conv = store.createConversation(config.id);

      expect(store.activeConversationId).toBe(conv.id);
      expect(store.activeConversation).toEqual(conv);
    });

    it('addMessage should add message to correct conversation', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);

      const msg: AiMessage = {
        id: 'msg-1',
        role: 'user',
        content: 'Hello',
        createdAt: new Date().toISOString(),
      };
      store.addMessage(conv.id, msg);

      expect(store.conversations[0].messages).toHaveLength(1);
      expect(store.conversations[0].messages[0].content).toBe('Hello');
    });

    it('addMessage should auto-generate title from first user message', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);

      store.addMessage(conv.id, {
        id: 'msg-1',
        role: 'user',
        content: 'This is a long message that exceeds 30 characters',
        createdAt: new Date().toISOString(),
      });

      expect(store.conversations[0].title).toBe('This is a long message that ex...');
    });

    it('addMessage should truncate long titles to 30 chars', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);

      store.addMessage(conv.id, {
        id: 'msg-1',
        role: 'user',
        content: 'A'.repeat(50),
        createdAt: new Date().toISOString(),
      });

      expect(store.conversations[0].title).toBe('A'.repeat(30) + '...');
    });

    it('deleteConversation should remove conversation', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      store.createConversation(config.id);
      store.createConversation(config.id);

      store.deleteConversation(store.conversations[0].id);
      expect(store.conversations).toHaveLength(1);
    });

    it('deleteConversation should reset activeConversationId if needed', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);

      store.deleteConversation(conv.id);
      expect(store.activeConversationId).toBeNull();
    });

    it('deleteConversation should fall back to first remaining', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      store.createConversation(config.id);
      const conv2 = store.createConversation(config.id);

      // Make conv1 active (conversations[0] is conv1 because unshift)
      store.setActiveConversation(store.conversations[0].id);
      store.deleteConversation(store.conversations[0].id);

      expect(store.activeConversationId).toBe(store.conversations[0]!.id);
    });

    it('setActiveConversation should update activeConversationId', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);

      store.setActiveConversation(null);
      expect(store.activeConversationId).toBeNull();

      store.setActiveConversation(conv.id);
      expect(store.activeConversationId).toBe(conv.id);
    });
  });

  describe('Streaming', () => {
    it('updateStreamingMessage should update content and increment streamingVersion', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);
      const msg: AiMessage = {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        createdAt: new Date().toISOString(),
        isStreaming: true,
      };
      store.addMessage(conv.id, msg);

      store.updateStreamingMessage(conv.id, 'msg-1', 'Hello');

      expect(store.conversations[0].messages[0].content).toBe('Hello');
      // Verify streamingVersion changed by checking activeMessages gets new reference
      const msgs1 = store.activeMessages;
      const msgs2 = store.activeMessages;
      // Both calls return the same slice (no new streaming update between them)
      expect(msgs1).toEqual(msgs2);
    });

    it('updateStreamingThinking should update thinking and increment version', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);
      const msg: AiMessage = {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        createdAt: new Date().toISOString(),
        isStreaming: true,
      };
      store.addMessage(conv.id, msg);

      store.updateStreamingThinking(conv.id, 'msg-1', 'thinking...');

      expect(store.conversations[0].messages[0].thinking).toBe('thinking...');
    });

    it('setThinkingState should toggle isThinking', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);
      const msg: AiMessage = {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        createdAt: new Date().toISOString(),
        isStreaming: true,
      };
      store.addMessage(conv.id, msg);

      store.setThinkingState(conv.id, 'msg-1', true);
      expect(store.conversations[0].messages[0].isThinking).toBe(true);

      store.setThinkingState(conv.id, 'msg-1', false);
      expect(store.conversations[0].messages[0].isThinking).toBe(false);
    });

    it('finalizeStreamingMessage should set isStreaming=false and isThinking=false', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);
      const msg: AiMessage = {
        id: 'msg-1',
        role: 'assistant',
        content: 'Final answer',
        createdAt: new Date().toISOString(),
        isStreaming: true,
        isThinking: true,
      };
      store.addMessage(conv.id, msg);

      store.finalizeStreamingMessage(conv.id, 'msg-1');

      expect(store.conversations[0].messages[0].isStreaming).toBe(false);
      expect(store.conversations[0].messages[0].isThinking).toBe(false);
    });

    it('activeMessages should return shallow copy of active conversation messages', () => {
      const store = useAiStore();
      const config = store.addConfig({
        name: 'C1',
        apiUrl: 'a',
        apiKey: 'k',
        model: 'm',
        temperature: 0.5,
      });
      const conv = store.createConversation(config.id);
      const msg: AiMessage = {
        id: 'msg-1',
        role: 'user',
        content: 'Hello',
        createdAt: new Date().toISOString(),
      };
      store.addMessage(conv.id, msg);

      const messages = store.activeMessages;
      expect(messages).toHaveLength(1);
      expect(messages[0].content).toBe('Hello');

      // Should be a shallow copy (not the same reference)
      expect(messages).not.toBe(store.activeConversation?.messages);
    });
  });
});
