import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useConnectionStore } from '../stores/connection';

describe('Connection Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  describe('Initial State', () => {
    it('should have correct default values', () => {
      const store = useConnectionStore();
      expect(store.connected).toBe(false);
      expect(store.connecting).toBe(false);
      expect(store.lastConnectedTime).toBeNull();
      expect(store.reconnectAttempts).toBe(0);
    });

    it('should have correct computed values initially', () => {
      const store = useConnectionStore();
      expect(store.isOnline).toBe(false);
      expect(store.isOffline).toBe(true);
      expect(store.isConnecting).toBe(false);
    });
  });

  describe('setConnected', () => {
    it('should set connected to true and update lastConnectedTime', () => {
      const store = useConnectionStore();
      const now = Date.now();
      vi.spyOn(Date, 'now').mockReturnValue(now);

      store.setConnected(true);

      expect(store.connected).toBe(true);
      expect(store.lastConnectedTime).toBe(now);
      expect(store.reconnectAttempts).toBe(0);
      expect(store.isOnline).toBe(true);
      expect(store.isOffline).toBe(false);
    });

    it('should set connected to false without changing lastConnectedTime', () => {
      const store = useConnectionStore();
      store.setConnected(true);
      const firstTime = store.lastConnectedTime;

      store.setConnected(false);

      expect(store.connected).toBe(false);
      expect(store.lastConnectedTime).toBe(firstTime);
    });

    it('should reset reconnectAttempts when connecting', () => {
      const store = useConnectionStore();
      store.setReconnectAttempts(5);

      store.setConnected(true);

      expect(store.reconnectAttempts).toBe(0);
    });

    it('should update lastConnectedTime on each successful connection', () => {
      const store = useConnectionStore();
      vi.spyOn(Date, 'now')
        .mockReturnValueOnce(1000)
        .mockReturnValueOnce(2000);

      store.setConnected(true);
      expect(store.lastConnectedTime).toBe(1000);

      store.setConnected(false);
      store.setConnected(true);
      expect(store.lastConnectedTime).toBe(2000);
    });
  });

  describe('setConnecting', () => {
    it('should set connecting state', () => {
      const store = useConnectionStore();
      store.setConnecting(true);
      expect(store.connecting).toBe(true);
      expect(store.isConnecting).toBe(true);
      expect(store.isOffline).toBe(false);
    });

    it('should clear connecting state', () => {
      const store = useConnectionStore();
      store.setConnecting(true);
      store.setConnecting(false);
      expect(store.connecting).toBe(false);
      expect(store.isConnecting).toBe(false);
    });
  });

  describe('setReconnectAttempts', () => {
    it('should set reconnect attempt count', () => {
      const store = useConnectionStore();
      store.setReconnectAttempts(3);
      expect(store.reconnectAttempts).toBe(3);
    });
  });

  describe('reset', () => {
    it('should reset all state to defaults', () => {
      const store = useConnectionStore();
      store.setConnected(true);
      store.setReconnectAttempts(5);

      store.reset();

      expect(store.connected).toBe(false);
      expect(store.connecting).toBe(false);
      expect(store.lastConnectedTime).toBeNull();
      expect(store.reconnectAttempts).toBe(0);
    });
  });

  describe('getConnectionStatusText', () => {
    it('should return "连接中..." when connecting', () => {
      const store = useConnectionStore();
      store.setConnecting(true);
      expect(store.getConnectionStatusText()).toBe('连接中...');
    });

    it('should return "在线" when connected', () => {
      const store = useConnectionStore();
      store.setConnected(true);
      expect(store.getConnectionStatusText()).toBe('在线');
    });

    it('should return "离线" when not connected and not connecting', () => {
      const store = useConnectionStore();
      expect(store.getConnectionStatusText()).toBe('离线');
    });
  });

  describe('Computed properties', () => {
    it('isOnline should equal connected', () => {
      const store = useConnectionStore();
      expect(store.isOnline).toBe(store.connected);
      store.setConnected(true);
      expect(store.isOnline).toBe(store.connected);
    });

    it('isOffline should be true only when not connected and not connecting', () => {
      const store = useConnectionStore();
      expect(store.isOffline).toBe(true);

      store.setConnected(true);
      expect(store.isOffline).toBe(false);

      store.setConnected(false);
      store.setConnecting(true);
      expect(store.isOffline).toBe(false);
    });
  });
});
