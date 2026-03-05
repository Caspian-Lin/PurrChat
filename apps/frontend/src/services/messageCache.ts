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
  private cryptoKey: CryptoKey | null = null;
  private keyInitialized = false;

  constructor() {
    this.initCryptoKey().then(() => {
      this.loadCacheFromStorage();
    });
  }

  // 初始化加密密钥
  private async initCryptoKey() {
    try {
      // 从localStorage获取加密密钥
      const keyData = localStorage.getItem('message_encryption_key');
      if (keyData) {
        // 导入现有密钥
        this.cryptoKey = await this.importKey(keyData);
        console.log('[MessageCache] Loaded existing encryption key');
      } else {
        // 生成新的加密密钥
        this.cryptoKey = await this.generateCryptoKey();
        const exportedKey = await this.exportKey(this.cryptoKey);
        localStorage.setItem('message_encryption_key', exportedKey);
        console.log('[MessageCache] Generated new encryption key');
      }
      this.keyInitialized = true;
    } catch (error) {
      console.error('[MessageCache] Failed to initialize crypto key:', error);
    }
  }

  // 等待密钥初始化完成
  private async waitForKeyInitialization() {
    while (!this.keyInitialized) {
      await new Promise(resolve => setTimeout(resolve, 10));
    }
  }

  // 生成加密密钥
  private async generateCryptoKey(): Promise<CryptoKey> {
    return await crypto.subtle.generateKey(
      {
        name: 'AES-GCM',
        length: 256,
      },
      true,
      ['encrypt', 'decrypt']
    );
  }

  // 导出密钥
  private async exportKey(key: CryptoKey): Promise<string> {
    const exported = await crypto.subtle.exportKey('jwk', key);
    return btoa(JSON.stringify(exported));
  }

  // 导入密钥
  private async importKey(keyData: string): Promise<CryptoKey> {
    const jwk = JSON.parse(atob(keyData));
    return await crypto.subtle.importKey(
      'jwk',
      jwk,
      {
        name: 'AES-GCM',
        length: 256,
      },
      true,
      ['encrypt', 'decrypt']
    );
  }

  // 加密数据
  private async encrypt(data: string): Promise<string> {
    await this.waitForKeyInitialization();
    if (!this.cryptoKey) {
      console.warn('[MessageCache] No encryption key available, storing data unencrypted');
      return data;
    }

    try {
      const encoder = new TextEncoder();
      const iv = crypto.getRandomValues(new Uint8Array(12));
      const encrypted = await crypto.subtle.encrypt(
        {
          name: 'AES-GCM',
          iv: iv,
        },
        this.cryptoKey,
        encoder.encode(data)
      );

      // 将IV和加密数据合并
      const combined = new Uint8Array(iv.length + encrypted.byteLength);
      combined.set(iv);
      combined.set(new Uint8Array(encrypted), iv.length);

      return btoa(String.fromCharCode(...combined));
    } catch (error) {
      console.error('[MessageCache] Encryption failed:', error);
      return data;
    }
  }

  // 解密数据
  private async decrypt(encryptedData: string): Promise<string> {
    await this.waitForKeyInitialization();
    if (!this.cryptoKey) {
      return encryptedData;
    }

    try {
      const combined = Uint8Array.from(atob(encryptedData), c => c.charCodeAt(0));
      const iv = combined.slice(0, 12);
      const encrypted = combined.slice(12);

      const decrypted = await crypto.subtle.decrypt(
        {
          name: 'AES-GCM',
          iv: iv,
        },
        this.cryptoKey,
        encrypted
      );

      const decoder = new TextDecoder();
      return decoder.decode(decrypted);
    } catch (error) {
      console.error('[MessageCache] Decryption failed:', error);
      return encryptedData;
    }
  }

  // 从localStorage加载缓存
  private async loadCacheFromStorage() {
    try {
      await this.waitForKeyInitialization();
      const keys = Object.keys(localStorage);
      for (const key of keys) {
        if (key.startsWith(this.storageKeyPrefix)) {
          const conversationId = key.replace(this.storageKeyPrefix, '');
          const encryptedData = localStorage.getItem(key);
          if (encryptedData) {
            try {
              const decryptedData = await this.decrypt(encryptedData);
              const cache: ConversationCache = JSON.parse(decryptedData);
              this.cache.set(conversationId, cache);
            } catch (error) {
              console.error('[MessageCache] Failed to load cache for conversation:', conversationId, error);
            }
          }
        }
      }
      console.log(`[MessageCache] Loaded message cache for ${this.cache.size} conversations`);
    } catch (error) {
      console.error('[MessageCache] Failed to load cache from storage:', error);
    }
  }

  // 保存缓存到localStorage
  private async saveCacheToStorage(conversationId: string) {
    const cache = this.cache.get(conversationId);
    if (!cache) {
      return;
    }

    try {
      const data = JSON.stringify(cache);
      const encryptedData = await this.encrypt(data);
      localStorage.setItem(`${this.storageKeyPrefix}${conversationId}`, encryptedData);
      console.log(`[MessageCache] Saved cache for conversation ${conversationId}`);
    } catch (error) {
      console.error('[MessageCache] Failed to save cache for conversation:', conversationId, error);
    }
  }

  // 获取会话的消息
  getMessages(conversationId: string): CachedMessage[] {
    const cache = this.cache.get(conversationId);
    return cache ? cache.messages : [];
  }

  // 获取会话的最后更新时间
  getLastUpdated(conversationId: string): number {
    const cache = this.cache.get(conversationId);
    return cache ? cache.lastUpdated : 0;
  }

  // 检查会话是否存在缓存
  hasCache(conversationId: string): boolean {
    return this.cache.has(conversationId);
  }

  // 添加消息到缓存
  async addMessage(conversationId: string, message: Message | CachedMessage) {
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
      await this.saveCacheToStorage(conversationId);
      console.log(`[MessageCache] Added message ${message.id} to conversation ${conversationId}`);
    } else {
      console.log(`[MessageCache] Message ${message.id} already exists in conversation ${conversationId}`);
    }
  }

  // 批量添加消息到缓存
  async addMessages(conversationId: string, messages: (Message | CachedMessage)[]) {
    let cache = this.cache.get(conversationId);
    if (!cache) {
      cache = {
        conversationId,
        messages: [],
        lastUpdated: Date.now(),
      };
      this.cache.set(conversationId, cache);
    }

    let addedCount = 0;
    // 只添加不存在的消息
    messages.forEach((message) => {
      const exists = cache!.messages.some((m) => m.id === message.id);
      if (!exists) {
        cache!.messages.push(message as CachedMessage);
        addedCount++;
      }
    });

    if (addedCount > 0) {
      cache.lastUpdated = Date.now();
      await this.saveCacheToStorage(conversationId);
      console.log(`[MessageCache] Added ${addedCount} messages to conversation ${conversationId}`);
    } else {
      console.log(`[MessageCache] No new messages to add for conversation ${conversationId}`);
    }
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
    getLastUpdated: messageCacheService.getLastUpdated.bind(messageCacheService),
    hasCache: messageCacheService.hasCache.bind(messageCacheService),
    clearConversation: messageCacheService.clearConversation.bind(messageCacheService),
    clearAll: messageCacheService.clearAll.bind(messageCacheService),
    exportConversation: messageCacheService.exportConversation.bind(messageCacheService),
    exportAll: messageCacheService.exportAll.bind(messageCacheService),
    downloadExport: messageCacheService.downloadExport.bind(messageCacheService),
  };
}
