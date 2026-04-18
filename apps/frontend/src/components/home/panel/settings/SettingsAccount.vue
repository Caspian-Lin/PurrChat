<template>
  <section id="settings-account" class="settings-section">
    <h2 class="settings-section__title">账号</h2>

    <!-- 头像上传 -->
    <div class="space-y-2 mb-6">
      <h3 class="settings-section__subtitle">头像</h3>
      <div class="relative group cursor-pointer w-fit" @click="handleAvatarClick">
        <div class="w-20 h-20 rounded-[var(--radius-md)] overflow-hidden">
          <img
            v-if="displayAvatarUrl"
            :src="displayAvatarUrl"
            alt="avatar"
            class="w-full h-full object-cover"
            referrerpolicy="no-referrer"
          />
          <div
            v-else
            class="w-full h-full flex items-center justify-center font-bold text-white text-2xl"
            style="background: var(--theme-gradient)"
          >
            {{ user?.username?.charAt(0) || 'U' }}
          </div>
        </div>
        <!-- 上传中遮罩 -->
        <div
          v-if="avatarState.uploading.value"
          class="absolute inset-0 rounded-[var(--radius-md)] bg-black/50 flex items-center justify-center"
        >
          <div
            class="w-7 h-7 border-2 border-white border-t-transparent rounded-full animate-spin"
          ></div>
        </div>
        <!-- hover 提示 -->
        <div
          v-else
          class="absolute inset-0 rounded-[var(--radius-md)] bg-black/0 group-hover:bg-black/40 transition-colors flex items-center justify-center"
        >
          <BsCamera
            class="text-white opacity-0 group-hover:opacity-100 transition-opacity"
            :size="22"
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
      <p v-if="avatarState.error.value" class="text-xs" style="color: var(--color-error)">
        {{ avatarState.error.value }}
      </p>
    </div>

    <!-- 个人信息 -->
    <div class="">
      <h3 class="settings-section__subtitle">个人信息</h3>

      <!-- 用户名：内联编辑 -->
      <div class="settings-field">
        <label class="settings-field__label">用户名</label>
        <!-- 只读模式 -->
        <div
          v-if="!isEditingUsername"
          class="settings-field__value flex items-center justify-between"
        >
          <span>{{ user?.username }}</span>
          <button
            class="p-1 rounded-[var(--radius-xs)] transition-colors hover:bg-[var(--surface-tertiary-color)]"
            style="color: var(--text-tertiary-color)"
            title="修改用户名"
            @click="startEditUsername"
          >
            <BsPencil :size="14" />
          </button>
        </div>
        <!-- 编辑模式 -->
        <div v-else class="flex items-center gap-2">
          <BaseInput
            ref="usernameInputRef"
            v-model="editingUsername"
            placeholder="输入新用户名"
            class="flex-1"
            @keyup.enter="saveUsername"
            @keyup.escape="cancelEditUsername"
          />
          <button
            :disabled="savingUsername"
            class="px-3 py-2 text-sm text-white rounded-[var(--radius-sm)] transition-opacity disabled:opacity-50"
            :style="{ backgroundColor: 'var(--theme-primary)' }"
            @click="saveUsername"
          >
            保存
          </button>
          <button
            class="px-3 py-2 text-sm rounded-[var(--radius-sm)] transition-colors"
            style="color: var(--text-secondary-color); background: var(--surface-tertiary-color)"
            @click="cancelEditUsername"
          >
            取消
          </button>
        </div>
        <p v-if="usernameError" class="text-xs mt-1" style="color: var(--color-error)">
          {{ usernameError }}
        </p>
      </div>

      <!-- 邮箱：只读 -->
      <div class="settings-field">
        <label class="settings-field__label">邮箱</label>
        <div class="settings-field__value">
          {{ user?.email || '未设置' }}
          <span
            v-if="user?.email_verified"
            class="text-xs ml-2"
            style="color: var(--color-success)"
          >
            已验证
          </span>
        </div>
      </div>

      <!-- 手机号：只读 -->
      <div class="settings-field">
        <label class="settings-field__label">手机号</label>
        <div class="settings-field__value">
          {{ user?.phone || '未设置' }}
          <span
            v-if="user?.phone_verified"
            class="text-xs ml-2"
            style="color: var(--color-success)"
          >
            已验证
          </span>
        </div>
      </div>

      <!-- 注册时间：只读 -->
      <div class="settings-field">
        <label class="settings-field__label">注册时间</label>
        <div class="settings-field__value">{{ formatDate(user?.created_at) }}</div>
      </div>
    </div>

    <!-- 安全 -->
    <div class="space-y-3 mt-8">
      <h3 class="settings-section__subtitle">安全</h3>
      <button
        class="px-4 py-2.5 text-sm rounded-[var(--radius-sm)] transition-colors"
        style="color: var(--text-secondary-color); background: var(--surface-tertiary-color)"
        @click="showPasswordModal = true"
      >
        修改密码
      </button>
    </div>

    <!-- 修改密码弹窗 -->
    <ChangePasswordModal
      :show="showPasswordModal"
      @update:show="showPasswordModal = $event"
      @success="handlePasswordChanged"
    />
  </section>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue';
import { BsPencil, BsCamera } from 'vue-icons-plus/bs';
import BaseInput from '../../../common/BaseInput.vue';
import ChangePasswordModal from './ChangePasswordModal.vue';
import type { User } from '../../../../models/types';
import { api } from '../../../../models/api';
import { useAuthStore } from '../../../../stores/auth';
import { useMessage } from '../../../../composables/useMessage';
import { useAvatarUpload } from '../../../../composables/useAvatarUpload';

interface Props {
  user: User | null;
}

const props = defineProps<Props>();

const authStore = useAuthStore();
const { success, error: showError } = useMessage();

// ===== 头像上传 =====
const fileInputRef = ref<HTMLInputElement | null>(null);
const avatarState = useAvatarUpload();

const displayAvatarUrl = computed(() => {
  if (avatarState.previewUrl.value) return avatarState.previewUrl.value;
  return authStore.user?.avatar_url || props.user?.avatar_url || '';
});

function handleAvatarClick() {
  if (avatarState.uploading.value) return;
  avatarState.clearError();
  fileInputRef.value?.click();
}

async function handleFileChange(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;

  const result = await avatarState.uploadAvatar(file);
  if (result) {
    success('头像更新成功');
  } else {
    showError(avatarState.error.value || '头像上传失败');
  }
  // 重置 input 以允许选择相同文件
  (event.target as HTMLInputElement).value = '';
}

// ===== 用户名内联编辑 =====
const isEditingUsername = ref(false);
const editingUsername = ref('');
const savingUsername = ref(false);
const usernameError = ref('');
const usernameInputRef = ref<InstanceType<typeof BaseInput> | null>(null);

function startEditUsername() {
  editingUsername.value = props.user?.username || '';
  usernameError.value = '';
  isEditingUsername.value = true;
  nextTick(() => {
    usernameInputRef.value?.$el?.querySelector('input')?.focus();
  });
}

function cancelEditUsername() {
  isEditingUsername.value = false;
  editingUsername.value = '';
  usernameError.value = '';
}

async function saveUsername() {
  const trimmed = editingUsername.value.trim();
  if (!trimmed) {
    usernameError.value = '用户名不能为空';
    return;
  }
  if (trimmed.length < 3) {
    usernameError.value = '用户名至少 3 个字符';
    return;
  }
  if (trimmed.length > 20) {
    usernameError.value = '用户名最多 20 个字符';
    return;
  }
  if (trimmed === props.user?.username) {
    isEditingUsername.value = false;
    return;
  }

  savingUsername.value = true;
  usernameError.value = '';
  try {
    const resp = await api.updateProfile({ username: trimmed });
    if (resp.success && resp.data) {
      authStore.user = resp.data;
      localStorage.setItem('user', JSON.stringify(resp.data));
      success('用户名修改成功');
      isEditingUsername.value = false;
    } else {
      usernameError.value = resp.message || '修改失败';
    }
  } catch {
    usernameError.value = '修改失败，请重试';
  } finally {
    savingUsername.value = false;
  }
}

// ===== 密码修改 =====
const showPasswordModal = ref(false);

function handlePasswordChanged() {
  showPasswordModal.value = false;
  success('密码修改成功，下次登录时生效');
}

// ===== 工具函数 =====
function formatDate(dateStr?: string): string {
  if (!dateStr) return '—';
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}
</script>
