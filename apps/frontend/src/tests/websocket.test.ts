import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

vi.mock('../config/app', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../config/app')>();
  return {
    ...actual,
    getWebSocketUrl: () => 'ws://localhost:8080/api/ws',
    logger: {
      log: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      info: vi.fn(),
    },
  };
});

vi.mock('../platform', () => ({
  getCurrentPlatformCapabilities: () => ({ runtime: { isNative: false } }),
}));

vi.mock('../stores/connection', () => ({
  useConnectionStore: () => ({
    setConnected: vi.fn(),
    setConnecting: vi.fn(),
    setReconnectAttempts: vi.fn(),
  }),
}));

import { WebSocketService } from '../services/websocket';
import { useAuthStore } from '../stores/auth';

class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  static instances: MockWebSocket[] = [];

  readonly url: string;
  readyState = MockWebSocket.CONNECTING;
  onopen: ((_event: Event) => void) | null = null;
  onmessage: ((_event: MessageEvent) => void) | null = null;
  onerror: ((_event: Event) => void) | null = null;
  onclose: ((_event: CloseEvent) => void) | null = null;
  send = vi.fn();
  close = vi.fn((code?: number, reason?: string) => {
    void code;
    void reason;
    this.readyState = MockWebSocket.CLOSED;
  });

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  open() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event('open'));
  }

  closeFromServer(code: number, reason = '') {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code, reason } as CloseEvent);
  }
}

describe('WebSocketService', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.useFakeTimers();
    vi.spyOn(Math, 'random').mockReturnValue(0.5);
    vi.stubGlobal('WebSocket', MockWebSocket);
    MockWebSocket.instances = [];

    const auth = useAuthStore();
    auth.user = { id: 'user-1', username: 'tester' } as typeof auth.user;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('does not create another socket while connecting or connected', () => {
    const service = new WebSocketService();

    service.connect();
    service.connect();
    expect(MockWebSocket.instances).toHaveLength(1);

    MockWebSocket.instances[0].open();
    service.connect();
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('does not reconnect when a newer same-device connection replaces it', () => {
    const service = new WebSocketService();
    service.connect();
    MockWebSocket.instances[0].open();

    MockWebSocket.instances[0].closeFromServer(4001, 'connection replaced by newer session');
    vi.runAllTimers();

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(service.connected.value).toBe(false);
  });

  it('keeps only one reconnect timer for repeated close callbacks', () => {
    const service = new WebSocketService();
    service.connect();
    const socket = MockWebSocket.instances[0];
    socket.open();

    socket.closeFromServer(1006);
    socket.closeFromServer(1006);

    expect(vi.getTimerCount()).toBe(1);
    vi.advanceTimersByTime(1000);
    expect(MockWebSocket.instances).toHaveLength(2);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('still retries transient capacity close responses', () => {
    const service = new WebSocketService();
    service.connect();
    MockWebSocket.instances[0].open();

    MockWebSocket.instances[0].closeFromServer(1013, 'server at capacity');
    vi.advanceTimersByTime(1000);

    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it('cancels a pending reconnect when reconnecting explicitly', () => {
    const service = new WebSocketService();
    service.connect();
    MockWebSocket.instances[0].open();
    MockWebSocket.instances[0].closeFromServer(1006);
    expect(vi.getTimerCount()).toBe(1);

    service.connect();
    expect(MockWebSocket.instances).toHaveLength(2);
    expect(vi.getTimerCount()).toBe(0);

    vi.runAllTimers();
    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it('cancels pending reconnect work on manual disconnect', () => {
    const service = new WebSocketService();
    service.connect();
    MockWebSocket.instances[0].open();
    MockWebSocket.instances[0].closeFromServer(1006);
    expect(vi.getTimerCount()).toBe(1);

    service.disconnect();
    expect(vi.getTimerCount()).toBe(0);

    vi.runAllTimers();
    expect(MockWebSocket.instances).toHaveLength(1);
  });
});
