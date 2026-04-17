<template>
  <CustomScrollbar class="flex-1 min-h-0">
    <div
      v-for="friendship in friends"
      :key="friendship.id"
      class="flex items-center gap-4 p-4 cursor-pointer transition-all duration-150 border-b hover:bg-hover-bg"
      style="border-color: var(--border-color)"
      @click="$emit('select', friendship)"
    >
      <div
        class="w-12 h-12 rounded-xl overflow-hidden flex-shrink-0 cursor-pointer"
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
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <span class="font-semibold truncate" style="color: var(--text-color)">
            {{ friendship.friend?.username }}
          </span>
          <span
            v-if="friendship.status !== 'accepted'"
            :class="['text-xs px-2 py-1 rounded', getFriendshipStatusColor(friendship.status)]"
            style="background: var(--surface-color)"
          >
            {{ formatFriendshipStatus(friendship.status) }}
          </span>
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
  </CustomScrollbar>
</template>

<script setup lang="ts">
import { formatFriendshipStatus, getFriendshipStatusColor } from '../../utils/userHelpers';
import type { Friendship } from '../../models/types';
import CustomScrollbar from '../common/CustomScrollbar.vue';

interface Props {
  friends: Friendship[];
}

defineProps<Props>();

defineEmits<{
  select: [friendship: Friendship];
  'show-user': [user: any];
}>();
</script>
