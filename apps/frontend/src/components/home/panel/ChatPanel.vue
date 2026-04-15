<template>
  <div class="flex h-full">
    <!-- 会话列表 -->
    <ResizableContainer
      direction="horizontal"
      :initial-size="320"
      :min-size="250"
      :max-size="500"
      class="bg-bg-primary border-r border-border-color"
    >
      <div class="flex flex-col h-full">
        <!-- 搜索用户 -->
        <div
          class="flex items-center gap-2 px-3 pt-5 pb-3 bg-bg-secondary border-b border-border-color flex-shrink-0"
        >
          <div class="flex-1 flex items-center bg-bg-quaternary rounded-md h-[40px] px-3">
            <input
              v-model="searchQuery"
              type="text"
              placeholder="搜索会话、消息内容、好友名..."
              class="w-full bg-transparent text-text-tertiary text-base font-normal outline-none"
              @input="handleSearch"
            />
          </div>
          <button
            v-if="searchQuery"
            class="relative p-2 flex items-center justify-center transition-all text-text-primary hover:text-text-primary"
            @click="clearSearch"
          >
            <BsXCircle />
          </button>
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-primary hover:text-text-primary"
            title="创建群聊"
            @click="handleCreateGroup"
          >
            <BsPlusLg />
          </button>
        </div>

        <!-- 会话列表 -->
        <div class="flex-1 min-h-0">
          <ConversationList
            :conversations="filteredConversations"
            :selected-id="selectedConversation?.id"
            :current-user-id="auth.currentUser?.id"
            @select="handleSelectConversation"
            @show-user="handleShowUserProfile"
            @delete-conversation="handleDeleteConversation"
          />
        </div>
      </div>
    </ResizableContainer>

    <!-- 聊天窗口 -->
    <div class="flex-1 flex flex-col bg-bg-tertiary">
      <ChatWindow
        ref="chatWindowRef"
        v-if="selectedConversation"
        :conversation="selectedConversation"
        :messages="messages"
        :current-user-id="auth.currentUser?.id"
        @send-message="handleSendMessage"
        @send-file-message="handleSendFileMessage"
        @export-messages="handleExportMessages"
        @show-user="handleShowUserProfile"
        @update-conversation="handleUpdateConversation"
        @create-group="handleCreateGroup"
        @show-detail="handleShowDetail"
      />

      <!-- 空状态 -->
      <div v-else class="flex-1 flex flex-col items-center justify-center text-text-tertiary">
        <div class="text-6xl mb-4">💬</div>
        <h3 class="text-2xl font-semibold mb-2 text-text-primary">欢迎来到 PurrChat</h3>
        <p>选择一个会话开始聊天</p>
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
      @send-friend-request="handleSendFriendRequestFromModal"
      @accept-request="handleAcceptRequestFromModal"
      @reject-request="handleRejectRequestFromModal"
      @start-chat="handleStartChatFromModal"
    />

    <!-- 搜索用户操作弹窗 -->
    <UserActionsModal
      v-model:show="showSearchModal"
      :user="selectedSearchUser"
      @send-friend-request="handleSendFriendRequest"
      @start-chat="handleStartChatWithSearchUser"
    />

    <!-- 创建群聊弹窗 -->
    <CreateGroupModal
      v-model:show="showCreateGroupModal"
      :friends="friends"
      @group-created="handleGroupCreated"
    />

    <!-- 会话详情弹窗 -->
    <ConversationDetailModal
      v-model:show="showConversationDetailModal"
      :conversation="selectedConversation"
      :current-user-id="auth.currentUser?.id"
      @show-user-profile="handleShowUserProfile"
      @members-changed="handleGroupUpdated"
      @start-chat="handleStartChatFromDetail"
    />

    <!-- 通知列表 -->
    <NotificationList :notifications="notifications" @remove-notification="removeNotification" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';
import { useAuthController } from '../../../controllers/authController';
import { useConversations } from '../../../composables/useConversations';
import { useFriends } from '../../../composables/useFriends';
import { useChat } from '../../../composables/useChat';
import { useMessageCache } from '../../../services/messageCache';
import { useMessageStore } from '../../../stores/message';
import { useNotification } from '../../../composables/useNotification';
import { useWebSocketEventManager } from '../../../services/websocketEventManager';
import { useConversationStateCache } from '../../../services/conversationStateCache';
import { useRoute, useRouter } from 'vue-router';
import { api } from '../../../models/api';
import ConversationList from '../ConversationList.vue';
import ChatWindow from '../ChatWindow.vue';
import UserProfileModal from '../UserProfileModal.vue';
import UserActionsModal from '../UserActionsModal.vue';
import CreateGroupModal from '../CreateGroupModal.vue';
import ConversationDetailModal from '../ConversationDetailModal.vue';
import NotificationList from '../../common/NotificationList.vue';
import ResizableContainer from '../../common/ResizableContainer.vue';
import type {
  User,
  Conversation,
  Message,
  Friendship,
  FileMessageContent,
} from '../../../models/types';
import { BsPlusLg, BsXCircle } from 'vue-icons-plus/bs';

// Auth
const auth = useAuthController();
const route = useRoute();
const router = useRouter();

// Composables
const { conversations, loadConversations, createConversation } = useConversations();
const { friends, loadFriends, sendFriendRequest, handleFriendRequest, loadPendingRequests } =
  useFriends();
const {
  loadMessages,
  checkAndLoadIncremental,
  sendMessage,
  sendFileMessage,
  exportMessages,
  clearMessages,
} = useChat();
const { addMessage: cacheMessage } = useMessageCache();
const { notifications, removeNotification } = useNotification();
const messageStore = useMessageStore();
const {
  setCurrentConversation,
  onMessageUpdate,
  offMessageUpdate,
  onConversationUpdate,
  offConversationUpdate,
} = useWebSocketEventManager();
const {
  isHidden,
  hideConversation,
  showConversation,
  getUnreadCount,
  incrementUnreadCount,
  clearUnreadCount,
} = useConversationStateCache();

// ChatWindow组件引用
const chatWindowRef = ref<InstanceType<typeof ChatWindow> | null>(null);

// 从messageStore获取当前会话的消息
const messages = computed(() => {
  if (!selectedConversation.value?.id) {
    return [];
  }
  return messageStore.getMessages(selectedConversation.value.id);
});

// 搜索状态
const searchQuery = ref('');

// 过滤后的会话列表（排除隐藏的会话，并添加未读计数）
const filteredConversations = computed(() => {
  // 使用conversations.value的所有会话，包括隐藏的
  const allConversations = conversations.value;

  // 过滤掉隐藏的会话
  const visibleConversations = allConversations.filter((conv) => !isHidden(conv.id));

  if (!searchQuery.value.trim()) {
    return visibleConversations.map((conv) => ({
      ...conv,
      unread_count: getUnreadCount(conv.id),
    }));
  }

  const query = searchQuery.value.toLowerCase();
  return visibleConversations
    .filter((conv) => {
      // 搜索会话名称
      if (conv.name && conv.name.toLowerCase().includes(query)) {
        return true;
      }

      // 搜索好友名称
      if (conv.members) {
        for (const member of conv.members) {
          if (member.user && member.user.username.toLowerCase().includes(query)) {
            return true;
          }
        }
      }

      // 搜索最后一条消息内容
      if (conv.last_message && conv.last_message.content.toLowerCase().includes(query)) {
        return true;
      }

      return false;
    })
    .map((conv) => ({
      ...conv,
      unread_count: getUnreadCount(conv.id),
    }));
});

// 搜索处理
const handleSearch = () => {
  console.log('[ChatPanel] Search query:', searchQuery.value);
};

// 清除搜索
const clearSearch = () => {
  searchQuery.value = '';
  console.log('[ChatPanel] Search cleared');
};

// State
const selectedConversation = ref<Conversation | null>(null);
const selectedUser = ref<User | null>(null);
const showProfileModal = ref(false);
const showSearchModal = ref(false);
const selectedSearchUser = ref<User | null>(null);
const showCreateGroupModal = ref(false);
const showConversationDetailModal = ref(false);
const isSendingRequest = ref(false);
const allFriendRequests = ref<Friendship[]>([]);

// Computed
const displayUser = computed(() => {
  return selectedUser.value || auth.currentUser;
});

// 获取用户的好友关系（用于 UserProfileModal）
const getUserFriendship = computed(() => {
  if (!selectedUser.value || !auth.currentUser?.id) return null;
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
  try {
    const response = await api.getAllFriendRequests();
    if (response.success && response.data) {
      allFriendRequests.value = response.data;
    }
  } catch (error) {
    console.error('[ChatPanel] Failed to load all friend requests:', error);
  }
};

// Handlers
const handleShowUserProfile = (user: User) => {
  selectedUser.value = user;
  showProfileModal.value = true;
};

// 处理从 UserProfileModal 发送好友请求
const handleSendFriendRequestFromModal = async () => {
  if (!selectedUser.value?.id) return;
  isSendingRequest.value = true;
  const success = await sendFriendRequest(selectedUser.value.id);
  isSendingRequest.value = false;
  if (success) {
    showProfileModal.value = false;
    selectedUser.value = null;
    await loadAllFriendRequests();
  }
};

// 处理从 UserProfileModal 接受好友请求
const handleAcceptRequestFromModal = async () => {
  if (!getUserFriendship.value?.conversation_id) return;
  const success = await handleFriendRequest(getUserFriendship.value.conversation_id, 'accept');
  if (success) {
    showProfileModal.value = false;
    selectedUser.value = null;
    await loadFriends();
    await loadPendingRequests();
    await loadAllFriendRequests();
  }
};

// 处理从 UserProfileModal 拒绝好友请求
const handleRejectRequestFromModal = async () => {
  if (!getUserFriendship.value?.conversation_id) return;
  const success = await handleFriendRequest(getUserFriendship.value.conversation_id, 'reject');
  if (success) {
    showProfileModal.value = false;
    selectedUser.value = null;
    await loadPendingRequests();
    await loadAllFriendRequests();
  }
};

// 处理从 UserProfileModal 开始聊天
const handleStartChatFromModal = async () => {
  if (!selectedUser.value?.id) return;
  const conversation = await createConversation(selectedUser.value.id);
  if (conversation) {
    if (isHidden(conversation.id)) {
      showConversation(conversation.id);
    }
    showProfileModal.value = false;
    selectedUser.value = null;
    handleSelectConversation(conversation);
  }
};

const handleSelectConversation = async (conversation: Conversation) => {
  selectedConversation.value = conversation;
  selectedUser.value = null;
  // 设置当前会话ID到WebSocket事件管理器
  setCurrentConversation(conversation.id);
  // 清除未读计数
  clearUnreadCount(conversation.id);
  // 使用增量加载检查并获取新消息
  await checkAndLoadIncremental(conversation.id);
};

// WebSocket事件处理器
const handleMessageUpdate = async (conversationId: string, message: Message) => {
  console.log('[ChatPanel] ===== 收到消息更新事件 =====');
  console.log('[ChatPanel] 消息会话ID:', conversationId);
  console.log('[ChatPanel] 当前选中会话ID:', selectedConversation.value?.id);
  console.log('[ChatPanel] 消息内容:', message.content);
  console.log('[ChatPanel] 消息ID:', message.id);
  console.log('[ChatPanel] 发送者ID:', message.sender_id);

  // 如果会话被隐藏，显示它
  if (isHidden(conversationId)) {
    console.log('[ChatPanel] 会话被隐藏，现在显示它');
    showConversation(conversationId);
    // 强制触发响应式更新
    conversations.value = [...conversations.value];
  }

  // 如果不是当前会话的消息，增加未读计数
  if (conversationId !== selectedConversation.value?.id) {
    console.log('[ChatPanel] 不是当前会话，增加未读计数');
    incrementUnreadCount(conversationId);
  }

  // 如果是当前会话的消息，自动滚动到底部
  if (conversationId === selectedConversation.value?.id) {
    console.log('[ChatPanel] 是当前会话，准备滚动到底部');
    await nextTick();
    // 调用ChatWindow的scrollToBottom方法
    if (chatWindowRef.value) {
      console.log('[ChatPanel] 调用scrollToBottom');
      chatWindowRef.value.scrollToBottom();
    } else {
      console.log('[ChatPanel] chatWindowRef.value 为空，无法滚动');
    }
  } else {
    console.log('[ChatPanel] 不是当前会话，不滚动');
  }
  console.log('[ChatPanel] ===== 消息更新事件处理完成 =====');
};

const handleConversationUpdate = async (conversation: Conversation) => {
  console.log('[ChatPanel] ===== 收到会话更新事件 =====');
  console.log('[ChatPanel] 会话ID:', conversation.id);
  console.log('[ChatPanel] 会话名称:', conversation.name);
  console.log('[ChatPanel] 最后消息:', conversation.last_message?.content);
  console.log('[ChatPanel] 准备重新加载会话列表');
  // 重新加载会话列表以获取最新数据
  await loadConversations();
  console.log('[ChatPanel] 会话列表重新加载完成');
  console.log('[ChatPanel] ===== 会话更新事件处理完成 =====');
};

const handleDeleteConversation = async (conversationId: string) => {
  // 前端隐藏会话，不删除后端数据
  hideConversation(conversationId);

  // 强制触发响应式更新
  conversations.value = [...conversations.value];

  // 如果删除的是当前选中的会话，清空选中状态
  if (selectedConversation.value?.id === conversationId) {
    selectedConversation.value = null;
    clearMessages();
  }
};

const handleSendFriendRequest = async () => {
  if (!selectedSearchUser.value?.id) return;

  const success = await sendFriendRequest(selectedSearchUser.value.id);
  if (success) {
    showSearchModal.value = false;
    selectedSearchUser.value = null;
    await loadConversations();
  }
};

const handleStartChatWithSearchUser = async () => {
  if (!selectedSearchUser.value?.id) return;

  const conversation = await createConversation(selectedSearchUser.value.id);
  if (conversation) {
    showSearchModal.value = false;
    selectedSearchUser.value = null;
    handleSelectConversation(conversation);
  }
};

const handleSendMessage = async (content: string) => {
  console.log('[ChatPanel] handleSendMessage called with content:', content);
  if (!selectedConversation.value?.id) {
    console.log('[ChatPanel] No selected conversation, returning');
    return;
  }

  // 如果会话被隐藏，显示它
  if (isHidden(selectedConversation.value.id)) {
    console.log('[ChatPanel] 会话被隐藏，现在显示它');
    showConversation(selectedConversation.value.id);
    // 强制触发响应式更新
    conversations.value = [...conversations.value];
  }

  console.log(
    '[ChatPanel] Calling sendMessage with conversationId:',
    selectedConversation.value.id
  );
  try {
    const success = await sendMessage(selectedConversation.value.id, content);
    console.log('[ChatPanel] sendMessage returned, success:', success);
    console.log('[ChatPanel] messages.value.length after sendMessage:', messages.value.length);
    if (success) {
      // 更新会话列表中的最后一条消息和更新时间
      const conversationIndex = conversations.value.findIndex(
        (c) => c.id === selectedConversation.value?.id
      );
      console.log('[ChatPanel] conversationIndex:', conversationIndex);
      console.log('[ChatPanel] messages.value.length:', messages.value.length);
      if (conversationIndex !== -1 && messages.value.length > 0) {
        const lastMessage = messages.value[messages.value.length - 1];
        console.log('[ChatPanel] lastMessage:', lastMessage);
        if (lastMessage) {
          const conversation = conversations.value[conversationIndex];
          if (conversation) {
            console.log('[ChatPanel] conversation before update:', {
              id: conversation.id,
              lastMessage: conversation.last_message?.content,
              updatedAt: conversation.updated_at,
            });
            // 使用展开运算符触发响应式更新
            conversations.value[conversationIndex] = {
              ...conversation,
              last_message: lastMessage,
              updated_at: new Date().toISOString(),
            };

            console.log('[ChatPanel] Updated conversation after send message:', {
              conversationId: selectedConversation.value.id,
              lastMessage: lastMessage.content,
              lastMessageCreatedAt: lastMessage.created_at,
              updatedAt: conversations.value[conversationIndex].updated_at,
            });

            // 缓存新消息
            await cacheMessage(selectedConversation.value.id, lastMessage);

            // 强制触发响应式更新
            console.log('[ChatPanel] Forcing reactive update by reassigning conversations.value');
            conversations.value = [...conversations.value];
            console.log(
              '[ChatPanel] conversations.value after reassign:',
              conversations.value.length
            );
          }
        }
      }
    }
  } catch (error) {
    console.error('[ChatPanel] Error in handleSendMessage:', error);
  }
};

const handleSendFileMessage = async (fileData: FileMessageContent) => {
  if (!selectedConversation.value?.id) return;

  if (isHidden(selectedConversation.value.id)) {
    showConversation(selectedConversation.value.id);
    conversations.value = [...conversations.value];
  }

  try {
    const success = await sendFileMessage(selectedConversation.value.id, fileData);
    if (success) {
      const conversationIndex = conversations.value.findIndex(
        (c) => c.id === selectedConversation.value?.id
      );
      if (conversationIndex !== -1 && messages.value.length > 0) {
        const lastMessage = messages.value[messages.value.length - 1];
        if (lastMessage) {
          const conversation = conversations.value[conversationIndex];
          if (conversation) {
            conversations.value[conversationIndex] = {
              ...conversation,
              last_message: lastMessage,
              updated_at: new Date().toISOString(),
            };
            await cacheMessage(selectedConversation.value.id, lastMessage);
            conversations.value = [...conversations.value];
          }
        }
      }
    }
  } catch (error) {
    console.error('[ChatPanel] Error in handleSendFileMessage:', error);
  }
};

const handleExportMessages = () => {
  if (!selectedConversation.value?.id) return;
  exportMessages(selectedConversation.value.id);
};

const handleUpdateConversation = async () => {
  // 更新会话列表和好友列表
  await loadConversations();
  await loadFriends();
};

const handleCreateGroup = () => {
  showCreateGroupModal.value = true;
};

const handleShowDetail = () => {
  if (selectedConversation.value) {
    showConversationDetailModal.value = true;
  }
};

const handleGroupCreated = async (conversationId: string) => {
  console.log('[ChatPanel] handleGroupCreated called with conversationId:', conversationId);
  showCreateGroupModal.value = false;
  // 重新加载会话列表
  await loadConversations();
  console.log('[ChatPanel] Conversations after reload:', conversations.value);
  // 选中新创建的群聊
  const newConversation = conversations.value.find((c) => c.id === conversationId);
  if (newConversation) {
    console.log('[ChatPanel] Selecting new group conversation:', newConversation);
    handleSelectConversation(newConversation);
  } else {
    console.log('[ChatPanel] New conversation not found in list:', conversationId);
  }
};

const handleGroupUpdated = async () => {
  showConversationDetailModal.value = false;
  // 重新加载会话列表和当前会话信息
  await loadConversations();
  if (selectedConversation.value) {
    const updatedConversation = conversations.value.find(
      (c) => c.id === selectedConversation.value?.id
    );
    if (updatedConversation) {
      selectedConversation.value = updatedConversation;
    }
  }
};

const handleStartChatFromDetail = (conversation: Conversation) => {
  console.log('[ChatPanel] handleStartChatFromDetail', { conversation });
  // 选中会话
  selectedConversation.value = conversation;
  // 清除未读计数
  clearUnreadCount(conversation.id);
  // 使用增量加载检查并获取新消息
  checkAndLoadIncremental(conversation.id);
};

// Watchers
watch(
  () => auth.currentUser,
  async () => {
    if (auth.currentUser) {
      await loadConversations();
      await loadFriends();
    }
  }
);

// 监听路由参数，如果有conversationId参数，选中对应的会话
watch(
  () => route.query.conversationId,
  async (conversationId) => {
    if (conversationId && typeof conversationId === 'string') {
      console.log('[ChatPanel] 路由参数中的会话ID:', conversationId);
      // 等待会话列表加载完成
      await loadConversations();
      // 查找对应的会话
      const conversation = conversations.value.find((c) => c.id === conversationId);
      if (conversation) {
        console.log('[ChatPanel] 找到会话，选中它:', conversation);
        // 显示会话（如果被隐藏）
        showConversation(conversation.id);
        // 强制触发响应式更新
        conversations.value = [...conversations.value];
        // 选中会话
        selectedConversation.value = conversation;
        // 清除路由参数
        router.replace({ path: '/chat', query: {} });
      } else {
        console.log('[ChatPanel] 未找到会话:', conversationId);
      }
    }
  }
);

watch(selectedConversation, async (newConv, oldConv) => {
  if (newConv && newConv.id !== oldConv?.id) {
    console.log('[ChatPanel] selectedConversation changed, loading messages for', newConv.id);
    // 先清空消息列表，避免消息残留
    clearMessages();
    // 然后加载新会话的消息
    await loadMessages(newConv.id);
    console.log('[ChatPanel] Messages loaded for conversation', newConv.id);
  }
});

// Lifecycle
onMounted(async () => {
  console.log('[ChatPanel] onMounted 开始');
  await auth.checkAuth();
  console.log('[ChatPanel] checkAuth 完成', { currentUser: auth.currentUser });
  if (auth.currentUser) {
    console.log('[ChatPanel] currentUser 存在，开始加载数据');
    await loadConversations();
    await loadFriends();
    await loadAllFriendRequests();

    // 注册WebSocket事件回调
    onMessageUpdate(handleMessageUpdate);
    onConversationUpdate(handleConversationUpdate);
  } else {
    console.log('[ChatPanel] currentUser 不存在，不加载数据');
  }
  console.log('[ChatPanel] onMounted 结束');
});

onUnmounted(() => {
  console.log('[ChatPanel] onUnmounted，清理 WebSocket 事件');
  // 清理WebSocket事件回调
  offMessageUpdate(handleMessageUpdate);
  offConversationUpdate(handleConversationUpdate);
  // 清除当前会话ID
  setCurrentConversation(null);
});
</script>

<style scoped></style>
