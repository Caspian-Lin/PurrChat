<template>
  <div class="flex h-full">
    <!-- 好友列表 -->
    <ResizableContainer
      direction="horizontal"
      :initial-size="320"
      :min-size="250"
      :max-size="500"
      class="bg-bg-primary border-r border-border-subtle"
    >
      <div class="flex flex-col h-full relative">
        <!-- 搜索好友 -->
        <div
          class="flex items-center gap-2 px-4 py-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0 relative"
        >
          <div
            class="flex-1 flex items-center bg-bg-quaternary rounded-[var(--radius-sm)] h-10 px-3"
          >
            <input
              v-model="searchQuery"
              type="text"
              placeholder="搜索好友或用户..."
              class="w-full bg-transparent text-text-primary placeholder-text-tertiary text-sm outline-none"
              @input="handleSearch"
              @focus="showSearchResults = true"
            />
          </div>
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
          class="flex items-center gap-4 px-4 py-3 bg-bg-secondary border-b border-border-subtle cursor-pointer hover:bg-hover-bg transition-colors flex-shrink-0"
          @click="showFriendRequestHistory = true"
        >
          <div class="relative">
            <div
              class="w-11 h-11 rounded-[var(--radius-md)] flex items-center justify-center text-white font-bold"
              style="background: var(--theme-secondary)"
            >
              🔔
            </div>
            <div
              class="absolute -top-1 -right-1 w-5 h-5 bg-[var(--theme-primary)] rounded-full flex items-center justify-center text-primary text-xs font-bold"
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
          v-if="
            showSearchResults &&
            (filteredFriends.length > 0 || searchedUsers.length > 0 || filteredGroups.length > 0)
          "
          class="absolute top-[80px] left-0 right-0 bg-bg-primary border border-border-subtle rounded-[var(--radius-lg)] shadow-lg z-50 max-h-[400px] overflow-y-auto scrollable"
          style="width: 300px"
        >
          <!-- 好友列表 -->
          <div v-if="filteredFriends.length > 0" class="border-b border-border-subtle">
            <div class="px-3 py-2 text-sm font-semibold text-text-secondary bg-bg-secondary">
              好友
            </div>
            <div class="px-1">
              <BaseListItem
                v-for="friendship in filteredFriends"
                :key="'friend-' + friendship.id"
                @click="handleSelectFriend(friendship)"
              >
                <template #avatar>
                  <div class="w-9 h-9 rounded-[var(--radius-md)] overflow-hidden flex-shrink-0">
                    <img
                      v-if="friendship.friend?.avatar_url"
                      :src="friendship.friend.avatar_url"
                      alt="avatar"
                      class="w-full h-full object-cover"
                    />
                    <div
                      v-else
                      class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
                      style="background: var(--theme-gradient)"
                    >
                      {{ friendship.friend?.username?.charAt(0) || '?' }}
                    </div>
                  </div>
                </template>

                <div class="font-semibold truncate text-text-primary text-sm">
                  {{ friendship.friend?.username }}
                </div>
                <div class="text-xs text-text-secondary">UID: {{ friendship.friend?.uid }}</div>
              </BaseListItem>
            </div>
          </div>

          <!-- 群聊列表 -->
          <div v-if="filteredGroups.length > 0" class="border-b border-border-subtle">
            <div class="px-3 py-2 text-sm font-semibold text-text-secondary bg-bg-secondary">
              群聊
            </div>
            <div class="px-1">
              <BaseListItem
                v-for="conversation in filteredGroups"
                :key="'group-' + conversation.id"
                @click="handleSelectGroup(conversation)"
              >
                <template #avatar>
                  <div
                    class="w-9 h-9 rounded-[var(--radius-md)] overflow-hidden flex-shrink-0"
                    style="background: var(--theme-gradient)"
                  >
                    <div
                      class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
                    >
                      {{ conversation.name?.charAt(0) || 'G' }}
                    </div>
                  </div>
                </template>

                <div class="font-semibold truncate text-text-primary text-sm">
                  {{ conversation.name }}
                </div>
                <div class="text-xs text-text-secondary">群聊</div>
              </BaseListItem>
            </div>
          </div>

          <!-- 搜索到的用户列表 -->
          <div v-if="searchedUsers.length > 0">
            <div class="px-3 py-2 text-sm font-semibold text-text-secondary bg-bg-secondary">
              用户
            </div>
            <div class="px-1">
              <BaseListItem
                v-for="user in searchedUsers"
                :key="'user-' + user.id"
                @click="handleSelectUser(user)"
              >
                <template #avatar>
                  <div class="w-9 h-9 rounded-[var(--radius-md)] overflow-hidden flex-shrink-0">
                    <img
                      v-if="user.avatar_url"
                      :src="user.avatar_url"
                      alt="avatar"
                      class="w-full h-full object-cover"
                    />
                    <div
                      v-else
                      class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
                      style="background: var(--theme-gradient)"
                    >
                      {{ user.username?.charAt(0) || '?' }}
                    </div>
                  </div>
                </template>

                <div class="flex items-center gap-2">
                  <span class="font-semibold truncate text-text-primary text-sm">{{
                    user.username
                  }}</span>
                  <span
                    class="text-xs px-1.5 py-0.5 bg-orange-500 text-white rounded-[var(--radius-xs)]"
                    >陌生人</span
                  >
                </div>
                <div class="text-xs text-text-secondary">UID: {{ user.uid }}</div>
              </BaseListItem>
            </div>
          </div>

          <!-- 无结果 -->
          <div
            v-if="
              filteredFriends.length === 0 &&
              searchedUsers.length === 0 &&
              filteredGroups.length === 0
            "
            class="p-4 text-center text-text-tertiary"
          >
            未找到匹配的用户
          </div>
        </div>

        <!-- 好友列表 -->
        <div class="flex-1 min-h-0">
          <!-- 标签切换 -->
          <div class="flex gap-2 p-3 bg-bg-secondary border-b border-border-subtle">
            <button
              :class="[
                'flex-1 py-2 px-4 rounded-[var(--radius-sm)] font-medium transition-colors',
                activeTab === 'friends'
                  ? 'bg-accent-color text-white'
                  : 'bg-bg-quaternary text-text-secondary hover:text-text-primary',
              ]"
              @click="activeTab = 'friends'"
            >
              好友
            </button>
            <button
              :class="[
                'flex-1 py-2 px-4 rounded-[var(--radius-sm)] font-medium transition-colors',
                activeTab === 'groups'
                  ? 'bg-accent-color text-white'
                  : 'bg-bg-quaternary text-text-secondary hover:text-text-primary',
              ]"
              @click="activeTab = 'groups'"
            >
              群聊
            </button>
          </div>

          <!-- 好友列表 -->
          <FriendList
            v-if="activeTab === 'friends'"
            :friends="friends"
            @select="handleSelectFriend"
            @show-user="handleShowUserProfile"
          />

          <!-- 群聊列表 -->
          <CustomScrollbar v-else class="flex-1 min-h-0">
            <div class="px-2 pt-1 pb-0.5">
              <BaseListItem
                v-for="conversation in filteredGroups"
                :key="conversation.id"
                @click="handleSelectGroup(conversation)"
              >
                <template #avatar>
                  <div
                    class="w-11 h-11 rounded-[var(--radius-md)] overflow-hidden"
                    style="background: var(--theme-gradient)"
                  >
                    <div
                      class="w-full h-full flex items-center justify-center font-bold text-white text-lg"
                    >
                      {{ conversation.name?.charAt(0) || 'G' }}
                    </div>
                  </div>
                </template>

                <div class="flex items-center gap-2">
                  <span class="font-semibold text-[15px] truncate text-text-primary">
                    {{ conversation.name }}
                  </span>
                  <span class="text-xs px-1 rounded-[var(--radius-xs)] bg-bg-secondary">群聊</span>
                </div>
                <div class="text-sm text-text-secondary truncate">
                  {{ conversation.last_message?.content || '暂无消息' }}
                </div>
              </BaseListItem>
            </div>
            <div
              v-if="filteredGroups.length === 0"
              class="flex flex-col items-center justify-center h-full text-center p-8 text-text-tertiary"
            >
              <p>暂无群聊</p>
            </div>
          </CustomScrollbar>
        </div>
      </div>
    </ResizableContainer>

    <!-- 好友信息窗口 -->
    <div class="flex-1 flex flex-col bg-bg-tertiary">
      <!-- 好友申请历史 -->
      <CustomScrollbar v-if="showFriendRequestHistory" class="flex-1">
        <div class="flex flex-col p-6 h-full">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-2xl font-bold text-text-primary">好友申请记录</h2>
            <button
              class="bg-bg-quaternary hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
              @click="showFriendRequestHistory = false"
            >
              <BsX class="text-2xl" />
            </button>
          </div>

          <div
            v-if="allFriendRequests.length === 0"
            class="flex-1 flex items-center justify-center text-text-tertiary"
          >
            <p>暂无好友申请记录</p>
          </div>

          <div v-else class="space-y-1">
            <BaseListItem v-for="request in allFriendRequests" :key="request.id">
              <template #avatar>
                <div
                  class="w-11 h-11 rounded-[var(--radius-md)] overflow-hidden cursor-pointer"
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
              </template>

              <div class="flex items-center justify-between">
                <div class="min-w-0 flex-1">
                  <div class="font-semibold text-text-primary text-sm">
                    {{ request.user?.username }}
                  </div>
                  <div class="text-xs text-text-secondary">
                    {{ getFriendRequestText(request) }}
                  </div>
                  <div class="text-xs text-text-tertiary">
                    {{ formatTime(request.created_at) }}
                  </div>
                </div>
                <div
                  v-if="request.status === 'pending' && isRequestRecipient(request)"
                  class="flex gap-1.5 ml-2"
                >
                  <button
                    class="px-3 py-1 bg-green-500 text-white rounded-[var(--radius-sm)] text-xs font-medium hover:bg-green-600 transition-colors"
                    @click="handleAcceptRequest(request)"
                  >
                    接受
                  </button>
                  <button
                    class="px-3 py-1 bg-red-500 text-white rounded-[var(--radius-sm)] text-xs font-medium hover:bg-red-600 transition-colors"
                    @click="handleRejectRequest(request)"
                  >
                    忽略
                  </button>
                </div>
                <div
                  v-else
                  :class="[
                    'px-2.5 py-1 rounded-[var(--radius-sm)] text-xs font-medium',
                    getFriendRequestStatusClass(request.status),
                  ]"
                >
                  {{ getFriendRequestStatusText(request.status) }}
                </div>
              </div>
            </BaseListItem>
          </div>
        </div>
      </CustomScrollbar>

      <FriendInfoModal
        v-else-if="selectedFriend"
        :friendship="selectedFriend"
        @close="selectedFriend = null"
        @start-chat="handleStartChatWithFriend"
      />

      <!-- 空状态 -->
      <div v-else class="flex-1 flex flex-col items-center justify-center text-text-tertiary">
        <div
          class="w-20 h-20 rounded-full flex items-center justify-center mb-6"
          style="background: var(--surface-color)"
        >
          <svg
            class="w-10 h-10"
            style="color: var(--text-tertiary-color)"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
            />
          </svg>
        </div>
        <h3 class="text-lg font-semibold mb-1 text-text-primary">好友列表</h3>
        <p class="text-sm">选择一个好友查看详情或开始聊天</p>
      </div>
    </div>

    <!-- 个人资料弹窗 -->
    <UserProfileModal
      v-model:show="showProfileModal"
      :user="displayUser"
      :is-current-user="!selectedUser || selectedUser.id === auth.currentUser?.id"
      :friendship="getUserFriendship"
      :loading="isSendingRequest"
      :current-user-id="auth.currentUser?.id"
      @send-friend-request="handleSendFriendRequest"
      @accept-request="handleAcceptRequestFromModal"
      @reject-request="handleRejectRequestFromModal"
      @start-chat="handleStartChatFromModal"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { useAuthController } from '../../../controllers/authController';
import { useFriends } from '../../../composables/useFriends';
import { useConversations } from '../../../composables/useConversations';
import { useWebSocketEventManager } from '../../../services/websocketEventManager';
import { useConversationStateCache } from '../../../services/conversationStateCache';
import { api } from '../../../models/api';
import { useRouter } from 'vue-router';
import FriendList from '../FriendList.vue';
import FriendInfoModal from '../FriendInfoModal.vue';
import UserProfileModal from '../UserProfileModal.vue';
import ResizableContainer from '../../common/ResizableContainer.vue';
import CustomScrollbar from '../../common/CustomScrollbar.vue';
import BaseListItem from '../../common/BaseListItem.vue';
import type { User, Friendship, Conversation } from '../../../models/types';
import { BsX } from 'vue-icons-plus/bs';

// Auth
const auth = useAuthController();

// Composables
const {
  friends,
  pendingRequests,
  loadFriends,
  loadPendingRequests,
  sendFriendRequest,
  handleFriendRequest,
} = useFriends();
const { conversations, loadConversations, createConversation } = useConversations();
const { onFriendRequest, offFriendRequest } = useWebSocketEventManager();
const { showConversation } = useConversationStateCache();
const router = useRouter();

// State
const selectedFriend = ref<Friendship | null>(null);
const selectedUser = ref<User | null>(null);
const showProfileModal = ref(false);
const showFriendRequestHistory = ref(false);
const searchQuery = ref('');
const showSearchResults = ref(false);
const searchedUsers = ref<User[]>([]);
const allFriendRequests = ref<Friendship[]>([]);
const isSendingRequest = ref(false);
const activeTab = ref<'friends' | 'groups'>('friends'); // 标签切换状态

// Computed
const displayUser = computed(() => {
  return selectedUser.value || auth.currentUser;
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

// 群聊列表（按名称排序）
const groupConversations = computed(() => {
  return conversations.value
    .filter((conv) => conv.conversation_type === 'group')
    .sort((a, b) => {
      const nameA = a.name || '';
      const nameB = b.name || '';
      return nameA.localeCompare(nameB, 'zh-CN');
    });
});

// 过滤后的群聊列表
const filteredGroups = computed(() => {
  if (!searchQuery.value) return groupConversations.value;
  const query = searchQuery.value.toLowerCase();
  return groupConversations.value.filter((conv) => {
    return conv.name?.toLowerCase().includes(query);
  });
});

// 获取用户的好友关系
const getUserFriendship = computed(() => {
  if (!selectedUser.value || !auth.currentUser?.id) return null;

  // 检查是否是当前用户自己
  if (selectedUser.value.id === auth.currentUser.id) return null;

  // 检查是否已经是好友
  const friendship = friends.value.find(
    (f) => f.friend?.id === selectedUser.value?.id || f.user?.id === selectedUser.value?.id
  );
  if (friendship) return friendship;

  // 检查是否有待处理的好友申请
  const pendingRequest = allFriendRequests.value.find(
    (r) => r.user?.id === selectedUser.value?.id || r.friend?.id === selectedUser.value?.id
  );
  if (pendingRequest) return pendingRequest;

  return null;
});

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

// WebSocket事件处理器
const handleFriendRequestUpdate = async (friendship: Friendship) => {
  console.log('[FriendsPanel] 收到好友请求更新事件:', friendship);

  // 重新加载相关数据
  await loadFriends();
  await loadPendingRequests();
  await loadAllFriendRequests();
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
  selectedUser.value = user;
  showProfileModal.value = true;
  showSearchResults.value = false;
  searchQuery.value = '';
};

const handleSelectGroup = async (conversation: Conversation) => {
  console.log('[FriendsPanel] handleSelectGroup', { conversation });
  // 显示会话（如果被隐藏）
  showConversation(conversation.id);
  // 跳转到聊天面板并选中对应会话
  router.push({ path: '/chat', query: { conversationId: conversation.id } });
};

const handleStartChatWithFriend = async () => {
  if (!selectedFriend.value?.friend?.id) return;

  const conversation = await createConversation(selectedFriend.value.friend.id);
  if (conversation) {
    // 显示会话（如果被隐藏）
    showConversation(conversation.id);
    // 跳转到聊天面板并选中对应会话
    router.push({ path: '/chat', query: { conversationId: conversation.id } });
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
  if (!selectedUser.value?.id) return;

  console.log('[FriendsPanel] handleSendFriendRequest', { userId: selectedUser.value.id });
  isSendingRequest.value = true;
  const success = await sendFriendRequest(selectedUser.value.id);
  isSendingRequest.value = false;
  if (success) {
    showProfileModal.value = false;
    selectedUser.value = null;
    // 重新加载好友申请记录
    await loadAllFriendRequests();
  }
};

// 处理接受好友请求（从 UserProfileModal 触发）
const handleAcceptRequestFromModal = async () => {
  if (!getUserFriendship.value?.conversation_id) return;

  console.log('[FriendsPanel] handleAcceptRequestFromModal', {
    conversationId: getUserFriendship.value.conversation_id,
  });

  const success = await handleFriendRequest(getUserFriendship.value.conversation_id, 'accept');
  if (success) {
    showProfileModal.value = false;
    selectedUser.value = null;
    // 重新加载数据
    await loadFriends();
    await loadPendingRequests();
    await loadAllFriendRequests();
  }
};

// 处理拒绝好友请求（从 UserProfileModal 触发）
const handleRejectRequestFromModal = async () => {
  if (!getUserFriendship.value?.conversation_id) return;

  console.log('[FriendsPanel] handleRejectRequestFromModal', {
    conversationId: getUserFriendship.value.conversation_id,
  });

  const success = await handleFriendRequest(getUserFriendship.value.conversation_id, 'reject');
  if (success) {
    showProfileModal.value = false;
    selectedUser.value = null;
    // 重新加载数据
    await loadPendingRequests();
    await loadAllFriendRequests();
  }
};

// 处理开始聊天（从 UserProfileModal 触发）
const handleStartChatFromModal = async () => {
  if (!selectedUser.value?.id) return;

  const conversation = await createConversation(selectedUser.value.id);
  if (conversation) {
    // 显示会话（如果被隐藏）
    showConversation(conversation.id);
    showProfileModal.value = false;
    selectedUser.value = null;
    // 跳转到聊天面板并选中对应会话
    router.push({ path: '/chat', query: { conversationId: conversation.id } });
  }
};

// 判断当前用户是否是请求的接收方
const isRequestRecipient = (request: Friendship): boolean => {
  // 在后端 SendFriendRequest 中，UserID 是发送者，FriendID 是接收者
  // 所以接收方应该检查 friendship.FriendID == auth.currentUser?.id
  return request.friend_id === auth.currentUser?.id;
};

// 处理接受好友请求
const handleAcceptRequest = async (request: Friendship) => {
  console.log('[FriendsPanel] handleAcceptRequest', {
    requestId: request.id,
    conversationId: request.conversation_id,
  });

  const success = await handleFriendRequest(request.conversation_id, 'accept');
  if (success) {
    // 重新加载数据
    await loadFriends();
    await loadPendingRequests();
    await loadAllFriendRequests();
  }
};

// 处理忽略好友请求
const handleRejectRequest = async (request: Friendship) => {
  console.log('[FriendsPanel] handleRejectRequest', {
    requestId: request.id,
    conversationId: request.conversation_id,
  });

  const success = await handleFriendRequest(request.conversation_id, 'reject');
  if (success) {
    // 重新加载数据
    await loadPendingRequests();
    await loadAllFriendRequests();
  }
};

// 辅助函数：获取好友请求文本
const getFriendRequestText = (request: Friendship): string => {
  if (request.status === 'pending') {
    // 判断是发送方还是接收方
    if (request.user_id === auth.currentUser?.id) {
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
  () => auth.currentUser,
  async (newValue) => {
    if (newValue) {
      console.log('[FriendsPanel] currentUser changed, 加载数据');
      await loadFriends();
      await loadPendingRequests();
      await loadAllFriendRequests();
      await loadConversations();
    }
  }
);

// Lifecycle
onMounted(async () => {
  console.log('[FriendsPanel] onMounted 开始');
  await auth.checkAuth();
  const user = auth.currentUser;
  console.log('[FriendsPanel] checkAuth 完成', { currentUser: user });
  if (user) {
    console.log('[FriendsPanel] currentUser 存在，开始加载数据');
    await loadFriends();
    await loadPendingRequests();
    await loadAllFriendRequests();
    await loadConversations();

    // 注册WebSocket事件回调
    onFriendRequest(handleFriendRequestUpdate);
  } else {
    console.log('[FriendsPanel] currentUser 不存在，不加载数据');
  }
  console.log('[FriendsPanel] onMounted 结束');
});

onUnmounted(() => {
  console.log('[FriendsPanel] onUnmounted，清理 WebSocket 事件');
  // 清理WebSocket事件回调
  offFriendRequest(handleFriendRequestUpdate);
});
</script>

<style scoped>
/* 点击外部关闭搜索结果 */
:deep(.fixed) {
  z-index: 50;
}
</style>
