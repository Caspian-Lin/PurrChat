<template>
  <section id="settings-account" class="settings-section">
    <h2 class="settings-section__title">账号</h2>

    <!-- 用户基本信息 -->
    <div class="flex items-center gap-4 mb-6">
      <img
        v-if="user?.avatar_url"
        :src="user.avatar_url"
        alt="avatar"
        class="w-16 h-16 rounded-[var(--radius-md,12px)] object-cover"
        referrerpolicy="no-referrer"
      />
      <div
        v-else
        class="w-16 h-16 rounded-[var(--radius-md,12px)] flex items-center justify-center font-bold text-white"
        style="background: var(--theme-gradient)"
      >
        {{ user?.username?.charAt(0) || 'U' }}
      </div>
      <div>
        <p class="text-text-primary font-medium">{{ user?.username }}</p>
        <p class="text-text-tertiary text-sm">UID: {{ user?.uid }}</p>
      </div>
    </div>

    <!-- 占位：个人信息 -->
    <div class="space-y-4">
      <h3 class="settings-section__subtitle">个人信息</h3>

      <div class="settings-field">
        <label class="settings-field__label">用户名</label>
        <div class="settings-field__value">{{ user?.username }}</div>
      </div>

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

      <div class="settings-field">
        <label class="settings-field__label">注册时间</label>
        <div class="settings-field__value">{{ formatDate(user?.created_at) }}</div>
      </div>

      <div class="p-4 rounded-[var(--radius-sm,8px)] text-text-tertiary text-sm" style="background: var(--surface-secondary-color)">
        修改用户名、手机号、邮箱等功能正在开发中...
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { User } from '../../../../models/types';

interface Props {
  user: User | null;
}

defineProps<Props>();

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
