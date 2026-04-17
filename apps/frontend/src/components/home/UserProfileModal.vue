<template>
  <BaseModal
    :show="show"
    title="用户信息"
    class="max-w-md"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col items-center gap-6">
      <!-- 头像区域 -->
      <div v-if="isCurrentUser" class="relative group cursor-pointer" @click="handleAvatarClick">
        <div class="w-28 h-28 roundrect overflow-hidden">
          <img
            v-if="displayAvatarUrl"
            :src="displayAvatarUrl"
            alt="avatar"
            class="w-full h-full object-cover"
            referrerpolicy="no-referrer"
            @error="(e) => console.error('[avatar] 头像加载失败:', displayAvatarUrl, e)"
          />
          <div
            v-else
            class="w-full h-full flex items-center justify-center font-bold text-white text-4xl"
            style="background: var(--theme-gradient)"
          >
            {{ user?.username?.charAt(0) || 'U' }}
          </div>
        </div>
        <!-- 上传中遮罩 -->
        <div
          v-if="uploading"
          class="absolute inset-0 roundrect bg-black/50 flex items-center justify-center"
        >
          <div
            class="w-8 h-8 border-2 border-white border-t-transparent rounded-full animate-spin"
          ></div>
        </div>
        <!-- hover 提示 -->
        <div
          v-else
          class="absolute inset-0 roundrect bg-black/0 group-hover:bg-black/40 transition-colors flex items-center justify-center"
        >
          <BsCamera
            class="text-white opacity-0 group-hover:opacity-100 transition-opacity"
            :size="24"
          />
        </div>
        <input
          ref="fileInputRef"
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
          class="hidden"
          @change="handleFileChange"
        />
      </div>
      <!-- 非当前用户：只读头像 -->
      <div v-else class="w-28 h-28 roundrect overflow-hidden">
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

      <!-- 上传错误提示 -->
      <div
        v-if="error"
        class="w-full px-4 py-2 bg-red-500/10 border border-red-500/30 rounded-lg text-red-500 text-sm text-center"
      >
        {{ error }}
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

      <!-- 好友状态操作按钮 -->
      <div class="w-full space-y-3">
        <!-- 陌生人：显示添加好友按钮 -->
        <button
          v-if="friendshipStatus === 'stranger'"
          class="w-full py-3 bg-accent-color text-white rounded-lg hover:opacity-80 transition-colors font-semibold"
          :disabled="loading"
          @click="handleSendFriendRequest"
        >
          {{ loading ? '发送中...' : '添加好友' }}
        </button>

        <!-- 已发送好友申请：显示已发送状态 -->
        <div
          v-else-if="friendshipStatus === 'sent'"
          class="w-full py-3 bg-yellow-500 text-white rounded-lg text-center font-semibold"
        >
          已发送好友申请
        </div>

        <!-- 待处理好友申请：显示接受和忽略按钮 -->
        <div v-else-if="friendshipStatus === 'pending'" class="flex gap-3">
          <button
            class="flex-1 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors font-semibold"
            :disabled="loading"
            @click="handleAcceptRequest"
          >
            {{ loading ? '处理中...' : '接受' }}
          </button>
          <button
            class="flex-1 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors font-semibold"
            :disabled="loading"
            @click="handleRejectRequest"
          >
            {{ loading ? '处理中...' : '忽略' }}
          </button>
        </div>

        <!-- 已是好友：显示开始聊天按钮 -->
        <button
          v-else-if="friendshipStatus === 'accepted'"
          class="w-full py-3 bg-accent-color text-white rounded-lg hover:opacity-80 transition-colors font-semibold"
          @click="emit('start-chat')"
        >
          开始聊天
        </button>

        <!-- 已拒绝：显示重新添加好友按钮 -->
        <button
          v-else-if="friendshipStatus === 'rejected'"
          class="w-full py-3 bg-accent-color text-white rounded-lg hover:opacity-80 transition-colors font-semibold"
          :disabled="loading"
          @click="handleSendFriendRequest"
        >
          {{ loading ? '发送中...' : '重新添加好友' }}
        </button>

        <!-- 当前用户：显示退出登录按钮 -->
        <button
          v-if="isCurrentUser"
          class="w-full py-3 rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors font-semibold"
          @click="emit('logout')"
        >
          退出登录
        </button>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { BsCamera } from 'vue-icons-plus/bs';
import BaseModal from '../common/BaseModal.vue';
import type { User, Friendship } from '../../models/types';
import { useAvatarUpload } from '../../composables/useAvatarUpload';
import { useAuthStore } from '../../stores/auth';

interface Props {
  show: boolean;
  user: User | null;
  isCurrentUser?: boolean;
  friendship?: Friendship | null;
  loading?: boolean;
  currentUserId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  logout: [];
  'send-friend-request': [];
  'accept-request': [];
  'reject-request': [];
  'start-chat': [];
}>();

const fileInputRef = ref<HTMLInputElement | null>(null);
const { uploading, error, previewUrl, uploadAvatar, clearError } = useAvatarUpload();
const authStore = useAuthStore();

// 显示的头像 URL：优先显示上传预览，否则显示 authStore 中最新的 avatar_url
const displayAvatarUrl = computed(() => {
  if (previewUrl.value) return previewUrl.value;
  return authStore.user?.avatar_url || props.user?.avatar_url || '';
});

// 弹窗关闭时清除状态
watch(
  () => props.show,
  (newVal) => {
    if (!newVal) {
      clearError();
    }
  }
);

function handleAvatarClick() {
  if (uploading.value) return;
  fileInputRef.value?.click();
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;

  await uploadAvatar(file);

  // 重置 input 以允许选择同一文件
  input.value = '';
}

// 计算好友状态
const friendshipStatus = computed(() => {
  if (!props.user || props.isCurrentUser) return 'none';

  if (!props.friendship) return 'stranger';

  const status = props.friendship.status;

  // 如果当前用户是发送方（当前登录用户是 user_id）
  if (props.friendship.user_id === props.currentUserId) {
    if (status === 'pending') return 'sent';
    if (status === 'rejected') return 'rejected';
    if (status === 'accepted') return 'accepted';
  }

  // 如果当前用户是接收方（当前登录用户是 friend_id）
  if (props.friendship.friend_id === props.currentUserId) {
    if (status === 'pending') return 'pending';
    if (status === 'accepted') return 'accepted';
  }

  return 'stranger';
});

// 处理发送好友请求
const handleSendFriendRequest = () => {
  emit('send-friend-request');
};

// 处理接受好友请求
const handleAcceptRequest = () => {
  emit('accept-request');
};

// 处理拒绝好友请求
const handleRejectRequest = () => {
  emit('reject-request');
};
</script>
