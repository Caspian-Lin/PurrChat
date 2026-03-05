<template>
  <BaseModal
    :show="show"
    title="个人资料"
    class="max-w-md"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col items-center gap-6">
      <div class="w-28 h-28 roundrect overflow-hidden">
        <img
          v-if="user?.avatar_url"
          :src="user.avatar_url"
          alt="avatar"
          class="w-full h-full object-cover"
        />
        <div
          v-else
          class="w-full h-full flex items-center justify-center font-bold text-white text-4xl"
          style="background: var(--theme-gradient)"
        >
          {{ user?.username?.charAt(0) || 'U' }}
        </div>
      </div>
      <div class="w-full space-y-4">
        <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
          <span class="font-semibold" style="color: var(--text-secondary-color)">UID:</span>
          <span style="color: var(--text-color)">{{ user?.uid }}</span>
        </div>
        <div class="flex justify-between p-3 rounded-lg" style="background: var(--surface-color)">
          <span class="font-semibold" style="color: var(--text-secondary-color)">用户名:</span>
          <span style="color: var(--text-color)">{{ user?.username }}</span>
        </div>
        <div
          v-if="user?.email"
          class="flex justify-between p-3 rounded-lg"
          style="background: var(--surface-color)"
        >
          <span class="font-semibold" style="color: var(--text-secondary-color)">邮箱:</span>
          <span style="color: var(--text-color)">
            {{ user.email }}
            <span v-if="!user.email_verified" class="text-text-tertiary text-sm">(未验证)</span>
          </span>
        </div>
        <div
          v-if="user?.phone"
          class="flex justify-between p-3 rounded-lg"
          style="background: var(--surface-color)"
        >
          <span class="font-semibold" style="color: var(--text-secondary-color)">手机号:</span>
          <span style="color: var(--text-color)">
            {{ user.phone }}
            <span v-if="!user.phone_verified" class="text-text-tertiary text-sm">(未验证)</span>
          </span>
        </div>
      </div>
      <button
        class="w-full py-3 rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors font-semibold"
        @click="emit('logout')"
      >
        退出登录
      </button>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import BaseModal from '../common/BaseModal.vue';
import type { User } from '../../models/types';

interface Props {
  show: boolean;
  user: User | null;
}

defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  logout: [];
}>();
</script>
