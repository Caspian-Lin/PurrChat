<template>
  <div
    class="flex flex-col size-full p-8 overflow-y-auto"
    style="background: var(--background-color)"
  >
    <div class="flex items-center justify-between mb-8">
      <h2 class="text-2xl font-bold" style="color: var(--text-color)">好友信息</h2>
      <n-button quaternary @click="$emit('close')"> 关闭 </n-button>
    </div>
    <div class="flex flex-col items-center gap-8">
      <div class="w-36 h-36 roundrect overflow-hidden">
        <img
          v-if="friendship.friend?.avatar_url"
          :src="friendship.friend.avatar_url"
          alt="avatar"
          class="w-full h-full object-cover"
        />
        <div
          v-else
          class="w-full h-full flex items-center justify-center font-bold text-white text-4xl"
          style="background: var(--theme-gradient)"
        >
          {{ friendship.friend?.username?.charAt(0) || '?' }}
        </div>
      </div>
      <div class="w-full max-w-md space-y-4">
        <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
          <span class="font-semibold" style="color: var(--text-secondary-color)">昵称:</span>
          <span style="color: var(--text-color)">{{ friendship.friend?.username }}</span>
        </div>
        <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
          <span class="font-semibold" style="color: var(--text-secondary-color)">UID:</span>
          <span style="color: var(--text-color)">{{ friendship.friend?.uid }}</span>
        </div>
        <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
          <span class="font-semibold" style="color: var(--text-secondary-color)">用户名:</span>
          <span style="color: var(--text-color)">{{ friendship.friend?.username }}</span>
        </div>
        <div
          v-if="friendship.friend?.email"
          class="flex justify-between p-3 rounded-lg"
          style="background: var(--surface-color)"
        >
          <span class="font-semibold" style="color: var(--text-secondary-color)">邮箱:</span>
          <span style="color: var(--text-color)">
            {{ friendship.friend.email }}
            <span v-if="!friendship.friend.email_verified" class="text-text-tertiary text-sm"
              >(未验证)</span
            >
          </span>
        </div>
        <div
          v-if="friendship.friend?.phone"
          class="flex justify-between p-3 rounded-lg"
          style="background: var(--surface-color)"
        >
          <span class="font-semibold" style="color: var(--text-secondary-color)">手机号:</span>
          <span style="color: var(--text-color)">
            {{ friendship.friend.phone }}
            <span v-if="!friendship.friend.phone_verified" class="text-text-tertiary text-sm"
              >(未验证)</span
            >
          </span>
        </div>
      </div>
      <Button type="primary" class="relative px-3 py-2" @click="$emit('start-chat')">
        发消息
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Friendship } from '../../models/types';

interface Props {
  friendship: Friendship;
}

defineProps<Props>();

defineEmits<{
  close: [];
  'start-chat': [];
}>();
</script>
