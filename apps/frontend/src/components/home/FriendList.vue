<template>
  <div class="flex-1 min-h-0 overflow-y-auto">
    <div class="px-2 pt-2 pb-0.5">
      <BaseListItem
        v-for="friendship in friends"
        :key="friendship.id"
        @click="$emit('select', friendship)"
      >
        <template #avatar>
          <div
            class="w-11 h-11 rounded-[var(--radius-md)] overflow-hidden cursor-pointer"
            @click.stop="$emit('show-user', friendship.friend!)"
          >
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
        </template>

        <!-- 内容 -->
        <div class="flex items-center gap-2">
          <span class="font-semibold text-[15px] truncate text-text-primary">
            {{ friendship.friend?.username }}
          </span>
          <span
            v-if="friendship.status !== 'accepted'"
            :class="[
              'text-xs px-1.5 py-0.5 rounded-[var(--radius-xs)]',
              getFriendshipStatusColor(friendship.status),
            ]"
            class="bg-bg-secondary"
          >
            {{ formatFriendshipStatus(friendship.status) }}
          </span>
        </div>
        <div class="text-sm text-text-secondary truncate">UID: {{ friendship.friend?.uid }}</div>
      </BaseListItem>
    </div>

    <div
      v-if="friends.length === 0"
      class="flex flex-col items-center justify-center h-full text-center p-8 text-text-tertiary"
    >
      <p>暂无好友</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { formatFriendshipStatus, getFriendshipStatusColor } from '../../utils/userHelpers';
import type { Friendship } from '../../models/types';
import BaseListItem from '../common/BaseListItem.vue';

interface Props {
  friends: Friendship[];
}

defineProps<Props>();

defineEmits<{
  select: [friendship: Friendship];
  'show-user': [user: any];
}>();
</script>
