import { ref } from 'vue';
import type { Message } from '../models/types';
import {
  messageKeyPrefix,
  messageKey,
  messageEncryptionKey,
  clearUserData as clearUserDataByKey,
} from '../utils/storageNamespace';

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
  private cryptoKey: CryptoKey | null = null;
  private keyInitialized = false;
  private currentUserId: string | null = null;
  private initialized = false;

  constructor() {
    // 不再自动初始化，等待 init(userId) 调用
  }

  /**
   * 初始化服务（首次或用户切换时调用）
   * 不会清除其他用户的数据，只加载当前用户的缓存
   */
  async init(userId: string) {
    console.log('[MessageCache] init called for user:', userId);
    this.currentUserId = userId;
    this.cache.clear();
    this.cryptoKey = null;
    this.keyInitialized = false;
    this.initialized = false;

    await this.initCryptoKey();
    await this.loadCacheFromStorage();
    this.initialized = true;
  }

  /** 当前用户是否有已初始化的缓存 */
  isInitialized(): boolean {
    return this.initialized && this.keyInitialized;
  }

  /** 获取当前用户 ID */
  getCurrentUserId(): string | null {
    return this.currentUserId;
  }

  private getStorageKeyPrefix(): string {
    return this.currentUserId ? messageKeyPrefix(this.currentUserId) : 'message_cache_';
  }

  private getEncryptionKey(): string {
    return this.currentUserId ? messageEncryptionKey(this.currentUserId) : 'message_encryption_key';
  }

  // 初始化加密密钥
  private async initCryptoKey() {
    const encKey = this.getEncryptionKey();
    try {
      const keyData = localStorage.getItem(encKey);
      if (keyData) {
        try {
          this.cryptoKey = await this.importKey(keyData);
        } catch {
          // 密钥损坏，清除密钥（不删除消息缓存，消息以明文保存时不需要密钥）
          localStorage.removeItem(encKey);
          try {
            this.cryptoKey = await this.generateCryptoKey();
            const exportedKey = await this.exportKey(this.cryptoKey);
            localStorage.setItem(encKey, exportedKey);
          } catch {
            this.cryptoKey = null;
          }
        }
      } else {
        try {
          this.cryptoKey = await this.generateCryptoKey();
          const exportedKey = await this.exportKey(this.cryptoKey);
          localStorage.setItem(encKey, exportedKey);
        } catch {
          this.cryptoKey = null;
        }
      }
      this.keyInitialized = true;
    } catch (error) {
      console.error('[MessageCache] Failed to initialize crypto key:', error);
      this.cryptoKey = null;
      this.keyInitialized = true;
    }
  }

  // 等待密钥初始化完成
  private async waitForKeyInitialization() {
    let attempts = 0;
    const maxAttempts = 1000;
    while (!this.keyInitialized && attempts < maxAttempts) {
      await new Promise((resolve) => setTimeout(resolve, 10));
      attempts++;
    }
    if (!this.keyInitialized) {
      this.keyInitialized = true;
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
      const combined = Uint8Array.from(atob(encryptedData), (c) => c.charCodeAt(0));
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

  // 从localStorage加载当前用户的缓存
  private async loadCacheFromStorage() {
    try {
      await this.waitForKeyInitialization();
      const prefix = this.getStorageKeyPrefix();
      const keys = Object.keys(localStorage);
      for (const key of keys) {
        if (key.startsWith(prefix)) {
          const conversationId = key.slice(prefix.length);
          const encryptedData = localStorage.getItem(key);
          if (encryptedData) {
            try {
              const decryptedData = await this.decrypt(encryptedData);
              const cache: ConversationCache = JSON.parse(decryptedData);
              this.cache.set(conversationId, cache);
            } catch (error) {
              console.error(
                '[MessageCache] Failed to load cache for conversation:',
                conversationId,
                error
              );
              localStorage.removeItem(key);
            }
          }
        }
      }
      console.log(
        `[MessageCache] Loaded message cache for ${this.cache.size} conversations (user: ${this.currentUserId})`
      );
    } catch (error) {
      console.error('[MessageCache] Failed to load cache from storage:', error);
    }
  }

  // 保存缓存到localStorage
  private async saveCacheToStorage(conversationId: string) {
    const cache = this.cache.get(conversationId);
    if (!cache) return;

    try {
      const data = JSON.stringify(cache);
      const encryptedData = await this.encrypt(data);
      const prefix = this.getStorageKeyPrefix();
      localStorage.setItem(`${prefix}${conversationId}`, encryptedData);
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

    const exists = cache.messages.some((m) => m.id === message.id);
    if (!exists) {
      cache.messages.push(message as CachedMessage);
      cache.lastUpdated = Date.now();
      await this.saveCacheToStorage(conversationId);
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
    }
  }

  /**
   * 从缓存中移除服务器上不存在的消息
   * @param conversationId 会话 ID
   * @param serverMessageIds 服务器返回的消息 ID 集合
   * @returns 被移除的消息数量
   */
  async reconcileWithServer(
    conversationId: string,
    serverMessageIds: Set<string>
  ): Promise<number> {
    const cache = this.cache.get(conversationId);
    if (!cache) return 0;

    const before = cache.messages.length;
    cache.messages = cache.messages.filter((m) => serverMessageIds.has(m.id));
    const removed = before - cache.messages.length;

    if (removed > 0) {
      cache.lastUpdated = Date.now();
      await this.saveCacheToStorage(conversationId);
      console.log(
        `[MessageCache] Reconciled ${removed} messages for conversation ${conversationId}`
      );
    }
    return removed;
  }

  // 清除会话的缓存
  clearConversation(conversationId: string) {
    this.cache.delete(conversationId);
    const prefix = this.getStorageKeyPrefix();
    localStorage.removeItem(`${prefix}${conversationId}`);
  }

  // 清除当前用户的所有缓存
  clearAll() {
    this.cache.clear();
    const prefix = this.getStorageKeyPrefix();
    const keys = Object.keys(localStorage);
    keys.forEach((key) => {
      if (key.startsWith(prefix)) {
        localStorage.removeItem(key);
      }
    });
  }

  // 清除当前用户的加密密钥和所有相关缓存
  private clearCurrentUserEncryptionData() {
    localStorage.removeItem(this.getEncryptionKey());
    this.clearAll();
    this.cryptoKey = null;
    this.keyInitialized = false;
  }

  // 导出会话消息为JSON文件
  exportConversation(conversationId: string): string | null {
    const cache = this.cache.get(conversationId);
    if (!cache) return null;
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
    init: messageCacheService.init.bind(messageCacheService),
    isInitialized: messageCacheService.isInitialized.bind(messageCacheService),
    getMessages: messageCacheService.getMessages.bind(messageCacheService),
    addMessage: messageCacheService.addMessage.bind(messageCacheService),
    addMessages: messageCacheService.addMessages.bind(messageCacheService),
    reconcileWithServer: messageCacheService.reconcileWithServer.bind(messageCacheService),
    getLastUpdated: messageCacheService.getLastUpdated.bind(messageCacheService),
    hasCache: messageCacheService.hasCache.bind(messageCacheService),
    clearConversation: messageCacheService.clearConversation.bind(messageCacheService),
    clearAll: messageCacheService.clearAll.bind(messageCacheService),
    exportConversation: messageCacheService.exportConversation.bind(messageCacheService),
    exportAll: messageCacheService.exportAll.bind(messageCacheService),
    downloadExport: messageCacheService.downloadExport.bind(messageCacheService),
  };
}
