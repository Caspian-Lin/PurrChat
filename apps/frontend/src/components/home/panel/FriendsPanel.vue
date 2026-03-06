<template>
  <div class="flex h-full">
    <!-- 好友列表 -->
    <ResizableContainer
      direction="horizontal"
      :initial-size="320"
      :min-size="250"
      :max-size="500"
      class="bg-bg-primary border-r border-border-color"
    >
      <!-- 搜索好友 -->
      <div
        class="flex items-center gap-2 p-3 bg-bg-secondary border-b border-border-color relative"
      >
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索好友或用户..."
          class="flex-1 bg-bg-quaternary rounded-md h-[40px] px-3 text-text-primary placeholder-text-tertiary focus:outline-none focus:ring-2 focus:ring-primary"
          @input="handleSearch"
          @focus="showSearchResults = true"
        />
        <button
          v-if="searchQuery"
          class="p-2 text-text-tertiary hover:text-text-primary transition-colors"
          @click="clearSearch"
        >
          ✕
        </button>
      </div>
      <!-- 好友消息条目（待处理好友申请） -->
      <div
        class="flex items-center gap-4 p-4 bg-bg-secondary border-b border-border-color cursor-pointer hover:bg-hover-bg transition-colors"
        @click="showFriendRequestHistory = true"
      >
        <div class="relative">
          <div
            class="w-12 h-12 rounded-full flex items-center justify-center text-white font-bold"
            style="background: var(--theme-gradient)"
          >
            🔔
          </div>
          <div
            class="absolute -top-1 -right-1 w-5 h-5 bg-red-500 rounded-full flex items-center justify-center text-white text-xs font-bold"
          >
            {{ pendingRequests.length }}
          </div>
        </div>
        <div class="flex-1 min-w-0">
          <div class="font-semibold truncate text-text-primary">好友申请</div>
          <div class="text-sm text-text-secondary">{{ pendingRequests.length }} 条待处理</div>
        </div>
        <div class="text-text-tertiary">></div>
      </div>

      <!-- 搜索结果dropdown -->
      <div
        v-if="showSearchResults && (filteredFriends.length > 0 || searchedUsers.length > 0)"
        class="absolute top-[80px] left-0 right-0 bg-bg-primary border border-border-color rounded-lg shadow-lg z-50 max-h-[400px] overflow-y-auto"
        style="width: 300px"
      >
        <!-- 好友列表 -->
        <div v-if="filteredFriends.length > 0" class="border-b border-border-color">
          <div class="px-3 py-2 text-sm font-semibold text-text-secondary bg-bg-secondary">
            好友
          </div>
          <div
            v-for="friendship in filteredFriends"
            :key="'friend-' + friendship.id"
            class="flex items-center gap-3 p-3 hover:bg-hover-bg cursor-pointer transition-colors"
            @click="handleSelectFriend(friendship)"
          >
            <div class="w-10 h-10 rounded-full overflow-hidden flex-shrink-0">
              <img
                v-if="friendship.friend?.avatar_url"
                :src="friendship.friend.avatar_url"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center font-bold text-white"
                style="background: var(--theme-gradient)"
              >
                {{ friendship.friend?.username?.charAt(0) || '?' }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-semibold truncate text-text-primary">
                {{ friendship.friend?.username }}
              </div>
              <div class="text-sm text-text-secondary">UID: {{ friendship.friend?.uid }}</div>
            </div>
          </div>
        </div>

        <!-- 搜索到的用户列表 -->
        <div v-if="searchedUsers.length > 0">
          <div class="px-3 py-2 text-sm font-semibold text-text-secondary bg-bg-secondary">
            用户
          </div>
          <div
            v-for="user in searchedUsers"
            :key="'user-' + user.id"
            class="flex items-center gap-3 p-3 hover:bg-hover-bg cursor-pointer transition-colors"
            @click="handleSelectUser(user)"
          >
            <div class="w-10 h-10 rounded-full overflow-hidden flex-shrink-0">
              <img
                v-if="user.avatar_url"
                :src="user.avatar_url"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center font-bold text-white"
                style="background: var(--theme-gradient)"
              >
                {{ user.username?.charAt(0) || '?' }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-semibold truncate text-text-primary">{{ user.username }}</span>
                <span class="text-xs px-2 py-0.5 bg-orange-500 text-white rounded">陌生人</span>
              </div>
              <div class="text-sm text-text-secondary">UID: {{ user.uid }}</div>
            </div>
          </div>
        </div>

        <!-- 无结果 -->
        <div
          v-if="filteredFriends.length === 0 && searchedUsers.length === 0"
          class="p-4 text-center text-text-tertiary"
        >
          未找到匹配的用户
        </div>
      </div>

      <!-- 好友列表 -->
      <FriendList
        :friends="friends"
        @select="handleSelectFriend"
        @show-user="handleShowUserProfile"
      />
    </ResizableContainer>

    <!-- 好友信息窗口 -->
    <div class="flex-1 flex flex-col bg-bg-tertiary">
      <!-- 好友申请历史 -->
      <div v-if="showFriendRequestHistory" class="flex-1 flex flex-col p-6 overflow-y-auto">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-2xl font-bold text-text-primary">好友申请记录</h2>
          <button
            class="text-text-tertiary hover:text-text-primary transition-colors"
            @click="showFriendRequestHistory = false"
          >
            ✕
          </button>
        </div>

        <div
          v-if="allFriendRequests.length === 0"
          class="flex-1 flex items-center justify-center text-text-tertiary"
        >
          <p>暂无好友申请记录</p>
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="request in allFriendRequests"
            :key="request.id"
            class="flex items-center gap-4 p-4 bg-bg-secondary rounded-lg"
          >
            <div
              class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0 cursor-pointer"
              @click="handleShowUserProfile(request.user!)"
            >
              <img
                v-if="request.user?.avatar_url"
                :src="request.user.avatar_url"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center font-bold text-white"
                style="background: var(--theme-gradient)"
              >
                {{ request.user?.username?.charAt(0) || '?' }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-semibold text-text-primary">{{ request.user?.username }}</div>
              <div class="text-sm text-text-secondary">
                {{ getFriendRequestText(request) }}
              </div>
              <div class="text-xs text-text-tertiary">
                {{ formatTime(request.created_at) }}
              </div>
            </div>
            <div
              :class="[
                'px-3 py-1 rounded-md text-sm font-medium',
                getFriendRequestStatusClass(request.status),
              ]"
            >
              {{ getFriendRequestStatusText(request.status) }}
            </div>
          </div>
        </div>
      </div>

      <FriendInfoModal
        v-else-if="selectedFriend"
        :friendship="selectedFriend"
        @close="selectedFriend = null"
        @start-chat="handleStartChatWithFriend"
      />

      <!-- 空状态 -->
      <div v-else class="flex-1 flex flex-col items-center justify-center text-text-tertiary">
        <div class="text-6xl mb-4">👥</div>
        <h3 class="text-2xl font-semibold mb-2 text-text-primary">好友列表</h3>
        <p>选择一个好友查看详情或开始聊天</p>
      </div>
    </div>

    <!-- 个人资料弹窗 -->
    <UserProfileModal v-model:show="showProfileModal" :user="displayUser" />

    <!-- 陌生人弹窗（显示申请好友按钮） -->
    <div
      v-if="showStrangerModal && selectedStranger"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showStrangerModal = false"
    >
      <div class="bg-bg-primary rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-bold text-text-primary">用户信息</h2>
          <button
            class="text-text-tertiary hover:text-text-primary transition-colors"
            @click="showStrangerModal = false"
          >
            ✕
          </button>
        </div>
        <div class="flex flex-col items-center gap-4">
          <div class="w-24 h-24 rounded-full overflow-hidden">
            <img
              v-if="selectedStranger.avatar_url"
              :src="selectedStranger.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white text-4xl"
              style="background: var(--theme-gradient)"
            >
              {{ selectedStranger.username?.charAt(0) || '?' }}
            </div>
          </div>
          <div class="w-full space-y-3">
            <div class="flex justify-between p-3 rounded-lg bg-bg-secondary">
              <span class="font-semibold text-text-secondary">昵称:</span>
              <span class="text-text-primary">{{ selectedStranger.username }}</span>
            </div>
            <div class="flex justify-between p-3 rounded-lg bg-bg-secondary">
              <span class="font-semibold text-text-secondary">UID:</span>
              <span class="text-text-primary">{{ selectedStranger.uid }}</span>
            </div>
            <div
              v-if="selectedStranger.email"
              class="flex justify-between p-3 rounded-lg bg-bg-secondary"
            >
              <span class="font-semibold text-text-secondary">邮箱:</span>
              <span class="text-text-primary">
                {{ selectedStranger.email }}
                <span v-if="!selectedStranger.email_verified" class="text-text-tertiary text-sm"
                  >(未验证)</span
                >
              </span>
            </div>
            <div
              v-if="selectedStranger.phone"
              class="flex justify-between p-3 rounded-lg bg-bg-secondary"
            >
              <span class="font-semibold text-text-secondary">手机号:</span>
              <span class="text-text-primary">
                {{ selectedStranger.phone }}
                <span v-if="!selectedStranger.phone_verified" class="text-text-tertiary text-sm"
                  >(未验证)</span
                >
              </span>
            </div>
          </div>
          <button
            class="w-full py-2 bg-primary text-white rounded-md hover:bg-primary-dark transition-colors"
            @click="handleSendFriendRequest"
          >
            添加好友
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { useAuthController } from '../../../controllers/authController';
import { useFriends } from '../../../composables/useFriends';
import { useConversations } from '../../../composables/useConversations';
import { useWebSocket } from '../../../services/websocket';
import { api } from '../../../models/api';
import { useRouter } from 'vue-router';
import FriendList from '../FriendList.vue';
import FriendInfoModal from '../FriendInfoModal.vue';
import UserProfileModal from '../UserProfileModal.vue';
import ResizableContainer from '../../common/ResizableContainer.vue';
import type { User, Friendship } from '../../../models/types';

// Auth
const auth = useAuthController();

// Composables
const { friends, pendingRequests, loadFriends, loadPendingRequests, sendFriendRequest } =
  useFriends();
const { createConversation } = useConversations();
const { connect, on: onWs, off: offWs } = useWebSocket();
const router = useRouter();

// State
const selectedFriend = ref<Friendship | null>(null);
const selectedUser = ref<User | null>(null);
const showProfileModal = ref(false);
const showFriendRequestHistory = ref(false);
const searchQuery = ref('');
const showSearchResults = ref(false);
const searchedUsers = ref<User[]>([]);
const selectedStranger = ref<User | null>(null);
const showStrangerModal = ref(false);
const allFriendRequests = ref<Friendship[]>([]);

// Computed
const displayUser = computed(() => {
  return selectedUser.value || auth.user;
});

const filteredFriends = computed(() => {
  if (!searchQuery.value) return [];
  const query = searchQuery.value.toLowerCase();
  return friends.value.filter((friendship) => {
    const friend = friendship.friend;
    if (!friend) return false;
    return friend.username.toLowerCase().includes(query) || friend.uid.toString().includes(query);
  });
});

// WebSocket handlers
const handleNewFriendRequest = async (data: any) => {
  console.log('[FriendsPanel] 收到新的好友请求', data);
  // 显示提醒
  alert(`收到来自 ${data.sender_id} 的好友请求`);
  // 重新加载待处理请求
  await loadPendingRequests();
  // 重新加载所有好友申请记录
  await loadAllFriendRequests();
};

const handleFriendRequestUpdate = async (data: any) => {
  console.log('[FriendsPanel] 收到好友请求更新', data);
  // 根据状态显示提醒
  if (data.status === 'accepted') {
    alert('好友请求已被接受');
  } else if (data.status === 'rejected') {
    alert('好友请求已被拒绝');
  }
  // 重新加载好友列表和待处理请求
  await loadFriends();
  await loadPendingRequests();
  // 重新加载所有好友申请记录
  await loadAllFriendRequests();
};

// 加载所有好友申请记录
const loadAllFriendRequests = async () => {
  console.log('[FriendsPanel] loadAllFriendRequests 开始');
  try {
    const response = await api.getAllFriendRequests();
    if (response.success && response.data) {
      allFriendRequests.value = response.data;
      console.log('[FriendsPanel] 所有好友申请记录加载成功', allFriendRequests.value.length, '条');
    } else {
      console.log('[FriendsPanel] 所有好友申请记录加载失败', response.message);
    }
  } catch (error) {
    console.error('[FriendsPanel] Failed to load all friend requests:', error);
  }
};

// Handlers
const handleShowUserProfile = (user: User) => {
  console.log('[FriendsPanel] handleShowUserProfile', { user });
  selectedUser.value = user;
  showProfileModal.value = true;
};

const handleSelectFriend = (friendship: Friendship) => {
  console.log('[FriendsPanel] handleSelectFriend', { friendship });
  selectedFriend.value = friendship;
  showFriendRequestHistory.value = false; // 关闭好友申请记录页面
  showSearchResults.value = false;
  searchQuery.value = '';
};

const handleSelectUser = (user: User) => {
  console.log('[FriendsPanel] handleSelectUser', { user });
  selectedStranger.value = user;
  showStrangerModal.value = true;
  showSearchResults.value = false;
  searchQuery.value = '';
};

const handleStartChatWithFriend = async () => {
  if (!selectedFriend.value?.friend?.id) return;

  const conversation = await createConversation(selectedFriend.value.friend.id);
  if (conversation) {
    // 跳转到聊天面板
    router.push('/chat');
  }
};

const handleSearch = async () => {
  console.log('[FriendsPanel] handleSearch', { query: searchQuery.value });
  if (!searchQuery.value.trim()) {
    searchedUsers.value = [];
    return;
  }

  try {
    const response = await api.searchUsers(searchQuery.value);
    if (response.success && response.data) {
      // 过滤掉已经是好友的用户
      const friendIds = new Set(friends.value.map((f) => f.friend?.id));
      searchedUsers.value = response.data.filter((user) => !friendIds.has(user.id));
      console.log('[FriendsPanel] 搜索到用户', searchedUsers.value.length, '个');
    }
  } catch (error) {
    console.error('[FriendsPanel] 搜索用户失败', error);
  }
};

const clearSearch = () => {
  console.log('[FriendsPanel] clearSearch');
  searchQuery.value = '';
  searchedUsers.value = [];
  showSearchResults.value = false;
};

const handleSendFriendRequest = async () => {
  if (!selectedStranger.value?.id) return;

  console.log('[FriendsPanel] handleSendFriendRequest', { userId: selectedStranger.value.id });
  const success = await sendFriendRequest(selectedStranger.value.id);
  if (success) {
    showStrangerModal.value = false;
    selectedStranger.value = null;
  }
};

// 辅助函数：获取好友请求文本
const getFriendRequestText = (request: Friendship): string => {
  if (request.status === 'pending') {
    // 判断是发送方还是接收方
    if (request.user_id === auth.user?.id) {
      return '已发送好友申请';
    } else {
      return '申请添加你为好友';
    }
  } else if (request.status === 'accepted') {
    return '已接受好友申请';
  } else if (request.status === 'rejected') {
    return '已拒绝好友申请';
  }
  return '';
};

// 辅助函数：格式化时间
const formatTime = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) {
    return '刚刚';
  } else if (minutes < 60) {
    return `${minutes}分钟前`;
  } else if (hours < 24) {
    return `${hours}小时前`;
  } else if (days < 7) {
    return `${days}天前`;
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' });
  }
};

// 辅助函数：获取好友请求状态样式
const getFriendRequestStatusClass = (status: string): string => {
  switch (status) {
    case 'pending':
      return 'bg-yellow-500 text-white';
    case 'accepted':
      return 'bg-green-500 text-white';
    case 'rejected':
      return 'bg-red-500 text-white';
    default:
      return 'bg-gray-500 text-white';
  }
};

// 辅助函数：获取好友请求状态文本
const getFriendRequestStatusText = (status: string): string => {
  switch (status) {
    case 'pending':
      return '待处理';
    case 'accepted':
      return '已接受';
    case 'rejected':
      return '已拒绝';
    default:
      return '未知';
  }
};

// Watchers
watch(
  () => auth.user,
  async (newValue) => {
    if (newValue) {
      console.log('[FriendsPanel] currentUser changed, 加载数据');
      await loadFriends();
      await loadPendingRequests();
      await loadAllFriendRequests();
    }
  }
);

// Lifecycle
onMounted(async () => {
  console.log('[FriendsPanel] onMounted 开始');
  await auth.checkAuth();
  const user = auth.user;
  console.log('[FriendsPanel] checkAuth 完成', { currentUser: user });
  if (user) {
    console.log('[FriendsPanel] currentUser 存在，开始加载数据');
    await loadFriends();
    await loadPendingRequests();
    await loadAllFriendRequests();
  } else {
    console.log('[FriendsPanel] currentUser 不存在，不加载数据');
  }

  // 连接WebSocket
  console.log('[FriendsPanel] 连接WebSocket');
  connect();

  // 注册WebSocket事件处理器
  onWs('new_friend_request', handleNewFriendRequest);
  onWs('friend_request_update', handleFriendRequestUpdate);

  console.log('[FriendsPanel] onMounted 结束');
});

onUnmounted(() => {
  console.log('[FriendsPanel] onUnmounted');
  // 移除WebSocket事件处理器
  offWs('new_friend_request', handleNewFriendRequest);
  offWs('friend_request_update', handleFriendRequestUpdate);
});
</script>

<style scoped>
/* 点击外部关闭搜索结果 */
:deep(.fixed) {
  z-index: 50;
}
</style>
