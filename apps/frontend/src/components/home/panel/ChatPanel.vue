<template>
  <!-- ========== 移动端布局 ========== -->
  <div v-if="isMobile" class="mobile-chat-panel">
    <!-- 会话列表（全屏） -->
    <div v-if="!selectedConversation" class="mobile-conversation-list">
      <!-- 搜索栏 -->
      <div class="mobile-search-bar">
        <div class="mobile-search-input">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索会话..."
            class="mobile-search-field"
            @input="handleSearch"
          />
          <button v-if="searchQuery" class="mobile-search-clear" @click="clearSearch">
            <BsXCircle :size="18" />
          </button>
        </div>
        <button class="mobile-action-btn" title="创建群聊" @click="handleCreateGroup">
          <BsPlusLg :size="22" />
        </button>
      </div>

      <!-- 列表 -->
      <div class="mobile-conversation-scroll">
        <ConversationList
          :conversations="filteredConversations"
          :selected-id="undefined"
          :current-user-id="auth.currentUser?.id"
          @select="handleMobileSelectConversation"
          @show-user="handleShowUserProfile"
          @delete-conversation="handleDeleteConversation"
        />
      </div>
    </div>

    <!-- 聊天窗口（全屏，从右侧滑入） -->
    <div v-else class="mobile-chat-view">
      <!-- 顶部栏 -->
      <div class="mobile-chat-header">
        <button class="mobile-back-btn" @click="handleMobileBack">
          <BsChevronLeft :size="22" />
        </button>
        <div class="mobile-chat-title">
          <span class="mobile-chat-name">{{ selectedConversation.name }}</span>
        </div>
        <button class="mobile-action-btn" @click="handleShowDetail">
          <BsThreeDotsVertical :size="20" />
        </button>
      </div>

      <!-- ChatWindow 组件 -->
      <ChatWindow
        ref="chatWindowRef"
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
    </div>

    <!-- 移动端也需要的弹窗 -->
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

    <UserActionsModal
      v-model:show="showSearchModal"
      :user="selectedSearchUser"
      @send-friend-request="handleSendFriendRequest"
      @start-chat="handleStartChatWithSearchUser"
    />

    <CreateGroupModal
      v-model:show="showCreateGroupModal"
      :friends="friends"
      @group-created="handleGroupCreated"
    />

    <ConversationDetailModal
      v-model:show="showConversationDetailModal"
      :conversation="selectedConversation"
      :current-user-id="auth.currentUser?.id"
      @show-user-profile="handleShowUserProfile"
      @members-changed="handleGroupUpdated"
      @start-chat="handleStartChatFromDetail"
    />

    <NotificationList :notifications="notifications" @remove-notification="removeNotification" />
  </div>

  <!-- ========== 桌面端布局（原始） ========== -->
  <div v-else>
    <BasePanel
      panel-id="chat"
      :initial-sidebar-width="320"
      :min-sidebar-width="250"
      :max-sidebar-width="500"
    >
      <template #sidebar>
        <div class="flex flex-col h-full">
          <!-- 搜索用户 -->
          <div
            class="flex items-center gap-2 px-4 py-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
          >
            <div
              class="flex-1 flex items-center bg-bg-quaternary rounded-[var(--radius-sm)] h-10 px-3"
            >
              <input
                v-model="searchQuery"
                type="text"
                placeholder="搜索会话、消息内容、好友名..."
                class="w-full bg-transparent text-text-primary text-base font-normal outline-none placeholder:text-text-tertiary"
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
      </template>

      <!-- 聊天窗口 -->
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
        <div
          class="w-20 h-20 rounded-full flex items-center justify-center mb-6"
          style="background: var(--message-sent-background)"
        >
          <svg
            class="w-10 h-10"
            style="color: var(--theme-primary)"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
            />
          </svg>
        </div>
        <h3 class="text-lg font-semibold mb-1 text-text-primary">欢迎来到 PurrChat</h3>
        <p class="text-sm">选择一个会话开始聊天</p>
      </div>
    </BasePanel>

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
import { usePlatform } from '../../../composables/usePlatform';
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
import BasePanel from './BasePanel.vue';
import type {
  User,
  Conversation,
  Message,
  Friendship,
  FileMessageContent,
} from '../../../models/types';
import { BsPlusLg, BsXCircle, BsChevronLeft, BsThreeDotsVertical } from 'vue-icons-plus/bs';

// Platform
const { isMobile } = usePlatform();

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

// ChatWindow 组件引用
const chatWindowRef = ref<InstanceType<typeof ChatWindow> | null>(null);

// 消息
const messages = computed(() => {
  if (!selectedConversation.value?.id) return [];
  return messageStore.getMessages(selectedConversation.value.id);
});

// 搜索
const searchQuery = ref('');

const filteredConversations = computed(() => {
  const allConversations = conversations.value;
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
      if (conv.name && conv.name.toLowerCase().includes(query)) return true;
      if (conv.members) {
        for (const member of conv.members) {
          if (member.user && member.user.username.toLowerCase().includes(query)) return true;
        }
      }
      if (conv.last_message && conv.last_message.content.toLowerCase().includes(query)) return true;
      return false;
    })
    .map((conv) => ({
      ...conv,
      unread_count: getUnreadCount(conv.id),
    }));
});

const handleSearch = () => {
  console.log('[ChatPanel] Search query:', searchQuery.value);
};

const clearSearch = () => {
  searchQuery.value = '';
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

const displayUser = computed(() => {
  return selectedUser.value || auth.currentUser;
});

const getUserFriendship = computed(() => {
  if (!selectedUser.value || !auth.currentUser?.id) return null;
  if (selectedUser.value.id === auth.currentUser.id) return null;
  const friendship = friends.value.find(
    (f) => f.friend?.id === selectedUser.value?.id || f.user?.id === selectedUser.value?.id
  );
  if (friendship) return friendship;
  const pendingRequest = allFriendRequests.value.find(
    (r) => r.user?.id === selectedUser.value?.id || r.friend?.id === selectedUser.value?.id
  );
  if (pendingRequest) return pendingRequest;
  return null;
});

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
  setCurrentConversation(conversation.id);
  clearUnreadCount(conversation.id);
  await checkAndLoadIncremental(conversation.id);
};

// 移动端：选择会话（同 desktop 逻辑）
const handleMobileSelectConversation = handleSelectConversation;

// 移动端：返回会话列表
const handleMobileBack = () => {
  selectedConversation.value = null;
  clearMessages();
  setCurrentConversation(null);
};

// WebSocket 事件处理器
const handleMessageUpdate = async (conversationId: string, message: Message) => {
  console.log('[ChatPanel] ===== 收到消息更新事件 =====');
  console.log('[ChatPanel] 消息会话ID:', conversationId);
  console.log('[ChatPanel] 当前选中会话ID:', selectedConversation.value?.id);
  console.log('[ChatPanel] 消息内容:', message.content);
  console.log('[ChatPanel] 消息ID:', message.id);
  console.log('[ChatPanel] 发送者ID:', message.sender_id);

  if (isHidden(conversationId)) {
    console.log('[ChatPanel] 会话被隐藏，现在显示它');
    showConversation(conversationId);
    conversations.value = [...conversations.value];
  }

  if (conversationId !== selectedConversation.value?.id) {
    console.log('[ChatPanel] 不是当前会话，增加未读计数');
    incrementUnreadCount(conversationId);
  }

  if (conversationId === selectedConversation.value?.id) {
    console.log('[ChatPanel] 是当前会话，准备滚动到底部');
    await nextTick();
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
  await loadConversations();
  console.log('[ChatPanel] 会话列表重新加载完成');
  console.log('[ChatPanel] ===== 会话更新事件处理完成 =====');
};

const handleDeleteConversation = async (conversationId: string) => {
  hideConversation(conversationId);
  conversations.value = [...conversations.value];

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

  if (isHidden(selectedConversation.value.id)) {
    console.log('[ChatPanel] 会话被隐藏，现在显示它');
    showConversation(selectedConversation.value.id);
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

            await cacheMessage(selectedConversation.value.id, lastMessage);

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
  await loadConversations();
  console.log('[ChatPanel] Conversations after reload:', conversations.value);
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
  selectedConversation.value = conversation;
  clearUnreadCount(conversation.id);
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

watch(
  () => route.query.conversationId,
  async (conversationId) => {
    if (conversationId && typeof conversationId === 'string') {
      console.log('[ChatPanel] 路由参数中的会话ID:', conversationId);
      await loadConversations();
      const conversation = conversations.value.find((c) => c.id === conversationId);
      if (conversation) {
        console.log('[ChatPanel] 找到会话，选中它:', conversation);
        showConversation(conversation.id);
        conversations.value = [...conversations.value];
        selectedConversation.value = conversation;
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
    clearMessages();
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

    onMessageUpdate(handleMessageUpdate);
    onConversationUpdate(handleConversationUpdate);
  } else {
    console.log('[ChatPanel] currentUser 不存在，不加载数据');
  }
  console.log('[ChatPanel] onMounted 结束');
});

onUnmounted(() => {
  console.log('[ChatPanel] onUnmounted，清理 WebSocket 事件');
  offMessageUpdate(handleMessageUpdate);
  offConversationUpdate(handleConversationUpdate);
  setCurrentConversation(null);
});
</script>

<style scoped>
/* ========== 移动端样式 ========== */
.mobile-chat-panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--background-color);
}

.mobile-conversation-list {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.mobile-search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--surface-color);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.mobile-search-input {
  flex: 1;
  display: flex;
  align-items: center;
  background: var(--bg-quaternary, rgba(0, 0, 0, 0.04));
  border-radius: var(--radius-sm, 8px);
  height: 40px;
  padding: 0 12px;
}

.mobile-search-field {
  width: 100%;
  background: transparent;
  color: var(--text-color);
  font-size: 15px;
  outline: none;
}

.mobile-search-field::placeholder {
  color: var(--text-tertiary-color);
}

.mobile-search-clear {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  background: none;
  border: none;
  color: var(--text-tertiary-color);
  cursor: pointer;
}

.mobile-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  min-width: 40px;
  background: none;
  border: none;
  color: var(--theme-primary);
  cursor: pointer;
  border-radius: var(--radius-sm, 8px);
  transition: background 0.15s ease;
  -webkit-tap-highlight-color: transparent;
}

.mobile-action-btn:active {
  background: var(--hover-bg, rgba(0, 0, 0, 0.06));
}

.mobile-conversation-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

/* 聊天视图 */
.mobile-chat-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  animation: slideInRight 0.2s ease-out;
}

@keyframes slideInRight {
  from {
    transform: translateX(100%);
  }
  to {
    transform: translateX(0);
  }
}

.mobile-chat-header {
  display: flex;
  align-items: center;
  padding: 8px 4px;
  height: 52px;
  min-height: 52px;
  background: var(--surface-color);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.mobile-back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: none;
  border: none;
  color: var(--text-primary-color);
  cursor: pointer;
  border-radius: var(--radius-sm, 8px);
  -webkit-tap-highlight-color: transparent;
}

.mobile-back-btn:active {
  background: var(--hover-bg, rgba(0, 0, 0, 0.06));
}

.mobile-chat-title {
  flex: 1;
  padding: 0 4px;
  overflow: hidden;
}

.mobile-chat-name {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
