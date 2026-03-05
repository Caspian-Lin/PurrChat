<template>
  <div class="flex h-full">
    <!-- 好友列表 -->
    <div
      class="flex flex-col min-w-[200px] max-w-[400px] bg-bg-primary border-r border-border-color"
    >
      <!-- 搜索好友 -->
      <div class="flex items-center gap-2 p-3 bg-bg-secondary border-b border-border-color">
        <div
          class="flex-1 flex items-center justify-center bg-bg-quaternary rounded-md h-[40px] px-3"
        >
          <div class="text-text-tertiary text-base font-normal">搜索好友...</div>
        </div>
        <div
          class="w-[40px] h-[40px] bg-accent-color rounded-md hover:opacity-80 transition-opacity cursor-pointer"
        />
      </div>

      <!-- 好友列表 -->
      <FriendList
        :friends="friends"
        @select="handleSelectFriend"
        @show-user="handleShowUserProfile"
      />
    </div>

    <!-- 好友信息窗口 -->
    <div class="flex-1 flex flex-col bg-bg-tertiary">
      <FriendInfoModal
        v-if="selectedFriend"
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useAuthController } from '../../../controllers/authController';
import { useFriends } from '../../../composables/useFriends';
import { useConversations } from '../../../composables/useConversations';
import { useRouter } from 'vue-router';
import FriendList from '../FriendList.vue';
import FriendInfoModal from '../FriendInfoModal.vue';
import UserProfileModal from '../UserProfileModal.vue';
import type { User, Friendship } from '../../../models/types';

// Auth
const auth = useAuthController();
const { currentUser } = auth;

// Composables
const { friends, loadFriends } = useFriends();
const { createConversation } = useConversations();
const router = useRouter();

// State
const selectedFriend = ref<Friendship | null>(null);
const selectedUser = ref<User | null>(null);
const showProfileModal = ref(false);

// Computed
const displayUser = computed(() => {
  return selectedUser.value || currentUser.value;
});

// Handlers
const handleShowUserProfile = (user: User) => {
  selectedUser.value = user;
  showProfileModal.value = true;
};

const handleSelectFriend = (friendship: Friendship) => {
  selectedFriend.value = friendship;
};

const handleStartChatWithFriend = async () => {
  if (!selectedFriend.value?.friend?.id) return;

  const conversation = await createConversation(selectedFriend.value.friend.id);
  if (conversation) {
    // 跳转到聊天面板
    router.push('/chat');
  }
};

// Watchers
watch(currentUser, async () => {
  if (currentUser.value) {
    await loadFriends();
  }
});

// Lifecycle
onMounted(async () => {
  await auth.checkAuth();
  if (currentUser.value) {
    await loadFriends();
  }
});
</script>

<style scoped></style>
