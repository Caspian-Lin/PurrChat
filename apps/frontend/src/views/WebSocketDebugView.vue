<template>
  <div class="p-6 bg-bg-primary min-h-screen">
    <h1 class="text-2xl font-bold mb-4 text-text-primary">WebSocket 调试工具</h1>

    <!-- 连接状态 -->
    <div class="mb-6 p-4 bg-bg-secondary rounded-lg">
      <h2 class="text-lg font-semibold mb-2 text-text-primary">连接状态</h2>
      <div class="flex gap-4">
        <div>
          <span class="text-text-secondary">连接状态:</span>
          <span
            :class="[
              'ml-2 px-2 py-1 rounded',
              connected ? 'bg-green-500 text-white' : 'bg-red-500 text-white',
            ]"
          >
            {{ connected ? '已连接' : '未连接' }}
          </span>
        </div>
        <div>
          <span class="text-text-secondary">当前会话ID:</span>
          <span class="ml-2 text-text-primary">{{ currentConversationId || '未设置' }}</span>
        </div>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="mb-6 flex gap-4">
      <button
        @click="connect"
        :disabled="connected"
        class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
      >
        连接 WebSocket
      </button>
      <button
        @click="disconnect"
        :disabled="!connected"
        class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50"
      >
        断开连接
      </button>
      <button @click="clearLogs" class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600">
        清空日志
      </button>
    </div>

    <!-- 设置当前会话 -->
    <div class="mb-6 p-4 bg-bg-secondary rounded-lg">
      <h2 class="text-lg font-semibold mb-2 text-text-primary">设置当前会话</h2>
      <div class="flex gap-2">
        <input
          v-model="conversationIdInput"
          type="text"
          placeholder="输入会话ID"
          class="flex-1 px-3 py-2 bg-bg-tertiary text-text-primary rounded border border-border-color"
        />
        <button
          @click="setCurrentConversation"
          class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"
        >
          设置
        </button>
      </div>
    </div>

    <!-- 发送测试消息 -->
    <div class="mb-6 p-4 bg-bg-secondary rounded-lg">
      <h2 class="text-lg font-semibold mb-2 text-text-primary">发送测试消息</h2>
      <div class="flex gap-2">
        <input
          v-model="testMessageContent"
          type="text"
          placeholder="输入测试消息内容"
          class="flex-1 px-3 py-2 bg-bg-tertiary text-text-primary rounded border border-border-color"
        />
        <button
          @click="sendTestMessage"
          :disabled="!testConversationId"
          class="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600 disabled:opacity-50"
        >
          发送
        </button>
      </div>
      <div class="mt-2">
        <input
          v-model="testConversationId"
          type="text"
          placeholder="目标会话ID"
          class="w-full px-3 py-2 bg-bg-tertiary text-text-primary rounded border border-border-color"
        />
      </div>
    </div>

    <!-- 事件日志 -->
    <div class="p-4 bg-bg-secondary rounded-lg">
      <h2 class="text-lg font-semibold mb-2 text-text-primary">事件日志</h2>
      <div class="h-96 overflow-y-auto bg-bg-tertiary rounded p-2 font-mono text-sm">
        <div
          v-for="(log, index) in logs"
          :key="index"
          :class="[
            'mb-1 p-1 rounded',
            {
              'bg-blue-100 text-blue-900': log.type === 'info',
              'bg-green-100 text-green-900': log.type === 'success',
              'bg-red-100 text-red-900': log.type === 'error',
              'bg-yellow-100 text-yellow-900': log.type === 'warning',
            },
          ]"
        >
          <span class="text-text-tertiary">[{{ log.time }}]</span>
          <span class="font-semibold">[{{ log.type }}]</span>
          <span>{{ log.message }}</span>
        </div>
        <div v-if="logs.length === 0" class="text-text-tertiary">暂无日志</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useWebSocket } from '../services/websocket';
import { useWebSocketEventManager } from '../services/websocketEventManager';
import { useAuthStore } from '../stores/auth';
import { api } from '../models/api';

const { connect: wsConnect, disconnect: wsDisconnect, connected } = useWebSocket();
const {
  setCurrentConversation: setWsCurrentConversation,
  onConversationUpdate,
  onMessageUpdate,
  onFriendRequest,
} = useWebSocketEventManager();
const authStore = useAuthStore();

const currentConversationId = ref<string | null>(null);
const conversationIdInput = ref('');
const testMessageContent = ref('');
const testConversationId = ref('');

interface LogEntry {
  time: string;
  type: 'info' | 'success' | 'error' | 'warning';
  message: string;
}

const logs = ref<LogEntry[]>([]);

function addLog(type: LogEntry['type'], message: string) {
  const now = new Date();
  const time = now.toLocaleTimeString('zh-CN', { hour12: false });
  logs.value.push({ time, type, message });
  // 自动滚动到底部
  setTimeout(() => {
    const container = document.querySelector('.overflow-y-auto');
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  }, 100);
}

function connect() {
  addLog('info', '尝试连接 WebSocket...');
  if (authStore.token && authStore.user) {
    wsConnect();
    addLog('success', `WebSocket 连接请求已发送 (用户ID: ${authStore.user.id})`);
  } else {
    addLog('error', '无法连接：用户未登录');
  }
}

function disconnect() {
  addLog('info', '断开 WebSocket 连接...');
  wsDisconnect();
  addLog('success', 'WebSocket 已断开');
}

function clearLogs() {
  logs.value = [];
  addLog('info', '日志已清空');
}

function setCurrentConversation() {
  if (conversationIdInput.value) {
    currentConversationId.value = conversationIdInput.value;
    setWsCurrentConversation(conversationIdInput.value);
    addLog('success', `当前会话已设置为: ${conversationIdInput.value}`);
  } else {
    addLog('warning', '请输入会话ID');
  }
}

async function sendTestMessage() {
  if (!testConversationId.value) {
    addLog('warning', '请输入目标会话ID');
    return;
  }
  if (!testMessageContent.value) {
    addLog('warning', '请输入消息内容');
    return;
  }

  try {
    addLog('info', `发送测试消息到会话 ${testConversationId.value}: ${testMessageContent.value}`);
    const response = await api.sendMessage({
      conversation_id: testConversationId.value,
      content: testMessageContent.value,
      msg_type: 'text',
    });
    if (response.success) {
      addLog('success', '测试消息发送成功');
      testMessageContent.value = '';
    } else {
      addLog('error', `发送失败: ${response.message}`);
    }
  } catch (error) {
    addLog('error', `发送失败: ${error}`);
  }
}

// WebSocket 事件处理器
const handleConversationUpdate = (conversation: any) => {
  addLog('info', `会话更新事件: ${conversation.id} - ${conversation.name}`);
};

const handleMessageUpdate = (conversationId: string, message: any) => {
  addLog(
    'success',
    `消息更新事件: 会话 ${conversationId} - 消息 ${message.id} - ${message.content}`
  );
};

const handleFriendRequest = (friendship: any) => {
  addLog('warning', `好友请求事件: ${friendship.id} - 状态 ${friendship.status}`);
};

// 生命周期
onMounted(() => {
  addLog('info', 'WebSocket 调试工具已加载');

  // 注册事件处理器
  onConversationUpdate(handleConversationUpdate);
  onMessageUpdate(handleMessageUpdate);
  onFriendRequest(handleFriendRequest);

  addLog('info', '事件处理器已注册');

  // 检查用户登录状态
  if (authStore.user) {
    addLog('info', `当前用户: ${authStore.user.username} (ID: ${authStore.user.id})`);
  } else {
    addLog('warning', '用户未登录');
  }
});

onUnmounted(() => {
  addLog('info', 'WebSocket 调试工具已卸载');
});
</script>
