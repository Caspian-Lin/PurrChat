<template>
  <div class="flex flex-row h-screen">
    <!-- 标题状态栏(最左侧) -->
    <div
      class="flex flex-col items-center justify-between py-4 gap-4 text-white shadow-lg relative overflow-hidden"
    >
      <DynamicBackground />
      <!-- 标题 -->
      <div
        class="relative z-10 flex-0 text-xl font-bold inset-y-0 left-0"
        style="color: var(--theme-primary)"
      >
        Purr <br />Chat
      </div>

      <!-- 会话和好友按钮 - 靠上 -->
      <div class="relative z-10 flex flex-col items-center gap-2">
        <n-button
          class="aspect-1 w-12 h-12"
          :type="activeTab === 'conversations' ? 'primary' : 'default'"
          @click="activeTab = 'conversations'"
        >
          <BsChatLeftDotsFill />
        </n-button>
        <n-button
          class="aspect-1 w-12 h-12"
          :type="activeTab === 'friends' ? 'primary' : 'default'"
          @click="activeTab = 'friends'"
        >
          <BsFillPersonLinesFill />
        </n-button>
      </div>

      <!-- 底部区域 - 主题切换、个人资料、退出登录 -->
      <div class="relative z-10 flex flex-col items-center gap-4 mt-auto">
        <ThemeSwitcher />
        <div
          class="flex flex-0 items-center gap-2 cursor-pointer px-2 py-2 rounded-lg hover:bg-white/10 transition-colors"
          @click="showProfileModal = true"
        >
          <img
            v-if="currentUser?.avatar_url"
            :src="currentUser.avatar_url"
            alt="avatar"
            class="w-10 h-10 rounded-full object-cover"
          />
          <div
            v-else
            class="w-10 h-10 rounded-full flex items-center justify-center font-bold text-white"
            style="background: var(--theme-gradient)"
          >
            {{ currentUser?.username?.charAt(0) || 'U' }}
          </div>
        </div>
        <n-button quaternary class="!text-white !border-white/50" @click="handleLogout">
          退出登录
        </n-button>
      </div>
    </div>

    <!-- 主内容区 -->
    <main class="flex flex-1 overflow-hidden">
      <!-- 左侧边栏 -->
      <div
        class="w-80 flex flex-col border-r"
        style="background: var(--surface-color); border-color: var(--border-color)"
      >
        <!-- 会话列表 -->
        <div v-if="activeTab === 'conversations'" class="flex-1 overflow-y-auto">
          <div
            v-for="conversation in conversations"
            :key="conversation.id"
            :class="[
              'flex items-center gap-4 p-4 cursor-pointer transition-colors border-b',
              { 'bg-blue-50 dark:bg-blue-900/20': selectedConversation?.id === conversation.id },
            ]"
            style="border-color: var(--border-color)"
            @click="selectConversation(conversation)"
          >
            <div class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0">
              <img
                v-if="getUserAvatar(getOtherUser(conversation))"
                :src="getUserAvatar(getOtherUser(conversation))"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center font-bold text-white"
                style="background: var(--theme-gradient)"
              >
                {{ getUserUsername(getOtherUser(conversation)).charAt(0) }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-semibold truncate" style="color: var(--text-color)">
                  {{ getUserUsername(getOtherUser(conversation)) }}
                </span>
                <n-tag v-if="conversation.has_pending_request" type="warning" size="small" round>
                  待处理
                </n-tag>
              </div>
              <div class="text-sm truncate" style="color: var(--text-secondary-color)">
                {{ conversation.last_message?.content || '暂无消息' }}
              </div>
            </div>
          </div>
          <div
            v-if="conversations.length === 0"
            class="flex flex-col items-center justify-center h-full text-center p-8"
            style="color: var(--text-secondary-color)"
          >
            <p>暂无会话</p>
          </div>
        </div>

        <!-- 好友列表 -->
        <div v-if="activeTab === 'friends'" class="flex-1 overflow-y-auto">
          <div
            v-for="friendship in friends"
            :key="friendship.id"
            class="flex items-center gap-4 p-4 cursor-pointer transition-colors border-b hover:bg-gray-100 dark:hover:bg-gray-700"
            style="border-color: var(--border-color)"
            @click="showFriendInfo(friendship)"
          >
            <div class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0">
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
              <div class="font-semibold truncate" style="color: var(--text-color)">
                {{ friendship.friend?.username }}
              </div>
              <div class="text-sm" style="color: var(--text-secondary-color)">
                UID: {{ friendship.friend?.uid }}
              </div>
            </div>
          </div>
          <div
            v-if="friends.length === 0"
            class="flex flex-col items-center justify-center h-full text-center p-8"
            style="color: var(--text-secondary-color)"
          >
            <p>暂无好友</p>
          </div>
        </div>

        <!-- 搜索用户 -->
        <div class="p-4 border-t" style="border-color: var(--border-color)">
          <n-input
            v-model:value="searchQuery"
            placeholder="搜索用户 (UID/手机号/邮箱)"
            @keyup.enter="searchUsers"
            class="mb-2"
          />
          <n-button type="primary" block @click="searchUsers"> 搜索 </n-button>
        </div>

        <!-- 搜索结果 -->
        <div
          v-if="searchResults.length > 0"
          class="max-h-80 overflow-y-auto border-t"
          style="border-color: var(--border-color)"
        >
          <div
            v-for="user in searchResults"
            :key="user.id"
            class="flex items-center gap-4 p-4 cursor-pointer transition-colors border-b hover:bg-gray-100 dark:hover:bg-gray-700"
            style="border-color: var(--border-color)"
            @click="showSearchUserActions(user)"
          >
            <div class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0">
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
                {{ user.username?.charAt(0) || 'U' }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-semibold truncate" style="color: var(--text-color)">
                {{ user.username }}
              </div>
              <div class="text-sm" style="color: var(--text-secondary-color)">
                UID: {{ user.uid }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧聊天区 -->
      <div class="flex-1 flex flex-col" style="background: var(--background-color)">
        <!-- 聊天窗口 -->
        <div v-if="selectedConversation" class="flex flex-col h-full">
          <!-- 聊天头部 -->
          <div
            class="flex items-center justify-between px-6 py-4 border-b"
            style="background: var(--card-background); border-color: var(--border-color)"
          >
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0">
                <img
                  v-if="getUserAvatar(getOtherUser(selectedConversation))"
                  :src="getUserAvatar(getOtherUser(selectedConversation))"
                  alt="avatar"
                  class="w-full h-full object-cover"
                />
                <div
                  v-else
                  class="w-full h-full flex items-center justify-center font-bold text-white"
                  style="background: var(--theme-gradient)"
                >
                  {{ getUserUsername(getOtherUser(selectedConversation)).charAt(0) }}
                </div>
              </div>
              <div>
                <div class="font-semibold" style="color: var(--text-color)">
                  {{ getUserUsername(getOtherUser(selectedConversation)) }}
                </div>
                <div class="text-sm" style="color: var(--text-secondary-color)">
                  UID: {{ getOtherUser(selectedConversation)?.uid }}
                </div>
              </div>
            </div>

            <!-- 好友请求操作 -->
            <div v-if="selectedConversation.has_pending_request" class="flex gap-2">
              <n-button type="success" @click="handleFriendRequest('accept')"> 接受 </n-button>
              <n-button type="error" @click="handleFriendRequest('reject')"> 拒绝 </n-button>
            </div>
          </div>

          <!-- 消息列表 -->
          <div class="flex-1 overflow-y-auto p-6 space-y-4" ref="messagesContainer">
            <div
              v-for="message in messages"
              :key="message.id"
              :class="[
                'flex gap-3 max-w-[70%]',
                { 'flex-row-reverse ml-auto': message.sender_id === currentUser?.id },
              ]"
            >
              <div class="w-10 h-10 rounded-full overflow-hidden flex-shrink-0">
                <img
                  v-if="message.sender?.avatar_url"
                  :src="message.sender.avatar_url"
                  alt="avatar"
                  class="w-full h-full object-cover"
                />
                <div
                  v-else
                  class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
                  style="background: var(--theme-gradient)"
                >
                  {{ message.sender?.username?.charAt(0) || '?' }}
                </div>
              </div>
              <div class="flex flex-col gap-1">
                <div class="text-xs font-semibold text-gray-600 dark:text-gray-400">
                  {{ message.sender?.username }}
                </div>
                <div
                  :class="[
                    'px-4 py-2 rounded-2xl break-words',
                    message.sender_id === currentUser?.id ? 'text-white' : '',
                  ]"
                  :style="{
                    background:
                      message.sender_id === currentUser?.id
                        ? 'var(--message-sent-background)'
                        : 'var(--message-received-background)',
                    color: message.sender_id === currentUser?.id ? 'white' : 'var(--text-color)',
                  }"
                >
                  {{ message.content }}
                </div>
                <div class="text-xs" style="color: var(--text-tertiary-color)">
                  {{ formatTime(message.created_at) }}
                </div>
              </div>
            </div>
          </div>

          <!-- 消息输入区 -->
          <div
            class="flex gap-3 px-6 py-4 border-t"
            style="background: var(--card-background); border-color: var(--border-color)"
          >
            <n-input
              v-model:value="newMessage"
              type="textarea"
              placeholder="输入消息..."
              :autosize="{ minRows: 1, maxRows: 4 }"
              @keydown.enter.prevent="sendMessage"
              class="flex-1"
            />
            <n-button
              type="primary"
              :disabled="!newMessage.trim()"
              @click="sendMessage"
              class="!h-10"
            >
              发送
            </n-button>
          </div>
        </div>

        <!-- 好友信息窗口 -->
        <div v-else-if="selectedFriend" class="flex flex-col h-full p-8 overflow-y-auto">
          <div class="flex items-center justify-between mb-8">
            <h2 class="text-2xl font-bold" style="color: var(--text-color)">好友信息</h2>
            <n-button quaternary @click="selectedFriend = null"> 关闭 </n-button>
          </div>
          <div class="flex flex-col items-center gap-8">
            <div class="w-36 h-36 rounded-full overflow-hidden">
              <img
                v-if="selectedFriend.friend?.avatar_url"
                :src="selectedFriend.friend.avatar_url"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center font-bold text-white text-4xl"
                style="background: var(--theme-gradient)"
              >
                {{ selectedFriend.friend?.username?.charAt(0) || '?' }}
              </div>
            </div>
            <div class="w-full max-w-md space-y-4">
              <div
                class="flex justify-between p-3 rounded-lg"
                style="background: var(--surface-color)"
              >
                <span class="font-semibold" style="color: var(--text-secondary-color)">昵称:</span>
                <span style="color: var(--text-color)">{{ selectedFriend.friend?.username }}</span>
              </div>
              <div
                class="flex justify-between p-3 rounded-lg"
                style="background: var(--surface-color)"
              >
                <span class="font-semibold" style="color: var(--text-secondary-color)">UID:</span>
                <span style="color: var(--text-color)">{{ selectedFriend.friend?.uid }}</span>
              </div>
              <div
                class="flex justify-between p-3 rounded-lg"
                style="background: var(--surface-color)"
              >
                <span class="font-semibold" style="color: var(--text-secondary-color)"
                  >用户名:</span
                >
                <span style="color: var(--text-color)">{{ selectedFriend.friend?.username }}</span>
              </div>
              <div
                v-if="selectedFriend.friend?.email"
                class="flex justify-between p-3 rounded-lg"
                style="background: var(--surface-color)"
              >
                <span class="font-semibold" style="color: var(--text-secondary-color)">邮箱:</span>
                <span style="color: var(--text-color)">
                  {{ selectedFriend.friend.email }}
                  <span v-if="!selectedFriend.friend.email_verified" class="text-orange-500 text-sm"
                    >(未验证)</span
                  >
                </span>
              </div>
              <div
                v-if="selectedFriend.friend?.phone"
                class="flex justify-between p-3 rounded-lg"
                style="background: var(--surface-color)"
              >
                <span class="font-semibold" style="color: var(--text-secondary-color)"
                  >手机号:</span
                >
                <span style="color: var(--text-color)">
                  {{ selectedFriend.friend.phone }}
                  <span v-if="!selectedFriend.friend.phone_verified" class="text-orange-500 text-sm"
                    >(未验证)</span
                  >
                </span>
              </div>
            </div>
            <n-button type="primary" size="large" @click="startChatWithFriend"> 发消息 </n-button>
          </div>
        </div>

        <!-- 空状态 -->
        <div
          v-else
          class="flex-1 flex flex-col items-center justify-center"
          style="color: var(--text-secondary-color)"
        >
          <div class="text-6xl mb-4">💬</div>
          <h3 class="text-2xl font-semibold mb-2" style="color: var(--text-color)">
            欢迎来到 PurrChat
          </h3>
          <p>选择一个会话开始聊天</p>
        </div>
      </div>
    </main>

    <!-- 个人资料弹窗 -->
    <n-modal v-model:show="showProfileModal" preset="card" title="个人资料" class="max-w-md">
      <div class="flex flex-col items-center gap-6">
        <div class="w-28 h-28 rounded-full overflow-hidden">
          <img
            v-if="currentUser?.avatar_url"
            :src="currentUser.avatar_url"
            alt="avatar"
            class="w-full h-full object-cover"
          />
          <div
            v-else
            class="w-full h-full flex items-center justify-center font-bold text-white text-4xl"
            style="background: var(--theme-gradient)"
          >
            {{ currentUser?.username?.charAt(0) || 'U' }}
          </div>
        </div>
        <div class="w-full space-y-4">
          <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
            <span class="font-semibold" style="color: var(--text-secondary-color)">UID:</span>
            <span style="color: var(--text-color)">{{ currentUser?.uid }}</span>
          </div>
          <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
            <span class="font-semibold" style="color: var(--text-secondary-color)">昵称:</span>
            <span style="color: var(--text-color)">{{ currentUser?.username }}</span>
          </div>
          <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
            <span class="font-semibold" style="color: var(--text-secondary-color)">用户名:</span>
            <span style="color: var(--text-color)">{{ currentUser?.username }}</span>
          </div>
          <div
            v-if="currentUser?.email"
            class="flex justify-between p-3 rounded-lg"
            style="background: var(--surface-color)"
          >
            <span class="font-semibold" style="color: var(--text-secondary-color)">邮箱:</span>
            <span style="color: var(--text-color)">
              {{ currentUser.email }}
              <span v-if="!currentUser.email_verified" class="text-orange-500 text-sm"
                >(未验证)</span
              >
            </span>
          </div>
          <div
            v-if="currentUser?.phone"
            class="flex justify-between p-3 rounded-lg"
            style="background: var(--surface-color)"
          >
            <span class="font-semibold" style="color: var(--text-secondary-color)">手机号:</span>
            <span style="color: var(--text-color)">
              {{ currentUser.phone }}
              <span v-if="!currentUser.phone_verified" class="text-orange-500 text-sm"
                >(未验证)</span
              >
            </span>
          </div>
        </div>
      </div>
    </n-modal>

    <!-- 搜索用户操作弹窗 -->
    <n-modal v-model:show="showSearchModal" preset="card" title="用户操作" class="max-w-md">
      <div class="flex flex-col gap-6">
        <div
          class="flex items-center gap-4 p-4 rounded-lg"
          style="background: var(--surface-color)"
        >
          <div class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0">
            <img
              v-if="selectedSearchUser?.avatar_url"
              :src="selectedSearchUser.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white"
              style="background: var(--theme-gradient)"
            >
              {{ selectedSearchUser?.username?.charAt(0) || 'U' }}
            </div>
          </div>
          <div>
            <div class="font-semibold" style="color: var(--text-color)">
              {{ selectedSearchUser?.username }}
            </div>
            <div class="text-sm" style="color: var(--text-secondary-color)">
              UID: {{ selectedSearchUser?.uid }}
            </div>
          </div>
        </div>
        <div class="flex gap-4">
          <n-button type="primary" block @click="sendFriendRequestToUser"> 发送好友请求 </n-button>
          <n-button block @click="startChatWithUser"> 开始聊天 </n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue';
import { NButton, NInput, NTag, NModal, useMessage } from 'naive-ui';
import { BsChatLeftDotsFill, BsFillPersonLinesFill } from 'vue-icons-plus/bs';
import { useAuthController } from '../controllers/authController';
import { api } from '../models/api';
import type { User, Conversation, Message, Friendship } from '../models/types';
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
import DynamicBackground from '../components/DynamicBackground.vue';

const auth = useAuthController();
const { currentUser, handleLogout } = auth;
const message = useMessage();

// 状态
const activeTab = ref<'conversations' | 'friends'>('conversations');
const conversations = ref<Conversation[]>([]);
const friends = ref<Friendship[]>([]);
const messages = ref<Message[]>([]);
const selectedConversation = ref<Conversation | null>(null);
const selectedFriend = ref<Friendship | null>(null);
const newMessage = ref('');
const searchQuery = ref('');
const searchResults = ref<User[]>([]);
const showProfileModal = ref(false);
const showSearchModal = ref(false);
const selectedSearchUser = ref<User | null>(null);
const messagesContainer = ref<HTMLElement | null>(null);

// 获取会话列表
const loadConversations = async () => {
  try {
    const response = await api.getConversations();
    if (response.success && response.data) {
      conversations.value = response.data;
    }
  } catch (error) {
    console.error('Failed to load conversations:', error);
  }
};

// 获取好友列表
const loadFriends = async () => {
  try {
    const response = await api.getFriends();
    if (response.success && response.data) {
      friends.value = response.data;
    }
  } catch (error) {
    console.error('Failed to load friends:', error);
  }
};

// 获取消息
const loadMessages = async (conversationId: string) => {
  try {
    const response = await api.getMessages(conversationId);
    if (response.success && response.data) {
      messages.value = response.data;
      scrollToBottom();
    }
  } catch (error) {
    console.error('Failed to load messages:', error);
  }
};

// 选择会话
const selectConversation = (conversation: Conversation) => {
  selectedConversation.value = conversation;
  selectedFriend.value = null;
  loadMessages(conversation.id);
};

// 显示好友信息
const showFriendInfo = (friendship: Friendship) => {
  selectedFriend.value = friendship;
  selectedConversation.value = null;
};

// 开始与好友聊天
const startChatWithFriend = async () => {
  if (!selectedFriend.value?.friend?.id) return;

  try {
    const response = await api.createConversation({
      target_user_id: selectedFriend.value.friend.id,
    });

    if (response.success && response.data) {
      await loadConversations();
      const conversation = conversations.value.find((c) => c.id === response.data?.id);
      if (conversation) {
        selectConversation(conversation);
      }
    }
  } catch (error) {
    console.error('Failed to start chat:', error);
  }
};

// 搜索用户
const searchUsers = async () => {
  if (!searchQuery.value.trim()) return;

  try {
    const response = await api.searchUsers(searchQuery.value);
    if (response.success && response.data) {
      searchResults.value = response.data;
    }
  } catch (error) {
    console.error('Failed to search users:', error);
  }
};

// 显示搜索用户操作
const showSearchUserActions = (user: User) => {
  selectedSearchUser.value = user;
  showSearchModal.value = true;
};

// 发送好友请求
const sendFriendRequestToUser = async () => {
  if (!selectedSearchUser.value?.id) return;

  try {
    const response = await api.sendFriendRequest({
      target_user_id: selectedSearchUser.value.id,
    });

    if (response.success) {
      message.success('好友请求已发送');
      showSearchModal.value = false;
      searchResults.value = [];
      searchQuery.value = '';
      await loadConversations();
    }
  } catch (error) {
    console.error('Failed to send friend request:', error);
    message.error('发送好友请求失败');
  }
};

// 开始与搜索用户聊天
const startChatWithUser = async () => {
  if (!selectedSearchUser.value?.id) return;

  try {
    const response = await api.createConversation({
      target_user_id: selectedSearchUser.value.id,
    });

    if (response.success && response.data) {
      showSearchModal.value = false;
      searchResults.value = [];
      searchQuery.value = '';
      await loadConversations();
      const conversation = conversations.value.find((c) => c.id === response.data?.id);
      if (conversation) {
        selectConversation(conversation);
      }
    }
  } catch (error) {
    console.error('Failed to start chat:', error);
  }
};

// 发送消息
const sendMessage = async () => {
  if (!selectedConversation.value?.id || !newMessage.value.trim()) return;

  try {
    const response = await api.sendMessage({
      conversation_id: selectedConversation.value.id,
      content: newMessage.value,
      msg_type: 'text',
    });

    if (response.success && response.data) {
      newMessage.value = '';
      messages.value.push(response.data);
      scrollToBottom();
    }
  } catch (error) {
    console.error('Failed to send message:', error);
    message.error('发送消息失败');
  }
};

// 处理好友请求
const handleFriendRequest = async (action: 'accept' | 'reject') => {
  if (!selectedConversation.value?.id) return;

  try {
    const response = await api.handleFriendRequest({
      conversation_id: selectedConversation.value.id,
      action,
    });

    if (response.success) {
      message.success(action === 'accept' ? '已接受好友请求' : '已拒绝好友请求');
      await loadConversations();
      await loadFriends();
    }
  } catch (error) {
    console.error('Failed to handle friend request:', error);
    message.error('操作失败');
  }
};

// 获取会话中的对方用户
const getOtherUser = (conversation: Conversation): User | undefined => {
  if (!currentUser.value) return undefined;
  return conversation.user1_id === currentUser.value.id ? conversation.user2 : conversation.user1;
};

// 安全地获取用户头像URL
const getUserAvatar = (user: User | undefined): string | undefined => {
  return user?.avatar_url;
};

// 安全地获取用户昵称
const getUserUsername = (user: User | undefined): string => {
  return user?.username || '未知用户';
};

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
  }
};

// 格式化时间
const formatTime = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  return date.toLocaleDateString();
};

// 监听当前用户变化
watch(currentUser, async () => {
  if (currentUser.value) {
    await loadConversations();
    await loadFriends();
  }
});

onMounted(async () => {
  await auth.checkAuth();
  if (currentUser.value) {
    await loadConversations();
    await loadFriends();
  }
});
</script>

<style scoped></style>
