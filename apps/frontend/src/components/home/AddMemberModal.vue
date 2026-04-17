<template>
  <BaseModal
    :show="show"
    title="添加成员"
    class="max-w-md"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col gap-4">
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          搜索好友
        </label>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="输入用户名或UID搜索..."
          class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
          style="
            background: var(--surface-color);
            border-color: var(--border-color);
            color: var(--text-color);
          "
        />
      </div>

      <!-- 搜索结果 -->
      <div
        v-if="searchQuery"
        class="max-h-64 overflow-y-auto rounded-lg scrollable"
        style="background: var(--surface-color)"
      >
        <div
          v-if="searchLoading"
          class="p-4 text-center"
          style="color: var(--text-secondary-color)"
        >
          搜索中...
        </div>
        <div
          v-else-if="searchResults.length === 0"
          class="p-4 text-center"
          style="color: var(--text-secondary-color)"
        >
          没有找到用户
        </div>
        <div
          v-for="user in searchResults"
          :key="user.id"
          class="flex items-center gap-3 p-3 border-b"
          style="border-color: var(--border-color)"
        >
          <div class="w-10 h-10 roundrect overflow-hidden flex-shrink-0">
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
          <div class="flex-1">
            <div class="font-medium" style="color: var(--text-color)">
              {{ user.username }}
            </div>
            <div class="text-xs" style="color: var(--text-secondary-color)">
              UID: {{ user.uid }}
            </div>
          </div>
          <button
            class="px-3 py-1.5 text-sm bg-accent-color text-white rounded-md hover:opacity-80 transition-colors"
            @click="handleAddMember(user)"
          >
            添加
          </button>
        </div>
      </div>

      <!-- 好友列表 -->
      <div v-else>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          选择好友
        </label>
        <div
          class="max-h-64 overflow-y-auto rounded-lg scrollable"
          style="background: var(--surface-color)"
        >
          <div
            v-for="friend in availableFriends"
            :key="friend.friend?.id"
            class="flex items-center gap-3 p-3 border-b"
            style="border-color: var(--border-color)"
          >
            <div class="w-10 h-10 roundrect overflow-hidden flex-shrink-0">
              <img
                v-if="friend.friend?.avatar_url"
                :src="friend.friend.avatar_url"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center font-bold text-white"
                style="background: var(--theme-gradient)"
              >
                {{ friend.friend?.username?.charAt(0) || 'U' }}
              </div>
            </div>
            <div class="flex-1">
              <div class="font-medium" style="color: var(--text-color)">
                {{ friend.friend?.username }}
              </div>
            </div>
            <button
              class="px-3 py-1.5 text-sm bg-accent-color text-white rounded-md hover:opacity-80 transition-colors"
              @click="handleAddMember(friend.friend!)"
            >
              添加
            </button>
          </div>
          <div
            v-if="availableFriends.length === 0"
            class="p-4 text-center"
            style="color: var(--text-secondary-color)"
          >
            没有可添加的好友
          </div>
        </div>
      </div>

      <div class="flex gap-3">
        <button
          class="flex-1 px-4 py-2 bg-bg-secondary text-text-primary rounded-md hover:bg-hover-bg transition-colors"
          @click="emit('update:show', false)"
        >
          取消
        </button>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import BaseModal from '../common/BaseModal.vue';
import { api } from '../../models/api';
import type { User, Friendship, Enrollment } from '../../models/types';

interface Props {
  show: boolean;
  conversationId: string;
  currentMembers: Enrollment[];
  friends?: Friendship[];
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  'member-added': [];
}>();

const searchQuery = ref('');
const searchLoading = ref(false);
const searchResults = ref<User[]>([]);

// 获取可添加的好友（排除已经在群里的）
const availableFriends = computed(() => {
  const memberIds = new Set(props.currentMembers.map((m) => m.user_id));
  return props.friends?.filter((f) => !memberIds.has(f.friend?.id || '')) || [];
});

// 搜索用户
const searchUsers = async () => {
  if (!searchQuery.value.trim()) {
    searchResults.value = [];
    return;
  }

  searchLoading.value = true;
  try {
    const response = await api.searchUsers(searchQuery.value.trim());
    if (response.success && response.data) {
      // 排除已经在群里的用户
      const memberIds = new Set(props.currentMembers.map((m) => m.user_id));
      searchResults.value = response.data.filter((u) => !memberIds.has(u.id));
    }
  } catch (error) {
    console.error('Failed to search users:', error);
  } finally {
    searchLoading.value = false;
  }
};

// 添加成员
const handleAddMember = async (user: User) => {
  try {
    const response = await api.addMemberToConversation({
      conversation_id: props.conversationId,
      user_id: user.id,
      role: 'member',
    });

    if (response.success) {
      emit('member-added');
      emit('update:show', false);
      // 重置搜索
      searchQuery.value = '';
      searchResults.value = [];
    }
  } catch (error) {
    console.error('Failed to add member:', error);
    alert('添加成员失败，请重试');
  }
};

// 监听搜索输入
let searchTimeout: ReturnType<typeof setTimeout> | null = null;
watch(searchQuery, () => {
  if (searchTimeout) {
    clearTimeout(searchTimeout);
  }
  searchTimeout = setTimeout(() => {
    searchUsers();
  }, 300);
});
</script>

<style scoped></style>
