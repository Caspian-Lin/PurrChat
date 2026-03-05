<template>
  <BaseModal
    :show="show"
    title="用户操作"
    class="max-w-md"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col gap-6">
      <div class="flex items-center gap-4 p-4 rounded-lg" style="background: var(--surface-color)">
        <div class="w-12 h-12 roundrect overflow-hidden flex-shrink-0">
          <img
            v-if="user?.avatar_url"
            :src="user.avatar_url"
            alt="avatar"
            class="w-full h-full object-cover"
          />
          <div
            v-else
            class="w-full h-full flex items-center justify-center font-bold text-white"
            style="background: var(--theme-gradient)"
          >
            {{ user?.username?.charAt(0) || 'U' }}
          </div>
        </div>
        <div>
          <div class="font-semibold" style="color: var(--text-color)">
            {{ user?.username }}
          </div>
          <div class="text-sm" style="color: var(--text-secondary-color)">UID: {{ user?.uid }}</div>
        </div>
      </div>
      <div class="flex gap-4">
        <BaseButton class="flex-1" type="primary" block @click="$emit('send-friend-request')">
          发送好友请求
        </BaseButton>
        <BaseButton class="flex-1" block @click="$emit('start-chat')"> 开始聊天 </BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import BaseModal from '../common/BaseModal.vue';
import BaseButton from '../common/BaseButton.vue';
import type { User } from '../../models/types';

interface Props {
  show: boolean;
  user: User | null;
}

defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  'send-friend-request': [];
  'start-chat': [];
}>();
</script>
