<template>
  <!-- 移动端布局 -->
  <div v-if="isMobile" class="h-screen">
    <MobileLayout />

    <!-- 个人资料弹窗（移动端也需要） -->
    <UserProfileModal
      :show="showProfile"
      :user="auth.currentUser"
      :is-current-user="true"
      @update:show="showProfile = $event"
      @logout="handleLogout"
    />
  </div>

  <!-- 桌面端布局 -->
  <div v-else class="flex h-screen">
    <SideNavbar :current-user="auth.currentUser" @show-profile="handleShowProfile" />

    <div class="flex-1">
      <router-view />
    </div>

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
import { usePlatform } from '../composables/usePlatform';
import { messageCacheService } from '../services/messageCache';
import { conversationStateCacheService } from '../services/conversationStateCache';
import SideNavbar from '../components/home/SideNavbar.vue';
import UserProfileModal from '../components/home/UserProfileModal.vue';
import MobileLayout from '../layouts/MobileLayout.vue';
import type { Conversation, Friendship } from '../models/types';

// Platform
const { isMobile } = usePlatform();

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

// WebSocket 事件处理器
const handleConversationUpdate = async (conversation: Conversation) => {
  console.log('[HomeView] ===== 会话更新事件 =====');
  console.log('[HomeView] 会话ID:', conversation.id);
  console.log('[HomeView] 会话名称:', conversation.name);
  console.log('[HomeView] 最后消息:', conversation.last_message?.content);
  console.log('[HomeView] 准备重新加载会话列表');
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
    console.log('[HomeView] 新好友请求');
    addNotification('info', '新好友请求', '收到新的好友请求');
    await loadPendingRequests();
  } else if (friendship.status === 'accepted') {
    console.log('[HomeView] 好友请求被接受');
    addNotification('success', '好友请求已接受', '好友请求已接受');
    await loadFriends();
  } else if (friendship.status === 'rejected') {
    console.log('[HomeView] 好友请求被拒绝');
    addNotification('warning', '好友请求已拒绝', '好友请求已拒绝');
    await loadPendingRequests();
  }

  console.log('[HomeView] 重新加载会话列表');
  await loadConversations();
  console.log('[HomeView] ===== 好友请求更新事件处理完成 =====');
};

// Lifecycle
onMounted(async () => {
  await auth.checkAuth();
  if (auth.currentUser) {
    console.log('[HomeView] currentUser 存在，初始化缓存服务');
    messageCacheService.init(auth.currentUser.id);
    conversationStateCacheService.init(auth.currentUser.id);

    console.log('[HomeView] 连接 WebSocket');
    connect();

    onConversationUpdate(handleConversationUpdate);
    onMessageUpdate(handleMessageUpdate);
    onFriendRequest(handleFriendRequestUpdate);
  }
});

onUnmounted(() => {
  console.log('[HomeView] onUnmounted，清理 WebSocket 事件');
  offConversationUpdate(handleConversationUpdate);
  offMessageUpdate(handleMessageUpdate);
  offFriendRequest(handleFriendRequestUpdate);

  disconnect();
});
</script>

<style scoped></style>
