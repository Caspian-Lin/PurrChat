import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useConnectionStore = defineStore('connection', () => {
  // 连接状态
  const connected = ref(false);
  const connecting = ref(false);
  const lastConnectedTime = ref<number | null>(null);
  const reconnectAttempts = ref(0);

  // 计算属性
  const isOnline = computed(() => connected.value);
  const isOffline = computed(() => !connected.value && !connecting.value);
  const isConnecting = computed(() => connecting.value);

  // 设置连接状态
  function setConnected(status: boolean) {
    connected.value = status;
    if (status) {
      lastConnectedTime.value = Date.now();
      reconnectAttempts.value = 0;
    }
  }

  // 设置连接中状态
  function setConnecting(status: boolean) {
    connecting.value = status;
  }

  // 设置重连尝试次数
  function setReconnectAttempts(attempts: number) {
    reconnectAttempts.value = attempts;
  }

  // 重置连接状态
  function reset() {
    connected.value = false;
    connecting.value = false;
    lastConnectedTime.value = null;
    reconnectAttempts.value = 0;
  }

  // 获取连接状态文本
  function getConnectionStatusText(): string {
    if (connecting.value) {
      return '连接中...';
    }
    if (connected.value) {
      return '在线';
    }
    return '离线';
  }

  return {
    // 状态
    connected,
    connecting,
    lastConnectedTime,
    reconnectAttempts,
    // 计算属性
    isOnline,
    isOffline,
    isConnecting,
    // 方法
    setConnected,
    setConnecting,
    setReconnectAttempts,
    reset,
    getConnectionStatusText,
  };
});
