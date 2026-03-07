<template>
  <div class="flex h-screen">
    <!-- 左侧导航栏 -->
    <SideNavbar :current-user="auth.currentUser.value" @show-profile="handleShowProfile" />

    <!-- 路由视图 - 显示不同的panel -->
    <div class="flex-1">
      <router-view />
    </div>

    <!-- 个人资料弹窗 -->
    <UserProfileModal
      :show="showProfile"
      :user="auth.currentUser.value"
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
import { useConversations } from '../composables/useConversations';
import { useFriends } from '../composables/useFriends';
import { useNotification } from '../composables/useNotification';
import SideNavbar from '../components/home/SideNavbar.vue';
import UserProfileModal from '../components/home/UserProfileModal.vue';

// Auth
const auth = useAuthController();
const { handleLogout } = auth;

// Composables
const { loadConversations } = useConversations();
const { loadFriends, loadPendingRequests } = useFriends();
const { addNotification } = useNotification();
const { connect, disconnect, on: onWs, off: offWs } = useWebSocket();

// Profile modal state
const showProfile = ref(false);

// Handlers
const handleShowProfile = () => {
  showProfile.value = true;
};

// WebSocket handlers
const handleNewMessage = async (data: any) => {
  console.log('[HomeView] New message received via WebSocket:', data);
  // 显示通知
  addNotification('info', '新消息', '收到新消息');
  // 重新加载会话列表
  await loadConversations();
};

const handleNewFriendRequest = async (data: any) => {
  console.log('[HomeView] New friend request received:', data);
  // 显示通知
  addNotification('info', '新好友请求', `收到来自 ${data.sender_id} 的好友请求`);
  // 重新加载会话列表
  await loadConversations();
  // 重新加载待处理请求
  await loadPendingRequests();
};

const handleFriendRequestUpdate = async (data: any) => {
  console.log('[HomeView] Friend request update received:', data);
  // 显示通知
  if (data.status === 'accepted') {
    addNotification('success', '好友请求已接受', `${data.sender_id} 接受了你的好友请求`);
  } else if (data.status === 'rejected') {
    addNotification('warning', '好友请求已拒绝', `${data.sender_id} 拒绝了你的好友请求`);
  }
  // 重新加载会话列表
  await loadConversations();
  // 重新加载好友列表
  await loadFriends();
};

const handleNewGroupConversation = async (data: any) => {
  console.log('[HomeView] New group conversation received:', data);
  // 显示通知
  addNotification('success', '群聊创建成功', `群聊 ${data.name} 创建成功`);
  // 重新加载会话列表
  await loadConversations();
};

const handleConversationMemberAdded = async (data: any) => {
  console.log('[HomeView] Conversation member added:', data);
  // 显示通知
  addNotification('info', '成员已添加', `新成员已加入群聊`);
  // 重新加载会话列表
  await loadConversations();
};

const handleConversationMemberRemoved = async (data: any) => {
  console.log('[HomeView] Conversation member removed:', data);
  // 显示通知
  if (data.user_id === auth.currentUser.value?.id) {
    addNotification('warning', '已移出群聊', '你已被移出群聊');
  } else {
    addNotification('info', '成员已移除', `成员已从群聊中移除`);
  }
  // 重新加载会话列表
  await loadConversations();
};

// Lifecycle
onMounted(async () => {
  await auth.checkAuth();
  if (auth.currentUser.value) {
    console.log('[HomeView] currentUser 存在，连接 WebSocket');
    // 连接WebSocket
    connect();

    // 注册WebSocket事件处理器
    onWs('new_message', handleNewMessage);
    onWs('new_friend_request', handleNewFriendRequest);
    onWs('friend_request_update', handleFriendRequestUpdate);
    onWs('new_group_conversation', handleNewGroupConversation);
    onWs('conversation_member_added', handleConversationMemberAdded);
    onWs('conversation_member_removed', handleConversationMemberRemoved);
  }
});

onUnmounted(() => {
  console.log('[HomeView] onUnmounted，断开 WebSocket');
  // 移除WebSocket事件处理器
  offWs('new_message', handleNewMessage);
  offWs('new_friend_request', handleNewFriendRequest);
  offWs('friend_request_update', handleFriendRequestUpdate);
  offWs('new_group_conversation', handleNewGroupConversation);
  offWs('conversation_member_added', handleConversationMemberAdded);
  offWs('conversation_member_removed', handleConversationMemberRemoved);

  // 断开WebSocket连接
  disconnect();
});
</script>

<style scoped></style>
