import { ref } from 'vue';
import type { Message } from '../models/types';

export interface CachedMessage {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  msg_type: string;
  created_at: string;
  sender?: {
    id: string;
    username: string;
    avatar_url?: string;
  };
}

export interface ConversationCache {
  conversationId: string;
  messages: CachedMessage[];
  lastUpdated: number;
}

class MessageCacheService {
  private cache = new Map<string, ConversationCache>();
  private storageKeyPrefix = 'message_cache_';
  private encryptionKey: string | null = null;

  constructor() {
    this.loadEncryptionKey();
    this.loadCacheFromStorage();
  }

  // 加载加密密钥
  private loadEncryptionKey() {
    // 从localStorage获取加密密钥
    this.encryptionKey = localStorage.getItem('message_encryption_key');
    if (!this.encryptionKey) {
      // 生成新的加密密钥
      this.encryptionKey = this.generateEncryptionKey();
      localStorage.setItem('message_encryption_key', this.encryptionKey);
    }
  }

  // 生成加密密钥
  private generateEncryptionKey(): string {
    const array = new Uint8Array(32);
    crypto.getRandomValues(array);
    return Array.from(array, (byte) => byte.toString(16).padStart(2, '0')).join('');
  }

  // 加密数据
  private encrypt(data: string): string {
    if (!this.encryptionKey) {
      return data;
    }

    try {
      // 简单的XOR加密（实际应用中应该使用更强大的加密算法）
      const key = this.encryptionKey;
      let encrypted = '';
      for (let i = 0; i < data.length; i++) {
        encrypted += String.fromCharCode(data.charCodeAt(i) ^ key.charCodeAt(i % key.length));
      }
      return btoa(encrypted); // Base64编码
    } catch (error) {
      console.error('Encryption failed:', error);
      return data;
    }
  }

  // 解密数据
  private decrypt(encryptedData: string): string {
    if (!this.encryptionKey) {
      return encryptedData;
    }

    try {
      const encrypted = atob(encryptedData); // Base64解码
      const key = this.encryptionKey;
      let decrypted = '';
      for (let i = 0; i < encrypted.length; i++) {
        decrypted += String.fromCharCode(encrypted.charCodeAt(i) ^ key.charCodeAt(i % key.length));
      }
      return decrypted;
    } catch (error) {
      console.error('Decryption failed:', error);
      return encryptedData;
    }
  }

  // 从localStorage加载缓存
  private loadCacheFromStorage() {
    try {
      const keys = Object.keys(localStorage);
      keys.forEach((key) => {
        if (key.startsWith(this.storageKeyPrefix)) {
          const conversationId = key.replace(this.storageKeyPrefix, '');
          const encryptedData = localStorage.getItem(key);
          if (encryptedData) {
            try {
              const decryptedData = this.decrypt(encryptedData);
              const cache: ConversationCache = JSON.parse(decryptedData);
              this.cache.set(conversationId, cache);
            } catch (error) {
              console.error('Failed to load cache for conversation:', conversationId, error);
            }
          }
        }
      });
      console.log('Loaded message cache for', this.cache.size, 'conversations');
    } catch (error) {
      console.error('Failed to load cache from storage:', error);
    }
  }

  // 保存缓存到localStorage
  private saveCacheToStorage(conversationId: string) {
    const cache = this.cache.get(conversationId);
    if (!cache) {
      return;
    }

    try {
      const data = JSON.stringify(cache);
      const encryptedData = this.encrypt(data);
      localStorage.setItem(`${this.storageKeyPrefix}${conversationId}`, encryptedData);
    } catch (error) {
      console.error('Failed to save cache for conversation:', conversationId, error);
    }
  }

  // 获取会话的消息
  getMessages(conversationId: string): CachedMessage[] {
    const cache = this.cache.get(conversationId);
    return cache ? cache.messages : [];
  }

  // 添加消息到缓存
  addMessage(conversationId: string, message: Message | CachedMessage) {
    let cache = this.cache.get(conversationId);
    if (!cache) {
      cache = {
        conversationId,
        messages: [],
        lastUpdated: Date.now(),
      };
      this.cache.set(conversationId, cache);
    }

    // 检查消息是否已存在
    const exists = cache.messages.some((m) => m.id === message.id);
    if (!exists) {
      cache.messages.push(message as CachedMessage);
      cache.lastUpdated = Date.now();
      this.saveCacheToStorage(conversationId);
    }
  }

  // 批量添加消息到缓存
  addMessages(conversationId: string, messages: (Message | CachedMessage)[]) {
    let cache = this.cache.get(conversationId);
    if (!cache) {
      cache = {
        conversationId,
        messages: [],
        lastUpdated: Date.now(),
      };
      this.cache.set(conversationId, cache);
    }

    // 只添加不存在的消息
    messages.forEach((message) => {
      const exists = cache!.messages.some((m) => m.id === message.id);
      if (!exists) {
        cache!.messages.push(message as CachedMessage);
      }
    });

    cache.lastUpdated = Date.now();
    this.saveCacheToStorage(conversationId);
  }

  // 清除会话的缓存
  clearConversation(conversationId: string) {
    this.cache.delete(conversationId);
    localStorage.removeItem(`${this.storageKeyPrefix}${conversationId}`);
  }

  // 清除所有缓存
  clearAll() {
    this.cache.clear();
    const keys = Object.keys(localStorage);
    keys.forEach((key) => {
      if (key.startsWith(this.storageKeyPrefix)) {
        localStorage.removeItem(key);
      }
    });
  }

  // 导出会话消息为JSON文件
  exportConversation(conversationId: string): string | null {
    const cache = this.cache.get(conversationId);
    if (!cache) {
      return null;
    }

    try {
      return JSON.stringify(cache.messages, null, 2);
    } catch (error) {
      console.error('Failed to export conversation:', conversationId, error);
      return null;
    }
  }

  // 导出所有消息为JSON文件
  exportAll(): string | null {
    const allMessages: Record<string, CachedMessage[]> = {};
    this.cache.forEach((cache, conversationId) => {
      allMessages[conversationId] = cache.messages;
    });

    try {
      return JSON.stringify(allMessages, null, 2);
    } catch (error) {
      console.error('Failed to export all messages:', error);
      return null;
    }
  }

  // 下载导出的JSON文件
  downloadExport(data: string, filename: string) {
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  // 获取缓存统计信息
  getStats() {
    let totalMessages = 0;
    const conversations: string[] = [];
    this.cache.forEach((cache, conversationId) => {
      totalMessages += cache.messages.length;
      conversations.push(conversationId);
    });

    return {
      totalConversations: conversations.length,
      totalMessages,
      conversations,
      totalSize: JSON.stringify(Array.from(this.cache.entries())).length,
    };
  }
}

// 创建全局消息缓存服务实例
export const messageCacheService = new MessageCacheService();

// Vue composable
export function useMessageCache() {
  const stats = ref(messageCacheService.getStats());

  const refreshStats = () => {
    stats.value = messageCacheService.getStats();
  };

  return {
    stats,
    refreshStats,
    getMessages: messageCacheService.getMessages.bind(messageCacheService),
    addMessage: messageCacheService.addMessage.bind(messageCacheService),
    addMessages: messageCacheService.addMessages.bind(messageCacheService),
    clearConversation: messageCacheService.clearConversation.bind(messageCacheService),
    clearAll: messageCacheService.clearAll.bind(messageCacheService),
    exportConversation: messageCacheService.exportConversation.bind(messageCacheService),
    exportAll: messageCacheService.exportAll.bind(messageCacheService),
    downloadExport: messageCacheService.downloadExport.bind(messageCacheService),
  };
}
