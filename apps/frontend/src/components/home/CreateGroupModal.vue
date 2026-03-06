<template>
  <BaseModal
    :show="show"
    title="创建群聊"
    class="max-w-md"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col gap-4">
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          群聊名称
        </label>
        <input
          v-model="groupName"
          type="text"
          placeholder="请输入群聊名称"
          class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
          style="
            background: var(--surface-color);
            border-color: var(--border-color);
            color: var(--text-color);
          "
          maxlength="100"
        />
      </div>

      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          选择成员
        </label>
        <div class="mb-2">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索好友..."
            class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
            style="
              background: var(--surface-color);
              border-color: var(--border-color);
              color: var(--text-color);
            "
          />
        </div>
        <div class="max-h-64 overflow-y-auto rounded-lg" style="background: var(--surface-color)">
          <div
            v-for="friend in filteredFriends"
            :key="friend.friend?.id"
            class="flex items-center gap-3 p-3 border-b"
            style="border-color: var(--border-color)"
          >
            <input
              v-if="friend.friend?.id"
              type="checkbox"
              :id="`friend-${friend.friend.id}`"
              :value="friend.friend.id"
              v-model="selectedMembers"
              class="w-4 h-4 rounded"
            />
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
          </div>
          <div
            v-if="filteredFriends.length === 0"
            class="p-4 text-center"
            style="color: var(--text-secondary-color)"
          >
            没有找到好友
          </div>
        </div>
      </div>

      <div
        class="flex items-center justify-between text-sm"
        style="color: var(--text-secondary-color)"
      >
        <span>已选择 {{ selectedMembers.length }} 位成员</span>
        <span>最少需要 2 位成员</span>
      </div>

      <div class="flex gap-3">
        <button
          class="flex-1 px-4 py-2 bg-bg-secondary text-text-primary rounded-md hover:bg-hover-bg transition-colors"
          @click="emit('update:show', false)"
        >
          取消
        </button>
        <button
          class="flex-1 px-4 py-2 bg-primary text-white rounded-md hover:bg-primary-dark transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!isValid"
          @click="handleCreateGroup"
        >
          创建群聊
        </button>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import BaseModal from '../common/BaseModal.vue';
import { api } from '../../models/api';
import type { Friendship } from '../../models/types';

interface Props {
  show: boolean;
  friends: Friendship[];
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  'group-created': [conversationId: string];
}>();

const groupName = ref('');
const searchQuery = ref('');
const selectedMembers = ref<string[]>([]);

// 过滤好友列表
const filteredFriends = computed(() => {
  if (!searchQuery.value) {
    return props.friends;
  }
  const query = searchQuery.value.toLowerCase();
  return props.friends.filter((friend) => friend.friend?.username?.toLowerCase().includes(query));
});

// 验证表单
const isValid = computed(() => {
  const valid = groupName.value.trim().length > 0 && selectedMembers.value.length >= 2;
  console.log('[CreateGroupModal] isValid check:', {
    groupName: groupName.value,
    groupNameLength: groupName.value.trim().length,
    selectedMembers: selectedMembers.value,
    selectedMembersLength: selectedMembers.value.length,
    valid: valid
  });
  return valid;
});

// 创建群聊
const handleCreateGroup = async () => {
  if (!isValid.value) {
    return;
  }

  try {
    const response = await api.createGroup({
      name: groupName.value.trim(),
      members: selectedMembers.value,
    });

    if (response.success && response.data) {
      emit('group-created', response.data.id);
      emit('update:show', false);
      // 重置表单
      groupName.value = '';
      searchQuery.value = '';
      selectedMembers.value = [];
    }
  } catch (error) {
    console.error('Failed to create group:', error);
    alert('创建群聊失败，请重试');
  }
};
</script>

<style scoped>
.roundrect {
  border-radius: 8px;
}
</style>
