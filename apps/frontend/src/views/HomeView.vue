<template>
  <div class="flex h-screen">
    <!-- 左侧导航栏 -->
    <SideNavbar :current-user="auth.currentUser" @show-profile="handleShowProfile" />

    <!-- 路由视图 - 显示不同的panel -->
    <div class="flex-1">
      <router-view />
    </div>

    <!-- 个人资料弹窗 -->
    <UserProfileModal
      :show="showProfile"
      :user="auth.currentUser"
      :is-current-user="true"
      @update:show="showProfile = $event"
      @logout="handleLogout"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { useAuthController } from '../controllers/authController';
import { useWebSocket } from '../services/websocket';
import { useWebSocketEventManager } from '../services/websocketEventManager';
import { useConversations } from '../composables/useConversations';
import { useFriends } from '../composables/useFriends';
import { useNotification } from '../composables/useNotification';
import { messageCacheService } from '../services/messageCache';
import { conversationStateCacheService } from '../services/conversationStateCache';
import SideNavbar from '../components/home/SideNavbar.vue';
import UserProfileModal from '../components/home/UserProfileModal.vue';
import type { Conversation, Friendship } from '../models/types';

// Auth
const auth = useAuthController();
const { handleLogout } = auth;

// Composables
const { loadConversations } = useConversations();
const { loadFriends, loadPendingRequests } = useFriends();
const { addNotification } = useNotification();
const { connect, disconnect } = useWebSocket();
const {
  onConversationUpdate,
  offConversationUpdate,
  onMessageUpdate,
  offMessageUpdate,
  onFriendRequest,
  offFriendRequest,
} = useWebSocketEventManager();

// Profile modal state
const showProfile = ref(false);

// Handlers
const handleShowProfile = () => {
  showProfile.value = true;
};

// WebSocket事件处理器
const handleConversationUpdate = async (conversation: Conversation) => {
  console.log('[HomeView] ===== 会话更新事件 =====');
  console.log('[HomeView] 会话ID:', conversation.id);
  console.log('[HomeView] 会话名称:', conversation.name);
  console.log('[HomeView] 最后消息:', conversation.last_message?.content);
  console.log('[HomeView] 准备重新加载会话列表');
  // 重新加载会话列表以获取最新数据（包括排序）
  await loadConversations();
  console.log('[HomeView] 会话列表重新加载完成');
  console.log('[HomeView] ===== 会话更新事件处理完成 =====');
};

const handleMessageUpdate = (conversationId: string, message: any) => {
  console.log('[HomeView] ===== 消息更新事件 =====');
  console.log('[HomeView] 消息会话ID:', conversationId);
  console.log('[HomeView] 消息内容:', message.content);
  console.log('[HomeView] 消息ID:', message.id);
  console.log('[HomeView] 发送者ID:', message.sender_id);
  console.log('[HomeView] 当前用户ID:', auth.currentUser?.id);

  // 消息已经通过messageStore更新，这里可以触发额外的UI更新
  // 例如：显示通知
  if (message.sender_id !== auth.currentUser?.id) {
    console.log('[HomeView] 是他人发送的消息，显示通知');
    addNotification('info', '新消息', message.content);
  } else {
    console.log('[HomeView] 是自己发送的消息，不显示通知');
  }
  console.log('[HomeView] ===== 消息更新事件处理完成 =====');
};

const handleFriendRequestUpdate = async (friendship: Friendship) => {
  console.log('[HomeView] ===== 好友请求更新事件 =====');
  console.log('[HomeView] 好友关系ID:', friendship.id);
  console.log('[HomeView] 状态:', friendship.status);

  if (friendship.status === 'pending') {
    // 新好友请求
    console.log('[HomeView] 新好友请求');
    addNotification('info', '新好友请求', `收到新的好友请求`);
    await loadPendingRequests();
  } else if (friendship.status === 'accepted') {
    // 好友请求被接受
    console.log('[HomeView] 好友请求被接受');
    addNotification('success', '好友请求已接受', '好友请求已接受');
    await loadFriends();
  } else if (friendship.status === 'rejected') {
    // 好友请求被拒绝
    console.log('[HomeView] 好友请求被拒绝');
    addNotification('warning', '好友请求已拒绝', '好友请求已拒绝');
    await loadPendingRequests();
  }

  // 重新加载会话列表
  console.log('[HomeView] 重新加载会话列表');
  await loadConversations();
  console.log('[HomeView] ===== 好友请求更新事件处理完成 =====');
};

// Lifecycle
onMounted(async () => {
  await auth.checkAuth();
  if (auth.currentUser) {
    console.log('[HomeView] currentUser 存在，初始化缓存服务');
    // 页面刷新时，用当前用户 ID 初始化缓存服务
    messageCacheService.init(auth.currentUser.id);
    conversationStateCacheService.init(auth.currentUser.id);

    console.log('[HomeView] 连接 WebSocket');
    // 连接WebSocket
    connect();

    // 注册WebSocket事件管理器的回调
    onConversationUpdate(handleConversationUpdate);
    onMessageUpdate(handleMessageUpdate);
    onFriendRequest(handleFriendRequestUpdate);
  }
});

onUnmounted(() => {
  console.log('[HomeView] onUnmounted，清理 WebSocket 事件');
  // 移除WebSocket事件管理器的回调
  offConversationUpdate(handleConversationUpdate);
  offMessageUpdate(handleMessageUpdate);
  offFriendRequest(handleFriendRequestUpdate);

  // 断开WebSocket连接
  disconnect();
});
</script>

<style scoped></style>
