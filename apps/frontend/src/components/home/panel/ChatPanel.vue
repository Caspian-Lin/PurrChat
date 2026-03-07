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
      <!-- 搜索用户 -->
      <div class="flex items-center gap-2 p-3 bg-bg-secondary border-b border-border-color">
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
      <ConversationList
        :conversations="filteredConversations"
        :selected-id="selectedConversation?.id"
        :current-user-id="currentUser?.id"
        @select="handleSelectConversation"
        @show-user="handleShowUserProfile"
        @delete-conversation="handleDeleteConversation"
      />
    </ResizableContainer>

    <!-- 聊天窗口 -->
    <div class="flex-1 flex flex-col bg-bg-tertiary">
      <ChatWindow
        v-if="selectedConversation"
        :conversation="selectedConversation"
        :messages="messages"
        :current-user-id="currentUser?.id"
        @send-message="handleSendMessage"
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
    <UserProfileModal v-model:show="showProfileModal" :user="displayUser" />

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
      :current-user-id="currentUser?.id"
      @show-user-profile="handleShowUserProfile"
      @members-changed="handleGroupUpdated"
    />

    <!-- 通知列表 -->
    <NotificationList :notifications="notifications" @remove-notification="removeNotification" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useAuthController } from '../../../controllers/authController';
import { useConversations } from '../../../composables/useConversations';
import { useFriends } from '../../../composables/useFriends';
import { useChat } from '../../../composables/useChat';
import { useMessageCache } from '../../../services/messageCache';
import { useMessageStore } from '../../../stores/message';
import { useNotification } from '../../../composables/useNotification';
import ConversationList from '../ConversationList.vue';
import ChatWindow from '../ChatWindow.vue';
import UserProfileModal from '../UserProfileModal.vue';
import UserActionsModal from '../UserActionsModal.vue';
import CreateGroupModal from '../CreateGroupModal.vue';
import ConversationDetailModal from '../ConversationDetailModal.vue';
import NotificationList from '../../common/NotificationList.vue';
import ResizableContainer from '../../common/ResizableContainer.vue';
import type { User, Conversation } from '../../../models/types';
import { BsPlusLg, BsXCircle } from 'vue-icons-plus/bs';
// Auth
const auth = useAuthController();
const { currentUser } = auth;

// Composables
const { conversations, loadConversations, createConversation, deleteConversation } =
  useConversations();
const { friends, loadFriends, sendFriendRequest } = useFriends();
const {
  messages,
  loadMessages,
  checkAndLoadIncremental,
  sendMessage,
  exportMessages,
  clearMessages,
} = useChat();
const { addMessage: cacheMessage } = useMessageCache();
const { notifications, removeNotification } = useNotification();
const messageStore = useMessageStore();

// 搜索状态
const searchQuery = ref('');

// 过滤后的会话列表
const filteredConversations = computed(() => {
  if (!searchQuery.value.trim()) {
    return conversations.value;
  }

  const query = searchQuery.value.toLowerCase();
  return conversations.value.filter((conv) => {
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
  });
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

// Computed
const displayUser = computed(() => {
  return selectedUser.value || currentUser;
});

// Handlers
const handleShowUserProfile = (user: User) => {
  selectedUser.value = user;
  showProfileModal.value = true;
};

const handleSelectConversation = async (conversation: Conversation) => {
  selectedConversation.value = conversation;
  selectedUser.value = null;
  // 使用增量加载检查并获取新消息
  await checkAndLoadIncremental(conversation.id);
};

const handleDeleteConversation = async (conversationId: string) => {
  const success = await deleteConversation(conversationId);
  if (success) {
    // 如果删除的是当前选中的会话，清空选中状态
    if (selectedConversation.value?.id === conversationId) {
      selectedConversation.value = null;
      clearMessages();
    }
    // 重新加载会话列表
    await loadConversations();
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

            // 更新message store
            messageStore.addMessage(selectedConversation.value.id, lastMessage);

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

// Watchers
watch(
  () => currentUser,
  async () => {
    if (currentUser) {
      await loadConversations();
      await loadFriends();
    }
  }
);

watch(selectedConversation, async (newConv, oldConv) => {
  if (newConv && newConv.id !== oldConv?.id) {
    // 先清空消息列表，避免消息残留
    clearMessages();
    // 然后加载新会话的消息
    await loadMessages(newConv.id);
  }
});

// Lifecycle
onMounted(async () => {
  console.log('[ChatPanel] onMounted 开始');
  await auth.checkAuth();
  console.log('[ChatPanel] checkAuth 完成', { currentUser });
  if (currentUser) {
    console.log('[ChatPanel] currentUser 存在，开始加载数据');
    await loadConversations();
    await loadFriends();
  } else {
    console.log('[ChatPanel] currentUser 不存在，不加载数据');
  }
  console.log('[ChatPanel] onMounted 结束');
});
</script>

<style scoped></style>
